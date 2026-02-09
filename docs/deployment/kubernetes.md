# Kubernetes Deployment Guide

Production deployment guide for n-kudo on Kubernetes.

## Prerequisites

- Kubernetes 1.28+
- kubectl configured
- Helm 3.12+ (optional, for Helm-based deployment)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Ingress                              │
│                    (TLS Termination)                         │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
   ┌────▼────┐              ┌─────▼────┐
   │ Frontend│              │  Backend │
   │ (Nginx) │              │(Control  │
   │         │              │ Plane)   │
   └────┬────┘              └─────┬────┘
        │                         │
        │    ┌────────────────┐   │
        └────┤   PostgreSQL   ├───┘
             │   (StatefulSet)│
             └────────────────┘
```

## Quick Start

### 1. Create Namespace

```bash
kubectl create namespace nkudo
kubectl config set-context --current --namespace=nkudo
```

### 2. Create Secrets

```bash
# Generate secrets
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export ADMIN_KEY=$(openssl rand -base64 32)

# Create secrets
kubectl create secret generic nkudo-db-secret \
  --from-literal=password="$POSTGRES_PASSWORD" \
  --from-literal=url="postgres://nkudo:$POSTGRES_PASSWORD@postgres:5432/nkudo?sslmode=disable"

kubectl create secret generic nkudo-admin-secret \
  --from-literal=key="$ADMIN_KEY"
```

### 3. Deploy PostgreSQL

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          value: "nkudo"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: password
        - name: POSTGRES_DB
          value: "nkudo"
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - nkudo
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - nkudo
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: postgres-data
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  clusterIP: None
EOF
```

### 4. Deploy Control Plane

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: nkudo-backend-config
data:
  CONTROL_PLANE_ADDR: ":8443"
  CA_COMMON_NAME: "n-kudo-agent-ca"
  REQUIRE_PERSISTENT_PKI: "true"
  HTTP_READ_TIMEOUT: "30s"
  HTTP_WRITE_TIMEOUT: "60s"
  HTTP_IDLE_TIMEOUT: "120s"
  DEFAULT_ENROLLMENT_TTL: "15m"
  AGENT_CERT_TTL: "24h"
  HEARTBEAT_INTERVAL: "15s"
  HEARTBEAT_OFFLINE_AFTER: "60s"
  PLAN_LEASE_TTL: "45s"
  MAX_PENDING_PLANS: "10"
  METRICS_ENABLED: "true"
  LOG_LEVEL: "info"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nkudo-backend
  labels:
    app: nkudo-backend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nkudo-backend
  template:
    metadata:
      labels:
        app: nkudo-backend
    spec:
      initContainers:
      - name: migrate
        image: ghcr.io/kubedoio/n-kudo-control-plane:latest
        command: ["/app/control-plane", "migrate", "-dir", "/app/db/migrations"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: url
      containers:
      - name: backend
        image: ghcr.io/kubedoio/n-kudo-control-plane:latest
        ports:
        - containerPort: 8443
          name: https
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: url
        - name: ADMIN_KEY
          valueFrom:
            secretKeyRef:
              name: nkudo-admin-secret
              key: key
        envFrom:
        - configMapRef:
            name: nkudo-backend-config
        volumeMounts:
        - name: pki
          mountPath: /app/pki
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8443
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 15
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8443
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
            - ALL
      volumes:
      - name: pki
        secret:
          secretName: nkudo-pki-secret
      securityContext:
        fsGroup: 1000
---
apiVersion: v1
kind: Service
metadata:
  name: nkudo-backend
  labels:
    app: nkudo-backend
spec:
  selector:
    app: nkudo-backend
  ports:
  - port: 8443
    targetPort: 8443
    name: https
  type: ClusterIP
EOF
```

### 5. Deploy Frontend

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nkudo-frontend
  labels:
    app: nkudo-frontend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nkudo-frontend
  template:
    metadata:
      labels:
        app: nkudo-frontend
    spec:
      containers:
      - name: frontend
        image: ghcr.io/kubedoio/n-kudo-frontend:latest
        ports:
        - containerPort: 80
          name: http
        env:
        - name: VITE_API_BASE_URL
          value: "https://nkudo-backend:8443"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 15
        readinessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 101
          capabilities:
            drop:
            - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: nkudo-frontend
  labels:
    app: nkudo-frontend
spec:
  selector:
    app: nkudo-frontend
  ports:
  - port: 80
    targetPort: 80
    name: http
  type: ClusterIP
EOF
```

### 6. Configure Ingress

```bash
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nkudo-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.nkudo.example.com
    - app.nkudo.example.com
    secretName: nkudo-tls
  rules:
  - host: api.nkudo.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nkudo-backend
            port:
              number: 8443
  - host: app.nkudo.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nkudo-frontend
            port:
              number: 80
EOF
```

## Complete Manifest

Save as `nkudo-deployment.yaml`:

```yaml
---
# Namespace
apiVersion: v1
kind: Namespace
metadata:
  name: nkudo
  labels:
    name: nkudo

---
# Secrets (replace with actual values)
apiVersion: v1
kind: Secret
metadata:
  name: nkudo-db-secret
  namespace: nkudo
type: Opaque
stringData:
  password: "CHANGE_ME"
  url: "postgres://nkudo:CHANGE_ME@postgres:5432/nkudo?sslmode=disable"

---
apiVersion: v1
kind: Secret
metadata:
  name: nkudo-admin-secret
  namespace: nkudo
type: Opaque
stringData:
  key: "CHANGE_ME"

---
# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: nkudo-config
  namespace: nkudo
data:
  CONTROL_PLANE_ADDR: ":8443"
  CA_COMMON_NAME: "n-kudo-agent-ca"
  REQUIRE_PERSISTENT_PKI: "true"
  HTTP_READ_TIMEOUT: "30s"
  HTTP_WRITE_TIMEOUT: "60s"
  DEFAULT_ENROLLMENT_TTL: "15m"
  AGENT_CERT_TTL: "24h"
  HEARTBEAT_INTERVAL: "15s"
  HEARTBEAT_OFFLINE_AFTER: "60s"
  PLAN_LEASE_TTL: "45s"
  MAX_PENDING_PLANS: "10"
  METRICS_ENABLED: "true"
  LOG_LEVEL: "info"

---
# PostgreSQL PVC
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data
  namespace: nkudo
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi

---
# PostgreSQL StatefulSet
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: nkudo
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          value: "nkudo"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: password
        - name: POSTGRES_DB
          value: "nkudo"
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: postgres-data

---
# PostgreSQL Service
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: nkudo
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
  clusterIP: None

---
# Backend Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nkudo-backend
  namespace: nkudo
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nkudo-backend
  template:
    metadata:
      labels:
        app: nkudo-backend
    spec:
      initContainers:
      - name: migrate
        image: ghcr.io/kubedoio/n-kudo-control-plane:latest
        command: ["/app/control-plane", "migrate", "-dir", "/app/db/migrations"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: url
      containers:
      - name: backend
        image: ghcr.io/kubedoio/n-kudo-control-plane:latest
        ports:
        - containerPort: 8443
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: nkudo-db-secret
              key: url
        - name: ADMIN_KEY
          valueFrom:
            secretKeyRef:
              name: nkudo-admin-secret
              key: key
        envFrom:
        - configMapRef:
            name: nkudo-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "2000m"

---
# Backend Service
apiVersion: v1
kind: Service
metadata:
  name: nkudo-backend
  namespace: nkudo
spec:
  selector:
    app: nkudo-backend
  ports:
  - port: 8443

---
# Frontend Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nkudo-frontend
  namespace: nkudo
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nkudo-frontend
  template:
    metadata:
      labels:
        app: nkudo-frontend
    spec:
      containers:
      - name: frontend
        image: ghcr.io/kubedoio/n-kudo-frontend:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"

---
# Frontend Service
apiVersion: v1
kind: Service
metadata:
  name: nkudo-frontend
  namespace: nkudo
spec:
  selector:
    app: nkudo-frontend
  ports:
  - port: 80
```

## Operations

### Scaling

```bash
# Scale backend
kubectl scale deployment nkudo-backend --replicas=3

# Scale frontend
kubectl scale deployment nkudo-frontend --replicas=3

# Horizontal Pod Autoscaler
kubectl autoscale deployment nkudo-backend --min=2 --max=10 --cpu-percent=70
```

### Monitoring

```bash
# Get pods
kubectl get pods -n nkudo

# Check logs
kubectl logs -f deployment/nkudo-backend -n nkudo
kubectl logs -f deployment/nkudo-frontend -n nkudo

# Describe resources
kubectl describe pod <pod-name> -n nkudo
```

### Backup

```bash
# Database backup
kubectl exec -it postgres-0 -n nkudo -- pg_dump -U nkudo nkudo > backup.sql

# Copy to local
kubectl cp nkudo/postgres-0:/var/lib/postgresql/data ./postgres-backup
```

### Updates

```bash
# Rolling update
kubectl set image deployment/nkudo-backend \
  backend=ghcr.io/kubedoio/n-kudo-control-plane:v1.1.0 -n nkudo

# Rollback
kubectl rollout undo deployment/nkudo-backend -n nkudo
```

## Troubleshooting

```bash
# Check events
kubectl get events -n nkudo --sort-by='.lastTimestamp'

# Debug pod
kubectl run debug --rm -it --image=alpine --restart=Never -n nkudo -- sh

# Port forward for local testing
kubectl port-forward svc/nkudo-backend 8443:8443 -n nkudo
```
