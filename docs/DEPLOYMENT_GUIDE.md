# StableRisk Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying StableRisk to production. It covers prerequisites, configuration, deployment procedures, and post-deployment verification.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Pre-Deployment Planning](#pre-deployment-planning)
3. [Environment Setup](#environment-setup)
4. [Build and Push Images](#build-and-push-images)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [TLS Configuration](#tls-configuration)
7. [Initial Configuration](#initial-configuration)
8. [Post-Deployment Verification](#post-deployment-verification)
9. [Monitoring Setup](#monitoring-setup)
10. [Backup Configuration](#backup-configuration)
11. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Infrastructure Requirements

- **Kubernetes Cluster**: v1.25 or higher
  - Minimum 3 nodes
  - 8 CPU cores and 32GB RAM total
  - 100GB persistent storage
- **kubectl**: Configured and authenticated
- **Docker Registry**: For storing images (e.g., GitHub Container Registry, Docker Hub, AWS ECR)
- **Domain Name**: For TLS and ingress (e.g., stablerisk.yourdomain.com)
- **TLS Certificates**: Let's Encrypt (automated) or custom certificates

### Software Requirements

- Docker 20.10+
- kubectl 1.25+
- helm 3.10+ (for monitoring stack)
- git
- openssl

### Access Requirements

- **TronGrid API Key**: Obtain from https://www.trongrid.io/
- **Container Registry Access**: Push/pull permissions
- **DNS Management**: Ability to create A records
- **Kubernetes Cluster Admin**: For creating namespaces and secrets

### Compliance Requirements

- **ISO27001**: Documented security controls
- **PCI-DSS**: Encryption, access control, audit logging

---

## Pre-Deployment Planning

### 1. Resource Sizing

Estimate resources based on expected transaction volume:

| Transaction Volume | Nodes | Total CPU | Total RAM | Storage |
|-------------------|-------|-----------|-----------|---------|
| < 10k tx/day | 3 | 8 cores | 32GB | 100GB |
| 10k-100k tx/day | 5 | 16 cores | 64GB | 500GB |
| > 100k tx/day | 10+ | 32+ cores | 128GB+ | 1TB+ |

### 2. Network Planning

- **Ingress IP**: Reserve static IP for load balancer
- **DNS**: Plan DNS records
  - Primary: stablerisk.yourdomain.com
  - API: api.stablerisk.yourdomain.com (optional)
- **Firewall Rules**:
  - Allow inbound: 80/tcp, 443/tcp
  - Allow outbound: 443/tcp (for TronGrid API)

### 3. Security Planning

- [ ] Generate strong secrets (32+ characters)
- [ ] Plan RBAC roles and users
- [ ] Document access control policies
- [ ] Prepare TLS certificate strategy
- [ ] Review compliance requirements

### 4. Backup Planning

- [ ] Choose backup storage location (S3, GCS, etc.)
- [ ] Configure backup retention (default: 30 days)
- [ ] Test backup/restore procedures
- [ ] Document recovery procedures

---

## Environment Setup

### 1. Clone Repository

```bash
git clone https://github.com/mikedewar/stablerisk.git
cd stablerisk
```

### 2. Generate Secrets

Create a secure directory for secrets:

```bash
mkdir -p secrets
cd secrets

# Generate random passwords and keys
DATABASE_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)
HMAC_KEY=$(openssl rand -base64 32)

# Save to file (DO NOT commit to git!)
cat > .env.secrets << EOF
DATABASE_PASSWORD=${DATABASE_PASSWORD}
JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}
HMAC_KEY=${HMAC_KEY}
TRONGRID_API_KEY=your-trongrid-api-key-here
EOF

echo "Secrets generated in secrets/.env.secrets"
echo "⚠️  KEEP THIS FILE SECURE! Do not commit to version control."
```

### 3. Configure Domain

Update configuration files with your domain:

```bash
export DOMAIN="stablerisk.yourdomain.com"

# Update Kubernetes ConfigMap
sed -i "s/stablerisk.yourdomain.com/${DOMAIN}/g" \
  deployments/kubernetes/configmap.yaml

# Update Nginx configuration
sed -i "s/stablerisk.yourdomain.com/${DOMAIN}/g" \
  deployments/nginx/nginx-production.conf

# Update Ingress
sed -i "s/stablerisk.yourdomain.com/${DOMAIN}/g" \
  deployments/kubernetes/ingress.yaml
```

---

## Build and Push Images

### 1. Configure Container Registry

```bash
# Example: GitHub Container Registry
export REGISTRY="ghcr.io"
export REGISTRY_USER="yourusername"
export IMAGE_PREFIX="${REGISTRY}/${REGISTRY_USER}/stablerisk"

# Login to registry
echo $GITHUB_TOKEN | docker login ghcr.io -u ${REGISTRY_USER} --password-stdin
```

### 2. Build Images

```bash
# Build API image
docker build -t ${IMAGE_PREFIX}/api:latest -f Dockerfile.api .

# Build Monitor image
docker build -t ${IMAGE_PREFIX}/monitor:latest -f Dockerfile.monitor .

# Build Raphtory image
docker build -t ${IMAGE_PREFIX}/raphtory:latest -f raphtory-service/Dockerfile raphtory-service/

# Build Web image
docker build -t ${IMAGE_PREFIX}/web:latest \
  --build-arg PUBLIC_API_URL=https://${DOMAIN}/api/v1 \
  --build-arg PUBLIC_WS_URL=wss://${DOMAIN}/api/v1/ws \
  -f web/Dockerfile web/
```

### 3. Push Images

```bash
docker push ${IMAGE_PREFIX}/api:latest
docker push ${IMAGE_PREFIX}/monitor:latest
docker push ${IMAGE_PREFIX}/raphtory:latest
docker push ${IMAGE_PREFIX}/web:latest
```

### 4. Update Kubernetes Manifests

Update image references in deployment files:

```bash
# Update image prefix in all deployments
find deployments/kubernetes -name "*.yaml" -type f -exec \
  sed -i "s|stablerisk/|${IMAGE_PREFIX}/|g" {} \;
```

---

## Kubernetes Deployment

### Step 1: Create Namespace

```bash
kubectl apply -f deployments/kubernetes/namespace.yaml
```

**Verify:**
```bash
kubectl get namespace stablerisk
```

### Step 2: Create ConfigMap

```bash
kubectl apply -f deployments/kubernetes/configmap.yaml
```

**Verify:**
```bash
kubectl describe configmap stablerisk-config -n stablerisk
```

### Step 3: Create Secrets

```bash
# Load secrets from file
source secrets/.env.secrets

# Create Kubernetes secret
kubectl create secret generic stablerisk-secrets -n stablerisk \
  --from-literal=DATABASE_PASSWORD="${DATABASE_PASSWORD}" \
  --from-literal=JWT_SECRET="${JWT_SECRET}" \
  --from-literal=ENCRYPTION_KEY="${ENCRYPTION_KEY}" \
  --from-literal=HMAC_KEY="${HMAC_KEY}" \
  --from-literal=TRONGRID_API_KEY="${TRONGRID_API_KEY}"
```

**Verify (values should be opaque):**
```bash
kubectl get secret stablerisk-secrets -n stablerisk
```

### Step 4: Deploy PostgreSQL

```bash
kubectl apply -f deployments/kubernetes/postgres.yaml

# Wait for PostgreSQL to be ready (may take 2-3 minutes)
kubectl wait --for=condition=ready pod -l app=postgres -n stablerisk --timeout=300s
```

**Verify:**
```bash
kubectl get statefulset postgres -n stablerisk
kubectl logs -n stablerisk postgres-0 --tail=20

# Test connection
kubectl exec -n stablerisk postgres-0 -- pg_isready -U stablerisk
```

### Step 5: Deploy Raphtory

```bash
kubectl apply -f deployments/kubernetes/raphtory.yaml

# Wait for Raphtory to be ready
kubectl wait --for=condition=ready pod -l app=raphtory -n stablerisk --timeout=300s
```

**Verify:**
```bash
kubectl get deployment raphtory -n stablerisk
kubectl logs -n stablerisk -l app=raphtory --tail=20

# Test health endpoint
kubectl exec -n stablerisk -l app=raphtory -- curl -f http://localhost:8000/health
```

### Step 6: Deploy Monitor

```bash
kubectl apply -f deployments/kubernetes/monitor.yaml

# Wait for Monitor to be ready
kubectl wait --for=condition=ready pod -l app=monitor -n stablerisk --timeout=300s
```

**Verify:**
```bash
kubectl get deployment monitor -n stablerisk
kubectl logs -n stablerisk -l app=monitor --tail=50
```

**Expected log output:**
```
{"level":"info","msg":"Starting StableRisk Monitor"}
{"level":"info","msg":"Connected to TronGrid WebSocket"}
{"level":"info","msg":"Transaction stored","tx_hash":"..."}
```

### Step 7: Deploy API

```bash
kubectl apply -f deployments/kubernetes/api.yaml

# Wait for API to be ready
kubectl wait --for=condition=ready pod -l app=api -n stablerisk --timeout=300s
```

**Verify:**
```bash
kubectl get deployment api -n stablerisk
kubectl get hpa api-hpa -n stablerisk

# Test health endpoint
kubectl exec -n stablerisk -l app=api -- curl -f http://localhost:8080/health
```

### Step 8: Deploy Web Dashboard

```bash
kubectl apply -f deployments/kubernetes/web.yaml

# Wait for Web to be ready
kubectl wait --for=condition=ready pod -l app=web -n stablerisk --timeout=300s
```

**Verify:**
```bash
kubectl get deployment web -n stablerisk
kubectl logs -n stablerisk -l app=web --tail=20
```

### Step 9: Check Overall Status

```bash
kubectl get all -n stablerisk
```

**Expected output:**
```
NAME                           READY   STATUS    RESTARTS   AGE
pod/api-xxx                    1/1     Running   0          2m
pod/monitor-xxx                1/1     Running   0          3m
pod/postgres-0                 1/1     Running   0          5m
pod/raphtory-xxx               1/1     Running   0          4m
pod/web-xxx                    1/1     Running   0          1m

NAME                       TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)
service/api-service        ClusterIP   10.96.xxx.xxx   <none>        8080/TCP,9090/TCP
service/monitor-service    ClusterIP   10.96.xxx.xxx   <none>        9090/TCP
service/postgres-service   ClusterIP   10.96.xxx.xxx   <none>        5432/TCP
service/raphtory-service   ClusterIP   10.96.xxx.xxx   <none>        8000/TCP
service/web-service        ClusterIP   10.96.xxx.xxx   <none>        3000/TCP
```

---

## TLS Configuration

### Option A: Let's Encrypt with cert-manager (Recommended)

#### 1. Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

#### 2. Create ClusterIssuer

```bash
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@yourdomain.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
EOF
```

#### 3. Deploy Ingress

```bash
kubectl apply -f deployments/kubernetes/ingress.yaml

# Wait for certificate (may take 2-3 minutes)
kubectl get certificate -n stablerisk -w
```

**Wait for certificate to show READY=True**

#### 4. Get Ingress IP

```bash
kubectl get ingress stablerisk-ingress -n stablerisk

# Get IP address
INGRESS_IP=$(kubectl get ingress stablerisk-ingress -n stablerisk \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "Ingress IP: ${INGRESS_IP}"
```

### Option B: Custom TLS Certificates

If using custom certificates:

```bash
# Create TLS secret from certificate files
kubectl create secret tls stablerisk-tls-cert -n stablerisk \
  --cert=path/to/fullchain.pem \
  --key=path/to/privkey.pem
```

Update ingress.yaml to reference your secret name.

---

## Initial Configuration

### 1. Update DNS

Create DNS A record pointing to ingress IP:

```
stablerisk.yourdomain.com → <INGRESS_IP>
```

DNS propagation may take 5-60 minutes.

**Verify DNS:**
```bash
nslookup stablerisk.yourdomain.com
# Should return INGRESS_IP
```

### 2. Create Admin User

Connect to the API pod and create an initial admin user:

```bash
# Get API pod name
API_POD=$(kubectl get pod -n stablerisk -l app=api -o jsonpath='{.items[0].metadata.name}')

# Connect to PostgreSQL
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk << 'EOF'
-- Generate a bcrypt hash for password "changeme123"
-- Use https://bcrypt-generator.com/ or bcrypt CLI tool

INSERT INTO users (id, username, email, password_hash, role, is_active, created_at, updated_at)
VALUES (
  gen_random_uuid(),
  'admin',
  'admin@stablerisk.local',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',  -- changeme123
  'admin',
  true,
  NOW(),
  NOW()
);
EOF
```

**⚠️ IMPORTANT:** Change the default password immediately after first login!

### 3. Test Initial Login

```bash
curl -X POST https://${DOMAIN}/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme123"}'
```

**Expected response:**
```json
{
  "access_token": "eyJhbGciOiJS...",
  "refresh_token": "eyJhbGciOiJS...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

---

## Post-Deployment Verification

### 1. Smoke Tests

```bash
#!/bin/bash
DOMAIN="stablerisk.yourdomain.com"

echo "Running smoke tests..."

# Test 1: Health endpoint
echo -n "Testing /health... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://${DOMAIN}/health)
if [ "$STATUS" == "200" ]; then
  echo "✓ PASS"
else
  echo "✗ FAIL (HTTP $STATUS)"
fi

# Test 2: API endpoint (should require auth)
echo -n "Testing /api/v1/outliers (should be 401)... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://${DOMAIN}/api/v1/outliers)
if [ "$STATUS" == "401" ]; then
  echo "✓ PASS"
else
  echo "✗ FAIL (HTTP $STATUS)"
fi

# Test 3: Web dashboard
echo -n "Testing web dashboard... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://${DOMAIN}/)
if [ "$STATUS" == "200" ]; then
  echo "✓ PASS"
else
  echo "✗ FAIL (HTTP $STATUS)"
fi

# Test 4: TLS certificate
echo -n "Testing TLS certificate... "
if openssl s_client -connect ${DOMAIN}:443 -servername ${DOMAIN} < /dev/null 2>&1 | grep -q "Verify return code: 0"; then
  echo "✓ PASS"
else
  echo "✗ FAIL"
fi

echo "Smoke tests complete!"
```

### 2. Functional Tests

```bash
# Login and get token
TOKEN=$(curl -s -X POST https://${DOMAIN}/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme123"}' | \
  jq -r '.access_token')

# Test authenticated endpoint
curl -H "Authorization: Bearer ${TOKEN}" \
  https://${DOMAIN}/api/v1/statistics/transactions?window=1h

# Expected: JSON with transaction statistics
```

### 3. Performance Baseline

```bash
# Install Apache Bench (if not installed)
# sudo apt-get install apache2-utils

# Simple load test (adjust concurrency as needed)
ab -n 1000 -c 10 -H "Authorization: Bearer ${TOKEN}" \
  https://${DOMAIN}/api/v1/health

# Review results for baseline performance
```

---

## Monitoring Setup

### 1. Install Prometheus Stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

### 2. Apply ServiceMonitors

```bash
kubectl apply -f docs/monitoring/servicemonitors/
```

### 3. Access Grafana

```bash
# Port-forward Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80 &

# Get admin password
kubectl get secret -n monitoring prometheus-grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d
echo

# Open http://localhost:3000
# Login: admin / <password from above>
```

### 4. Import Dashboards

Import the StableRisk dashboard from `docs/monitoring/grafana-dashboards/stablerisk-overview.json`

### 5. Configure Alerts

```bash
kubectl apply -f docs/monitoring/alerts/stablerisk-alerts.yaml
```

See `docs/runbooks/04-monitoring.md` for complete monitoring setup.

---

## Backup Configuration

### 1. Configure Automated Backups

```bash
# Create PVC for backups
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: backup-pvc
  namespace: stablerisk
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
EOF

# Deploy backup CronJob
kubectl apply -f docs/runbooks/backup-cronjob.yaml
```

### 2. Test Backup

```bash
# Trigger manual backup
kubectl create job -n stablerisk postgres-backup-manual \
  --from=cronjob/postgres-backup

# Check job status
kubectl logs -n stablerisk job/postgres-backup-manual
```

### 3. Configure Off-Site Backups (Optional)

If using S3:

```bash
# Create S3 credentials secret
kubectl create secret generic s3-credentials -n stablerisk \
  --from-literal=aws-access-key-id=AKIAIOSFODNN7EXAMPLE \
  --from-literal=aws-secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Update backup CronJob to include S3 upload.

See `docs/runbooks/03-backup-restore.md` for complete backup procedures.

---

## Compliance Verification

### ISO27001 Checklist

- [ ] Access control implemented (RBAC)
- [ ] Encryption at rest (Kubernetes Secrets)
- [ ] Encryption in transit (TLS 1.3)
- [ ] Audit logging enabled
- [ ] Backup procedures documented
- [ ] Incident response procedures documented
- [ ] Security monitoring configured

### PCI-DSS Checklist

- [ ] Strong cryptography (AES-256, TLS 1.3)
- [ ] Access control (JWT, RBAC)
- [ ] Audit trails (PostgreSQL audit_logs)
- [ ] Network security (Ingress, firewall rules)
- [ ] Regular testing (automated CI/CD tests)
- [ ] Security patches (automated image updates)

---

## Troubleshooting

### Issue: Pods Not Starting

```bash
# Check pod status
kubectl get pods -n stablerisk

# Describe problematic pod
kubectl describe pod <pod-name> -n stablerisk

# Check logs
kubectl logs <pod-name> -n stablerisk
```

See `docs/runbooks/02-troubleshooting.md` for detailed troubleshooting procedures.

### Issue: Cannot Access via HTTPS

1. Check ingress status: `kubectl get ingress -n stablerisk`
2. Verify DNS: `nslookup ${DOMAIN}`
3. Check certificate: `kubectl get certificate -n stablerisk`
4. Review ingress logs: `kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller`

### Issue: Authentication Failures

1. Verify secrets: `kubectl get secret stablerisk-secrets -n stablerisk`
2. Check API logs: `kubectl logs -n stablerisk -l app=api | grep auth`
3. Test password hash in database

---

## Maintenance

### Regular Tasks

**Daily:**
- Monitor dashboards for anomalies
- Review error logs
- Check backup success

**Weekly:**
- Review outlier patterns
- Check resource usage trends
- Test alerting

**Monthly:**
- Review SLOs
- Test backup restore
- Security patch review
- Capacity planning review

**Quarterly:**
- Disaster recovery drill
- Security audit
- Performance optimization review

---

## Scaling

### Horizontal Scaling

API service auto-scales via HPA (2-10 replicas). To adjust:

```bash
kubectl edit hpa api-hpa -n stablerisk
# Modify minReplicas, maxReplicas, targetCPUUtilizationPercentage
```

### Vertical Scaling

To increase resources for a service:

```bash
kubectl edit deployment <service> -n stablerisk
# Modify resources.requests and resources.limits
```

### Database Scaling

To increase PostgreSQL storage:

```bash
# Edit PVC (requires StorageClass with allowVolumeExpansion: true)
kubectl edit pvc postgres-storage-postgres-0 -n stablerisk
# Increase storage size
```

---

## Upgrading

### Application Upgrade

```bash
# Build and push new images with version tag
docker build -t ${IMAGE_PREFIX}/api:v1.1.0 -f Dockerfile.api .
docker push ${IMAGE_PREFIX}/api:v1.1.0

# Update deployment
kubectl set image deployment/api api=${IMAGE_PREFIX}/api:v1.1.0 -n stablerisk

# Monitor rollout
kubectl rollout status deployment/api -n stablerisk

# Rollback if issues
kubectl rollout undo deployment/api -n stablerisk
```

### Database Migration

```bash
# Run migrations (method depends on migration tool)
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -f /migrations/002_add_column.sql
```

---

## Next Steps

After successful deployment:

1. **Security Hardening**
   - Change default admin password
   - Review RBAC policies
   - Configure network policies
   - Enable pod security policies

2. **Monitoring Configuration**
   - Set up alert notifications (Slack, PagerDuty)
   - Create custom dashboards
   - Configure log retention

3. **Documentation**
   - Document environment-specific configurations
   - Create team runbooks
   - Document escalation procedures

4. **Training**
   - Train team on operational procedures
   - Conduct incident response drills
   - Review monitoring dashboards

---

## Support

For issues or questions:

- **Documentation**: docs/
- **Runbooks**: docs/runbooks/
- **API Documentation**: docs/api/
- **GitHub Issues**: https://github.com/mikedewar/stablerisk/issues
- **Email**: support@stablerisk.example.com

---

## Appendix

### A. Environment Variables Reference

See `deployments/kubernetes/configmap.yaml` for complete list.

### B. Port Reference

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| API | 8080 | HTTP | REST API |
| API | 9090 | HTTP | Metrics |
| Monitor | 9090 | HTTP | Metrics |
| Raphtory | 8000 | HTTP | Graph API |
| Web | 3000 | HTTP | Dashboard |
| PostgreSQL | 5432 | TCP | Database |

### C. Useful Commands

```bash
# Get all resources
kubectl get all -n stablerisk

# Watch pod status
kubectl get pods -n stablerisk -w

# Follow logs
kubectl logs -f -n stablerisk -l app=api

# Execute command in pod
kubectl exec -it -n stablerisk <pod-name> -- /bin/sh

# Port forward for debugging
kubectl port-forward -n stablerisk <pod-name> 8080:8080
```

---

**Deployment Guide Version:** 1.0.0
**Last Updated:** 2026-01-11
