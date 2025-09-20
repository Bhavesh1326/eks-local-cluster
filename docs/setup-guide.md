# Setup Guide - EKS Local Cluster

## Phase 1: Initial Setup and Git Repository

### Prerequisites Verification ✅
- Docker: v28.4.0 ✅
- kubectl: v1.32.2 ✅
- Helm: v3.19.0 ✅
- Kind: v0.30.0 ✅
- Git: v2.47.1 ✅

### Commands Executed:

1. **Initialize Git Repository**
   ```bash
   git init
   ```

2. **Create Directory Structure**
   ```powershell
   New-Item -ItemType Directory -Path cluster, argocd, istio, observability, security, docs -Force
   ```

3. **Create Kind Cluster Configuration**
   - File: `cluster/kind-config.yaml`
   - Features: 3-node cluster (1 control-plane, 2 workers)
   - Port mappings for Istio ingress gateway

## Next Steps

### Phase 2: Create Local Kubernetes Cluster
```bash
kind create cluster --config=cluster/kind-config.yaml
```

### Phase 3: Install Argo CD
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

This guide will be updated as we progress through each phase.
