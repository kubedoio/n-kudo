# n-kudo Helm Chart

A Helm chart for deploying the n-kudo control plane on Kubernetes.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.12+
- Persistent Volume provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

### Add the repository

```bash
# Clone the repository
git clone https://github.com/kubedoio/n-kudo.git
cd n-kudo/deployments/helm

# Install the chart
helm install nkudo ./nkudo
```

### Install with custom values

```bash
# Create a custom values file
cat > my-values.yaml <<EOF
replicaCount: 3

resources:
  limits:
    cpu: 2000m
    memory: 2Gi

postgresql:
  enabled: true
  auth:
    password: mysecretpassword

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: nkudo.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    enabled: true
    secretName: nkudo-tls
EOF

# Install with custom values
helm install nkudo ./nkudo -f my-values.yaml
```

### Production Installation

```bash
helm install nkudo ./nkudo -f ./nkudo/values-production.yaml
```

## Configuration

The following table lists the configurable parameters of the n-kudo chart and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Global Docker image registry | `""` |
| `global.imagePullSecrets` | Global Docker registry secret names | `[]` |
| `global.storageClass` | Global StorageClass for Persistent Volume(s) | `""` |
| `global.nodeSelector` | Global node labels for pod assignment | `{}` |
| `global.tolerations` | Global tolerations for pod assignment | `[]` |
| `global.affinity` | Global affinity rules for pod assignment | `{}` |

### Image Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.registry` | Image registry | `ghcr.io` |
| `image.repository` | Image repository | `kubedoio/n-kudo-control-plane` |
| `image.tag` | Image tag (immutable tags are recommended) | `""` (defaults to appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Specify docker-registry secret names | `[]` |

### Deployment Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `strategy.type` | Deployment strategy type | `RollingUpdate` |
| `podAnnotations` | Additional pod annotations | `{}` |
| `podLabels` | Additional pod labels | `{}` |
| `podSecurityContext` | Security context for the pod | See values.yaml |
| `containerSecurityContext` | Security context for the container | See values.yaml |
| `resources` | CPU/Memory resource requests/limits | See values.yaml |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `tolerations` | Tolerations for pod assignment | `[]` |
| `affinity` | Affinity rules for pod assignment | `{}` |
| `topologySpreadConstraints` | Topology spread constraints | `[]` |
| `priorityClassName` | Priority class name | `""` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes Service type | `ClusterIP` |
| `service.httpPort` | HTTP port | `8080` |
| `service.grpcPort` | gRPC port | `9090` |
| `service.metricsPort` | Metrics port | `9091` |
| `service.annotations` | Service annotations | `{}` |
| `headlessService.enabled` | Enable headless service | `false` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.tls.enabled` | Enable TLS | `false` |
| `ingress.tls.secretName` | TLS secret name | `""` |
| `ingress.hosts` | Ingress hosts configuration | `[{host: nkudo.local, paths: [{path: /, pathType: Prefix}]}]` |
| `ingress.grpc.enabled` | Enable gRPC ingress | `false` |
| `ingress.grpc.annotations` | gRPC ingress annotations | See values.yaml |

### Database Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `database.type` | Database type (postgresql, sqlite) | `postgresql` |
| `postgresql.enabled` | Enable PostgreSQL subchart | `true` |
| `postgresql.architecture` | PostgreSQL architecture | `standalone` |
| `postgresql.auth.username` | PostgreSQL username | `nkudo` |
| `postgresql.auth.database` | PostgreSQL database | `nkudo` |
| `postgresql.auth.password` | PostgreSQL password | auto-generated |
| `postgresql.primary.persistence.size` | PostgreSQL PVC size | `10Gi` |
| `database.external.host` | External PostgreSQL host | `""` |
| `database.external.port` | External PostgreSQL port | `5432` |
| `database.external.database` | External database name | `nkudo` |
| `database.external.user` | External database user | `nkudo` |
| `database.external.existingSecret` | Existing secret for DB credentials | `""` |

### Security Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `security.mtls.enabled` | Enable mTLS for edge communication | `true` |
| `security.mtls.caCert` | CA certificate (base64) | auto-generated |
| `security.mtls.caKey` | CA key (base64) | auto-generated |
| `security.mtls.existingCASecret` | Existing secret for CA | `""` |
| `security.adminKey` | Admin API key | auto-generated |
| `security.jwt.expiry` | JWT token expiry | `24h` |

### Persistence Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.storageClass` | Storage class name | `""` |
| `persistence.size` | PVC size | `5Gi` |
| `persistence.existingClaim` | Existing PVC name | `""` |

### Monitoring Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.enabled` | Enable Prometheus metrics | `false` |
| `monitoring.serviceMonitor.enabled` | Enable ServiceMonitor | `false` |
| `monitoring.serviceMonitor.namespace` | ServiceMonitor namespace | `""` |
| `monitoring.serviceMonitor.interval` | Scrape interval | `30s` |
| `monitoring.prometheusRule.enabled` | Enable PrometheusRule | `false` |
| `monitoring.grafana.enabled` | Enable Grafana dashboard | `false` |

### Autoscaling Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory utilization | `80` |

### Backup Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `backup.enabled` | Enable backup sidecar | `false` |
| `backup.schedule` | Backup schedule (cron) | `"0 2 * * *"` |
| `backup.retention` | Backup retention count | `7` |
| `backup.s3.enabled` | Enable S3 backup | `false` |
| `backup.s3.endpoint` | S3 endpoint | `""` |
| `backup.s3.bucket` | S3 bucket | `""` |

## Persistence

The n-kudo chart mounts a Persistent Volume at the `/data` path. The volume is created using dynamic volume provisioning. If you want to disable persistence, set `persistence.enabled` to `false`.

### Existing PersistentVolumeClaim

1. Create the PersistentVolume
2. Create the PersistentVolumeClaim
3. Install the chart

```bash
helm install nkudo ./nkudo --set persistence.existingClaim=PVC_NAME
```

## Security

### mTLS

The chart supports mTLS for secure edge communication. CA certificates are automatically generated if not provided:

```yaml
security:
  mtls:
    enabled: true
    existingCASecret: my-ca-secret
```

To retrieve the generated CA certificate:

```bash
kubectl get secret nkudo-ca -o jsonpath='{.data.ca\.crt}' | base64 -d
```

### Admin Key

The admin API key is auto-generated and stored in a secret:

```bash
kubectl get secret nkudo-security -o jsonpath='{.data.admin-key}' | base64 -d
```

## Upgrading

### To 0.2.0

Review the release notes and breaking changes before upgrading. Always backup your database before major upgrades:

```bash
# Backup first
helm get values nkudo > nkudo-values-backup.yaml

# Upgrade
helm upgrade nkudo ./nkudo -f nkudo-values-backup.yaml
```

### Database Migrations

Database migrations run automatically as init containers during deployment upgrades.

## Uninstallation

```bash
# Uninstall the release
helm uninstall nkudo

# Optional: Delete PVC (WARNING: This will delete all data!)
kubectl delete pvc -l app.kubernetes.io/name=nkudo
```

## Troubleshooting

### Check pod status

```bash
kubectl get pods -l app.kubernetes.io/name=nkudo
```

### View logs

```bash
kubectl logs -l app.kubernetes.io/name=nkudo -f
```

### Check events

```bash
kubectl get events --field-selector involvedObject.name=<pod-name>
```

### Common Issues

#### Pod stuck in Pending

Check resource constraints and PVC status:

```bash
kubectl describe pod <pod-name>
kubectl get pvc
```

#### Database connection errors

Verify PostgreSQL is running and credentials are correct:

```bash
kubectl get pods -l app.kubernetes.io/name=postgresql
kubectl logs -l app.kubernetes.io/name=nkudo | grep -i database
```

## Development

### Lint the chart

```bash
helm lint ./nkudo
```

### Template rendering

```bash
helm template nkudo ./nkudo --debug
```

### Run tests

```bash
helm test nkudo
```

## Contributing

Contributions are welcome! Please see the [main repository](https://github.com/kubedoio/n-kudo) for contribution guidelines.

## License

This chart is released under the same license as the n-kudo project. See the [LICENSE](https://github.com/kubedoio/n-kudo/blob/main/LICENSE) file for details.
