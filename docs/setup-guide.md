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

## Phase 2: Local Kubernetes Cluster ✅

### Commands Executed:
```bash
kind create cluster --config=cluster/kind-config.yaml
kubectl cluster-info --context kind-eks-local-cluster
kubectl get nodes
```

**Result**: 3-node cluster created successfully
- Control plane: eks-local-cluster-control-plane
- Workers: eks-local-cluster-worker, eks-local-cluster-worker2

## Phase 3: Argo CD GitOps ✅

### Commands Executed:
```bash
kubectl apply -k argocd/
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s
```

### Access Information:
- **URL**: http://localhost:8080
- **Username**: admin
- **Password**: `DO5UrMN5uga-RUjn`

### Port Forwarding:
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:80
```

## Phase 4: Istio Service Mesh ✅

### Commands Executed:
```bash
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm install istio-base istio/base -n istio-system
helm install istiod istio/istiod -n istio-system --wait
helm install istio-gateway istio/gateway -n istio-ingress -f istio/gateway-values.yaml
kubectl apply -f istio/basic-gateway.yaml
```

### Access Information:
- **HTTP Gateway**: http://localhost:30081
- **HTTPS Gateway**: https://localhost:30444
- **Istio Control Plane**: Running in istio-system namespace

## Phase 5: Observability Stack ✅

### Components Deployed:
- **Prometheus + Grafana**: Metrics and dashboards
- **Loki**: Log aggregation
- **Jaeger**: Distributed tracing
- **AlertManager**: Alert management

### Access Information (Port Forwarded):
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686
- **Loki**: http://localhost:3100

### Port Forwarding Commands (Active):
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:80
kubectl port-forward svc/prometheus-grafana -n observability 3000:80
kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n observability 9090:9090
kubectl port-forward svc/jaeger-query -n observability 16686:16686
kubectl port-forward svc/loki -n observability 3100:3100
```

## Phase 6: Go Microservices ✅

### Microservices Deployed:
- **user-service**: http://localhost:8081
  - Endpoints: `/users`, `/users/{id}`, `/health`, `/metrics`
  - Prometheus metrics enabled
  - 2 replicas with Istio sidecar injection

### Commands Executed:
```bash
# Build and deploy user-service
cd microservices/user-service
docker build -t user-service:latest .
kind load docker-image user-service:latest --name eks-local-cluster
kubectl apply -f k8s/
kubectl port-forward svc/user-service 8081:8080
```

### Test Commands:
```bash
curl http://localhost:8081/health
curl http://localhost:8081/users
curl http://localhost:8081/users/1
curl http://localhost:8081/metrics
```

## Phase 7: Security Stack ✅

### Components Deployed:

#### Kyverno Policy Enforcement
- **Policies Active**:
  - No privileged containers
  - Resource limits required
  - No `:latest` tags
  - Required labels enforcement
  - No host namespace sharing

#### Falco Runtime Security
- **Monitoring**: Suspicious activities, network connections, file access
- **Alerts**: Integrated with Loki for log aggregation

#### Trivy Container Scanning
- **Daily Scans**: CronJob for vulnerability scanning
- **Integration**: Kubernetes-native security scanning

#### Gitleaks Secret Detection
- **Repository Scanning**: Automated secret detection
- **Configuration**: Custom rules for common secrets

### Security Test:
```bash
# Test Kyverno policies (this should fail)
kubectl run test-pod --image=nginx:latest --restart=Never
# Should be blocked by disallow-latest-tag policy
```

## Phase 8: Complete Architecture ✅

### All Services Running:
- **Argo CD**: http://localhost:8080 (admin/DO5UrMN5uga-RUjn)
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686
- **Loki**: http://localhost:3100
- **User Service**: http://localhost:8081

### Security Policies Active:
- ✅ Admission control with Kyverno
- ✅ Runtime threat detection with Falco
- ✅ Container scanning with Trivy
- ✅ Secret detection with Gitleaks

### Monitoring Stack:
- ✅ Metrics collection (Prometheus)
- ✅ Log aggregation (Loki)
- ✅ Distributed tracing (Jaeger)
- ✅ Visualization (Grafana)

This completes a full production-grade local Kubernetes platform! 🎉
