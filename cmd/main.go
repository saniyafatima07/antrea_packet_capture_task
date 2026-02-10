package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"context"
	"path/filepath"
	"strconv"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/watch"
	 v1 "k8s.io/api/core/v1"
	 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Capture struct {
	cmd  *exec.Cmd
	pod  string
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
		stopCapture(uid, cap)
	}
}

func getClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func watchPods(client *kubernetes.Clientset, nodeName string) {
	for {
		watcher, err := client.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}

		for event := range watcher.ResultChan() {
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				continue
			}
			handlePod(pod)
		}
	}
}

func handlePod(pod *v1.Pod) {
	uid := string(pod.UID)
	val, exists := pod.Annotations["tcpdump.antrea.io"]

	if exists {
		if _, running := captures[uid]; running {
			return
		}
		startCapture(pod, val)
		return
	}

	if !exists {
		if cap, running := captures[uid]; running {
			stopCapture(uid, cap)
		}
	}
}

func startCapture(pod *v1.Pod, n string) {
	file := fmt.Sprintf("/outputs/capture-%s.pcap", pod.Name)

	files, err := strconv.Atoi(n)
	if err != nil {
		files = 1 
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	cmd := exec.CommandContext(ctx, "tcpdump", "-i", "any", "-C", "1", "-W", strconv.Itoa(files), "-w", file)
	err = cmd.Start()
    if err != nil {
        fmt.Println("Failed to start tcpdump for pod:", pod.Name, "error:", err)
        return
    }

	captures[string(pod.UID)] = &Capture{
        cmd:    cmd,
        pod:    pod.Name,
        cancel: cancel,
    }

	fmt.Println("Started capture for pod:", pod.Name)
}

func stopCapture(uid string, cap *Capture) {
    fmt.Println("Stopping capture for pod:", cap.pod)

    cap.cancel()
    cap.cmd.Wait()

    files, _ := filepath.Glob(fmt.Sprintf("/outputs/capture-%s.pcap*", cap.pod))
    for _, f := range files {
        os.Remove(f)
    }

    delete(captures, uid)
}