# Docker Compose Deployment Guide

Production deployment guide for n-kudo using Docker Compose.

## Prerequisites

### System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| Docker | 24.0+ | 25.0+ |
| Docker Compose | 2.20+ | 2.23+ |
| RAM | 4 GB | 8 GB |
| CPU | 2 cores | 4 cores |
| Disk | 20 GB | 50 GB SSD |
| Network | 100 Mbps | 1 Gbps |

### Supported Operating Systems

- Ubuntu 22.04 LTS or newer
- Debian 12 or newer
- RHEL 9 / Rocky Linux 9
- Amazon Linux 2023

### Required Ports

| Port | Service | Description |
|------|---------|-------------|
| 8443 | Backend | Control Plane API (HTTPS) |
| 3000 | Frontend | Web Dashboard |
| 5432 | PostgreSQL | Database (optional external access) |

## Quick Start

### 1. Install Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# Verify installation
docker --version
docker compose version
```

### 2. Clone and Configure

```bash
# Clone repository
git clone https://github.com/kubedoio/n-kudo.git
cd n-kudo

# Copy production environment file
cp .env.production.example .env.production

# Edit with your settings (see Configuration section)
nano .env.production
```

### 3. Create Data Directory

```bash
# Create data directory for persistent storage
sudo mkdir -p /opt/nkudo/data/{postgres,pki}
sudo chown -R 1000:1000 /opt/nkudo/data
```

### 4. Generate Secrets

```bash
# Generate strong passwords and keys
echo "POSTGRES_PASSWORD=$(openssl rand -base64 32)" >> .env.production
echo "ADMIN_KEY=$(openssl rand -base64 32)" >> .env.production

# Update DATABASE_URL with generated password
sed -i "s/CHANGE_ME_TO_A_STRONG_RANDOM_PASSWORD_32_CHARS_MIN/$(grep POSTGRES_PASSWORD .env.production | cut -d= -f2)/g" .env.production
```

### 5. Start Services

```bash
# Start all services
docker compose -f docker-compose.yml --env-file .env.production up -d

# Check status
docker compose ps
docker compose logs -f
```

### 6. Verify Deployment

```bash
# Check health endpoints
curl -k https://localhost:8443/healthz
curl http://localhost:3000/health

# View logs
docker compose logs -f backend
docker compose logs -f postgres
```

## Configuration

### Required Settings

Edit `.env.production` and set these required values:

```bash
# Database
POSTGRES_PASSWORD=<strong_random_password>
DATABASE_URL=postgres://nkudo:<password>@postgres:5432/nkudo?sslmode=disable

# Security
ADMIN_KEY=<strong_random_key>

# TLS (for production)
REQUIRE_PERSISTENT_PKI=true
CA_CERT_FILE=/app/pki/ca.crt
CA_KEY_FILE=/app/pki/ca.key
SERVER_CERT_FILE=/app/pki/server.crt
SERVER_KEY_FILE=/app/pki/server.key
```

### TLS Certificate Setup

#### Option A: Using Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot

# Obtain certificates
sudo certbot certonly --standalone -d your-domain.com

# Copy to data directory
sudo mkdir -p /opt/nkudo/data/pki
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem /opt/nkudo/data/pki/server.crt
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem /opt/nkudo/data/pki/server.key
sudo chown -R 1000:1000 /opt/nkudo/data/pki
```

#### Option B: Using Custom Certificates

```bash
# Copy your certificates
sudo mkdir -p /opt/nkudo/data/pki
sudo cp /path/to/your/ca.crt /opt/nkudo/data/pki/
sudo cp /path/to/your/ca.key /opt/nkudo/data/pki/
sudo cp /path/to/your/server.crt /opt/nkudo/data/pki/
sudo cp /path/to/your/server.key /opt/nkudo/data/pki/
sudo chown -R 1000:1000 /opt/nkudo/data/pki
sudo chmod 600 /opt/nkudo/data/pki/*.key
```

#### Option C: Self-Signed (Development Only)

```bash
# Generate self-signed certificates
sudo mkdir -p /opt/nkudo/data/pki
cd /opt/nkudo/data/pki

# Generate CA
openssl req -x509 -sha256 -days 3650 -newkey rsa:4096 \
  -keyout ca.key -out ca.crt \
  -subj "/C=US/O=n-kudo/CN=n-kudo-ca"

# Generate server cert
openssl req -new -newkey rsa:4096 -keyout server.key -out server.csr \
  -subj "/C=US/O=n-kudo/CN=your-domain.com" -nodes

openssl x509 -req -sha256 -days 365 -in server.csr \
  -CA ca.crt -CAkey ca.key -out server.crt -CAcreateserial

sudo chown -R 1000:1000 /opt/nkudo/data/pki
```

## Production Checklist

### Security

- [ ] Change all default passwords and keys
- [ ] Enable TLS with valid certificates (not self-signed)
- [ ] Configure firewall rules (allow only necessary ports)
- [ ] Disable password authentication for SSH
- [ ] Enable automatic security updates
- [ ] Set up fail2ban for brute force protection
- [ ] Review and set appropriate CORS origins
- [ ] Enable audit logging

### Database

- [ ] Use strong PostgreSQL password
- [ ] Enable PostgreSQL SSL (if external access needed)
- [ ] Configure automated backups
- [ ] Set up database monitoring
- [ ] Test backup restoration procedure

### Monitoring

- [ ] Configure log aggregation (ELK, Loki, etc.)
- [ ] Set up metrics collection (Prometheus)
- [ ] Create alerting rules
- [ ] Set up health check monitoring
- [ ] Configure uptime monitoring

### Backups

- [ ] Automated daily database backups
- [ ] PKI certificates backed up securely
- [ ] Backup encryption enabled
- [ ] Offsite backup storage configured
- [ ] Regular backup restoration tests

### High Availability (Optional)

- [ ] Database replication configured
- [ ] Load balancer set up
- [ ] Multiple control plane instances
- [ ] Health checks for failover

## Maintenance

### Updates

```bash
# Pull latest images
docker compose pull

# Restart with new images
docker compose up -d

# Verify after update
docker compose ps
docker compose logs -f backend
```

### Backup

```bash
# Backup database
docker exec nkudo-postgres pg_dump -U nkudo nkudo > backup_$(date +%Y%m%d).sql

# Backup PKI
tar czf pki_backup_$(date +%Y%m%d).tar.gz /opt/nkudo/data/pki

# Automated backup script
#!/bin/bash
BACKUP_DIR=/opt/nkudo/backups
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

# Database backup
docker exec nkudo-postgres pg_dump -U nkudo nkudo | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# PKI backup
tar czf $BACKUP_DIR/pki_$DATE.tar.gz -C /opt/nkudo/data pki

# Cleanup old backups (keep 7 days)
find $BACKUP_DIR -type f -mtime +7 -delete
```

### Log Rotation

Logs are automatically rotated with the following settings:
- Maximum file size: 10 MB
- Maximum files: 3
- Total max log size per service: 30 MB

To view logs:
```bash
# All services
docker compose logs

# Specific service
docker compose logs backend

# Follow logs
docker compose logs -f backend

# Last 100 lines
docker compose logs --tail 100 backend
```

## Troubleshooting

### Services Won't Start

```bash
# Check for port conflicts
sudo netstat -tlnp | grep -E '8443|3000|5432'

# Check logs
docker compose logs backend
docker compose logs postgres

# Verify environment file
 docker compose config
```

### Database Connection Issues

```bash
# Test database connection
docker exec nkudo-backend wget -qO- --post-data="" postgres:5432

# Check postgres logs
docker compose logs postgres

# Verify credentials
docker exec -it nkudo-postgres psql -U nkudo -c "\conninfo"
```

### TLS/Certificate Issues

```bash
# Verify certificate
docker exec nkudo-backend openssl x509 -in /app/pki/server.crt -text -noout

# Test TLS connection
docker exec nkudo-backend wget --no-check-certificate https://localhost:8443/healthz

# Check certificate expiry
docker exec nkudo-backend openssl x509 -in /app/pki/server.crt -noout -dates
```

### Resource Limits

```bash
# Check resource usage
docker stats

# Adjust limits in docker-compose.yml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
```

## Uninstallation

```bash
# Stop and remove containers
docker compose down

# Remove volumes (WARNING: deletes all data)
docker compose down -v

# Remove data directory
sudo rm -rf /opt/nkudo/data
```

## Support

- Documentation: https://github.com/kubedoio/n-kudo/tree/main/docs
- Issues: https://github.com/kubedoio/n-kudo/issues
- Runbooks: See `docs/runbooks/` for troubleshooting guides
