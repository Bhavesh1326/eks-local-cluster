# EKS Local Cluster Case Study

This project demonstrates a secure, observable, and scalable local Kubernetes platform using Kind, with GitOps managed through Argo CD, Istio service mesh, and a comprehensive observability and security stack.

## Architecture Overview

- **Local Cluster**: Kind (Kubernetes in Docker)
- **GitOps**: Argo CD
- **Service Mesh**: Istio
- **Observability**: Prometheus, Grafana, Loki, Jaeger
- **Policy Enforcement**: Kyverno
- **Runtime Security**: Falco
- **Image Scanning**: Trivy
- **Secret Detection**: Gitleaks

## Prerequisites

- Docker
- kubectl
- Helm
- Kind
- Git

## Quick Start

1. Create the local cluster: `kind create cluster --config=cluster/kind-config.yaml`
2. Install Argo CD: `kubectl apply -k argocd/`
3. Deploy applications via GitOps

## Project Structure

```
├── cluster/                    # Kind cluster configuration
├── argocd/                     # Argo CD installation and apps
├── istio/                      # Istio service mesh configuration
├── observability/              # Monitoring and logging stack
├── security/                   # Security tools (Kyverno, Falco, Trivy)
├── microservices/              # Sample microservices
└── docs/                       # Documentation
```

## Documentation

See the `docs/` folder for detailed setup instructions and architecture decisions.
