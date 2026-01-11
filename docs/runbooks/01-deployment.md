# Deployment Runbook

## Overview

This runbook covers deployment procedures for StableRisk to Kubernetes clusters.

## Prerequisites

- [ ] Kubernetes cluster running (v1.25+)
- [ ] kubectl configured and authenticated
- [ ] Docker images built and pushed to registry
- [ ] TLS certificates obtained (Let's Encrypt or custom)
- [ ] All secrets prepared (see checklist below)

## Pre-Deployment Checklist

### Secrets Required

- [ ] `DATABASE_PASSWORD` - PostgreSQL password (min 32 chars)
- [ ] `JWT_SECRET` - JWT signing key (min 32 chars, random)
- [ ] `ENCRYPTION_KEY` - AES-256 key (32 bytes, base64)
- [ ] `HMAC_KEY` - HMAC signing key (32 bytes, base64)
- [ ] `TRONGRID_API_KEY` - TronGrid API key

Generate secrets:

```bash
# Generate random passwords/keys
DATABASE_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)
HMAC_KEY=$(openssl rand -base64 32)

echo "DATABASE_PASSWORD=$DATABASE_PASSWORD"
echo "JWT_SECRET=$JWT_SECRET"
echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"
echo "HMAC_KEY=$HMAC_KEY"
```

### Configuration Review

- [ ] Update ConfigMap with correct domain name
- [ ] Review resource limits in deployments
- [ ] Configure HPA min/max replicas
- [ ] Set correct TronGrid API endpoint
- [ ] Configure detection thresholds

## Deployment Procedure

### Step 1: Create Namespace

```bash
kubectl apply -f deployments/kubernetes/namespace.yaml

# Verify
kubectl get namespace stablerisk
```

**Expected Output:**
```
NAME         STATUS   AGE
stablerisk   Active   5s
```

### Step 2: Apply ConfigMap

```bash
kubectl apply -f deployments/kubernetes/configmap.yaml

# Verify
kubectl get configmap -n stablerisk
kubectl describe configmap stablerisk-config -n stablerisk
```

### Step 3: Create Secrets

```bash
kubectl create secret generic stablerisk-secrets -n stablerisk \
  --from-literal=DATABASE_PASSWORD="${DATABASE_PASSWORD}" \
  --from-literal=JWT_SECRET="${JWT_SECRET}" \
  --from-literal=ENCRYPTION_KEY="${ENCRYPTION_KEY}" \
  --from-literal=HMAC_KEY="${HMAC_KEY}" \
  --from-literal=TRONGRID_API_KEY="${TRONGRID_API_KEY}"

# Verify (values should be Opaque)
kubectl get secret stablerisk-secrets -n stablerisk -o yaml
```

**Warning:** Never commit secrets to version control or display in logs!

### Step 4: Deploy PostgreSQL

```bash
kubectl apply -f deployments/kubernetes/postgres.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=postgres -n stablerisk --timeout=300s

# Verify
kubectl get statefulset postgres -n stablerisk
kubectl logs -n stablerisk postgres-0 --tail=50
```

**Health Check:**
```bash
kubectl exec -n stablerisk postgres-0 -- pg_isready -U stablerisk
# Expected: postgres:5432 - accepting connections
```

### Step 5: Deploy Raphtory

```bash
kubectl apply -f deployments/kubernetes/raphtory.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=raphtory -n stablerisk --timeout=300s

# Verify
kubectl get deployment raphtory -n stablerisk
kubectl logs -n stablerisk -l app=raphtory --tail=50
```

**Health Check:**
```bash
kubectl exec -n stablerisk -l app=raphtory -- curl -f http://localhost:8000/health
# Expected: {"status":"ok"}
```

### Step 6: Deploy Monitor

```bash
kubectl apply -f deployments/kubernetes/monitor.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=monitor -n stablerisk --timeout=300s

# Verify
kubectl get deployment monitor -n stablerisk
kubectl logs -n stablerisk -l app=monitor --tail=50
```

**Expected Log Output:**
```
{"level":"info","msg":"Starting StableRisk Monitor"}
{"level":"info","msg":"Connected to TronGrid WebSocket"}
{"level":"info","msg":"Connected to Raphtory"}
```

### Step 7: Deploy API

```bash
kubectl apply -f deployments/kubernetes/api.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=api -n stablerisk --timeout=300s

# Verify
kubectl get deployment api -n stablerisk
kubectl get hpa api-hpa -n stablerisk
```

**Health Check:**
```bash
kubectl exec -n stablerisk -l app=api -- curl -f http://localhost:8080/health
```

### Step 8: Deploy Web

```bash
kubectl apply -f deployments/kubernetes/web.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=web -n stablerisk --timeout=300s

# Verify
kubectl get deployment web -n stablerisk
```

### Step 9: Deploy Ingress

```bash
kubectl apply -f deployments/kubernetes/ingress.yaml

# Wait for ingress to get external IP
kubectl get ingress stablerisk-ingress -n stablerisk -w
```

**Expected Output (after ~2 minutes):**
```
NAME                  HOSTS                          ADDRESS          PORTS
stablerisk-ingress    stablerisk.yourdomain.com      203.0.113.10     80,443
```

### Step 10: Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n stablerisk

# Expected: All pods in Running state with READY 1/1 (or 2/2 for API)
```

**Full Status Check:**
```bash
#!/bin/bash
echo "=== Deployment Status ==="
kubectl get deployments -n stablerisk
echo ""
echo "=== StatefulSets ==="
kubectl get statefulsets -n stablerisk
echo ""
echo "=== Pods ==="
kubectl get pods -n stablerisk
echo ""
echo "=== Services ==="
kubectl get services -n stablerisk
echo ""
echo "=== Ingress ==="
kubectl get ingress -n stablerisk
echo ""
echo "=== HPA Status ==="
kubectl get hpa -n stablerisk
```

### Step 11: Post-Deployment Smoke Tests

```bash
# Get ingress URL
INGRESS_IP=$(kubectl get ingress stablerisk-ingress -n stablerisk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test health endpoint
curl -k https://${INGRESS_IP}/health

# Test API endpoint (should return 401 without auth)
curl -k https://${INGRESS_IP}/api/v1/outliers

# Test web dashboard (should return HTML)
curl -k https://${INGRESS_IP}/ | head -20
```

### Step 12: Create Initial Admin User

```bash
# Connect to API pod
API_POD=$(kubectl get pod -n stablerisk -l app=api -o jsonpath='{.items[0].metadata.name}')

kubectl exec -it -n stablerisk $API_POD -- /bin/sh

# Inside the pod, use the admin CLI (if available) or insert directly
# This example assumes you have a user creation tool
# Otherwise, hash a password and insert into PostgreSQL
```

**Manual User Creation:**
```bash
# Connect to PostgreSQL
kubectl exec -it -n stablerisk postgres-0 -- psql -U stablerisk

-- Create admin user (password: changeme123)
-- Bcrypt hash of "changeme123"
INSERT INTO users (id, username, email, password_hash, role, is_active, created_at, updated_at)
VALUES (
  gen_random_uuid(),
  'admin',
  'admin@stablerisk.local',
  '$2a$10$YourBcryptHashHere',
  'admin',
  true,
  NOW(),
  NOW()
);
```

## Post-Deployment

### Update DNS

Point your domain to the ingress IP:

```bash
# Get IP
kubectl get ingress stablerisk-ingress -n stablerisk -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Create A record:
# stablerisk.yourdomain.com -> <INGRESS_IP>
```

### SSL Certificate Verification

```bash
# Test SSL
openssl s_client -connect stablerisk.yourdomain.com:443 -servername stablerisk.yourdomain.com < /dev/null

# Check certificate details
echo | openssl s_client -connect stablerisk.yourdomain.com:443 2>/dev/null | openssl x509 -noout -dates
```

### Monitoring Setup

- [ ] Verify Prometheus is scraping `/metrics` endpoint
- [ ] Configure alerting rules
- [ ] Set up dashboards (Grafana)
- [ ] Configure log aggregation (ELK/Loki)

## Rollback Procedure

If deployment fails:

```bash
# Rollback deployments
kubectl rollout undo deployment/api -n stablerisk
kubectl rollout undo deployment/monitor -n stablerisk
kubectl rollout undo deployment/web -n stablerisk
kubectl rollout undo deployment/raphtory -n stablerisk

# Check rollback status
kubectl rollout status deployment/api -n stablerisk
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n stablerisk

# Check logs
kubectl logs <pod-name> -n stablerisk --previous
```

Common issues:
- **ImagePullBackOff**: Check image name and registry authentication
- **CrashLoopBackOff**: Check logs for startup errors
- **Pending**: Check resource constraints and node capacity

### Database Connection Issues

```bash
# Test database connectivity from API pod
kubectl exec -it -n stablerisk <api-pod> -- nc -zv postgres-service 5432

# Check database logs
kubectl logs -n stablerisk postgres-0
```

### Ingress Not Getting IP

```bash
# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Verify ingress class
kubectl get ingressclass
```

## Compliance Notes

- All secrets are stored in Kubernetes Secrets (encrypted at rest)
- TLS 1.3 enforced via Ingress configuration
- Audit logs enabled on all mutation operations
- RBAC configured via Kubernetes service accounts

## Next Steps

After successful deployment:
1. Review monitoring dashboards
2. Set up automated backups (see `02-backup-restore.md`)
3. Configure alerting rules
4. Document any environment-specific configurations
5. Schedule security review
