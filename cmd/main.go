package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Capture struct {
	cmd    *exec.Cmd
	pod    *v1.Pod
	cancel context.CancelFunc
}

var captures = make(map[string]*Capture)

func main() {
	node := os.Getenv("NODE_NAME")
	if node == "" {
		panic("NODE_NAME not set")
	}

	client, err := getClient()
	if err != nil {
		panic(err)
	}

	go watchPods(client, node)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch

	fmt.Println("Shutting down controller")

	for uid, cap := range captures {
		stopCapture(uid, cap, false)
	}
}

func getClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func watchPods(client *kubernetes.Clientset, node string) {
	watcher, err := client.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*v1.Pod)
		if !ok {
			continue
		}

		switch event.Type {
		case watch.Added, watch.Modified:
			handlePod(pod, node)
		case watch.Deleted:
			handlePodDelete(pod, node)
		}
	}
}

func handlePod(pod *v1.Pod, node string) {
	if pod.Spec.NodeName != node {
		return
	}

	uid := string(pod.UID)
	val, exists := pod.Annotations["tcpdump.antrea.io"]

	if exists {
		if _, running := captures[uid]; !running {
			startCapture(pod, val)
		}
	} else {
		if cap, running := captures[uid]; running {
			stopCapture(uid, cap, true)
		} else {
			files, _ := filepath.Glob(
				fmt.Sprintf("/outputs/capture-%s.pcap*", pod.Name),
			)
			for _, f := range files {
				os.Remove(f)
			}
			fmt.Println("Deleted capture files for pod:", pod.Name)
		}
	}
}

func handlePodDelete(pod *v1.Pod, node string) {
	if pod.Spec.NodeName != node {
		return
	}

	if cap, running := captures[string(pod.UID)]; running {
		stopCapture(string(pod.UID), cap, true)
	}
}

func startCapture(pod *v1.Pod, n string) {
	duration, err := strconv.Atoi(n)
	if err != nil || duration <= 0 {
		duration = 10
	}

	file := fmt.Sprintf("/outputs/capture-%s.pcap", pod.Name)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(
		ctx,
		"tcpdump",
		"-i", "any",
		"-U",
		"-w", file,
	)

	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start tcpdump:", err)
		return
	}

	captures[string(pod.UID)] = &Capture{
		cmd:    cmd,
		pod:    pod,
		cancel: cancel,
	}

	fmt.Println("Started capture for pod:", pod.Name)

	go func(uid string) {
		time.Sleep(time.Duration(duration) * time.Second)
		if cap, running := captures[uid]; running {
			stopCapture(uid, cap, false)
		}
	}(string(pod.UID))
}

func stopCapture(uid string, cap *Capture, deleteFiles bool) {
	fmt.Println("Stopping capture for pod:", cap.pod.Name)

	cap.cancel()

	if cap.cmd.Process != nil {
		cap.cmd.Process.Signal(syscall.SIGTERM)
	}

	cap.cmd.Wait()

	if deleteFiles {
		files, _ := filepath.Glob(
			fmt.Sprintf("/outputs/capture-%s.pcap*", cap.pod.Name),
		)
		for _, f := range files {
			os.Remove(f)
		}
		fmt.Println("Deleted capture files for pod:", cap.pod.Name)
	}

	delete(captures, uid)
}