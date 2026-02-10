# Antrea - Packet Capture

This repository contains a simple implementation of a Kubernetes packet capture controller, inspired by Antrea's PacketCapture feature. The controller runs as a DaemonSet and performs tcpdump captures on Pods on-demand using annotations.

---

## Features

- Runs as a **DaemonSet** on every node.
- Watches **Pods on the same node**.
- Captures network traffic **only when annotation is present**: tcpdump.antrea.io: "<N>"
  where `N` is the capture duration in seconds.
- Automatically **stops capture** and cleans up pcap files when annotation is removed.
- Containerized using **Ubuntu 24.04, Go and tcpdump**.

---

## File Structure

├── cmd/  
│   └── main.go  
├── manifests/  
│   ├── daemonset.yaml  
│   ├── packet-capture-rbac.yaml  
│   └── test-pod.yaml  
├── outputs/  
│   ├── capture-files.txt        
│   ├── capture-output.txt       
│   ├── capture-output.pcap  
│   ├── pod-describe.txt         
│   └── pods.txt                 
├── config.yaml   
├── Dockerfile  
├── go.mod  
├── go.sum  
└── README.md  

---

## Build and Deploy

1. **Build the controller image**
```bash
docker build -t packet-capture:dev .
kind load docker-image packet-capture:dev --name <cluster-name>
```

2. **Deploy RBAC**
```
kubectl apply -f manifests/packet-capture-rbac.yaml
```

3. **Deploy DaemonSet**
```
kubectl apply -f manifests/daemonset.yaml
```

4. **Deploy test Pod**
```
kubectl apply -f manifests/test-pod.yaml
```

## Verification

1. **Annotate the test Pod to start capture**
```bash
kubectl annotate pod test-ping tcpdump.antrea.io="5"
```

2. **Check the node where the test Pod is running**
```bash
kubectl get pod test-ping -o wide
```

3. **Get the packet-capture Pod running on the same node**
```bash
kubectl get pods -n kube-system -o wide | grep packet-capture
```

4. **Verify that the capture file is created**
```bash
kubectl exec -n kube-system <packet-capture-pod> -- \
sh -c 'ls -lh /outputs/capture-*' > outputs/capture-files.txt
```

5. **Copy the pcap file to the local machine**
```bash
kubectl cp kube-system/<packet-capture-pod>:/outputs/capture-test-ping.pcap \
outputs/capture-output.pcap
```

6. **Generate human-readable output from the pcap**
```bash
tcpdump -r outputs/capture-output.pcap > outputs/capture-output.txt
```

7. **Remove the annotation to stop capture**
```bash
kubectl annotate pod test-ping tcpdump.antrea.io-
```

8. **Verify that the pcap files are deleted**
```bash
kubectl exec -n kube-system <packet-capture-pod> -- ls -lh /outputs
```