# Antrea - Packet Capture

This repository contains a simple implementation of a Kubernetes packet capture controller, inspired by Antrea's PacketCapture feature. The controller runs as a DaemonSet and performs tcpdump captures on Pods on-demand using annotations.

---

## Features

- Runs as a **DaemonSet** on every node.
- Watches **Pods on the same node**.
- Captures network traffic **only when annotation is present**: tcpdump.antrea.io: "<N>"
where `N` is the max number of capture files.
- Automatically **stops capture** and cleans up pcap files when annotation is removed.
- Containerized using **Ubuntu 24.04 + Go + tcpdump**.

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
kind load docker-image packet-capture:dev
```

2. **Deploy RBAC**
```
kubectl apply -f manifests/packet-capture-rbac.yaml
```

3. **Deploy DaemonSet**
```
kubectl apply -f manifests/packet-capture-daemonset.yaml
```

4. **Deploy test Pod**
```
kubectl apply -f manifests/test-pod.yaml
```

## Verification

1. **Check Pod node**
```bash
kubectl get pod test-ping -o wide
```

2. **Exec into the DaemonSet pod on the same node**
```
kubectl exec -n kube-system <packet-capture-pod> -- ls -l /outputs
```

3. **Copy pcap to local**
```
kubectl cp kube-system/<packet-capture-pod>:/outputs/test.pcap ./outputs/capture-output.txt
```

4. **Read pcap output**
```
tcpdump -r ./capture-output.pcap
```

4. **Remove annotation**
```
kubectl annotate pod test-ping tcpdump.antrea.io-
```