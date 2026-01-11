# Troubleshooting Runbook

## Overview

This runbook covers common issues and their resolution procedures for StableRisk.

## Quick Diagnostics

### System Health Check

```bash
#!/bin/bash
# Run this script for quick system overview

echo "=== Namespace Status ==="
kubectl get all -n stablerisk

echo -e "\n=== Pod Health ==="
kubectl get pods -n stablerisk -o wide

echo -e "\n=== Recent Events ==="
kubectl get events -n stablerisk --sort-by='.lastTimestamp' | tail -20

echo -e "\n=== Resource Usage ==="
kubectl top pods -n stablerisk

echo -e "\n=== Service Endpoints ==="
kubectl get endpoints -n stablerisk
```

## Issue Categories

### 1. Monitor Service Issues

#### Problem: Monitor Not Receiving Transactions

**Symptoms:**
- No new transactions in database
- Monitor logs show "no connection" errors
- Outlier detection returns empty results

**Diagnosis:**

```bash
# Check monitor pod logs
kubectl logs -n stablerisk -l app=monitor --tail=100 | grep -i error

# Check TronGrid connectivity
kubectl exec -n stablerisk -l app=monitor -- curl -v https://api.trongrid.io
```

**Common Causes:**

1. **TronGrid API Key Invalid**
   ```bash
   # Verify API key secret
   kubectl get secret stablerisk-secrets -n stablerisk -o json | \
     jq -r '.data.TRONGRID_API_KEY' | base64 -d

   # Test API key
   curl -H "TRON-PRO-API-KEY: <your-key>" https://api.trongrid.io/wallet/getnowblock
   ```

2. **Network Policy Blocking Outbound**
   ```bash
   # Test outbound connectivity
   kubectl exec -n stablerisk -l app=monitor -- nc -zv api.trongrid.io 443
   ```

   **Fix:** Update NetworkPolicy to allow egress to TronGrid

3. **Raphtory Connection Failed**
   ```bash
   # Test Raphtory connectivity
   kubectl exec -n stablerisk -l app=monitor -- curl http://raphtory-service:8000/health
   ```

   **Fix:** Check Raphtory service status (see section 3)

**Resolution:**

```bash
# Restart monitor pods
kubectl rollout restart deployment/monitor -n stablerisk

# Watch logs
kubectl logs -n stablerisk -l app=monitor -f
```

**Success Criteria:**
- Logs show "Connected to TronGrid WebSocket"
- Logs show "Transaction stored" messages
- Transaction count increasing in database

---

### 2. API Service Issues

#### Problem: API Returns 500 Errors

**Symptoms:**
- Web dashboard shows errors
- `/health` endpoint fails
- High error rate in logs

**Diagnosis:**

```bash
# Check API logs
kubectl logs -n stablerisk -l app=api --tail=200 | grep -i error

# Check API pod status
kubectl describe pod -n stablerisk -l app=api

# Test health endpoint
kubectl exec -n stablerisk -l app=api -- curl http://localhost:8080/health
```

**Common Causes:**

1. **Database Connection Pool Exhausted**

   **Symptoms in logs:**
   ```
   pq: sorry, too many clients already
   ```

   **Fix:**
   ```bash
   # Increase max_connections in PostgreSQL
   kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c \
     "ALTER SYSTEM SET max_connections = 200;"

   kubectl rollout restart statefulset/postgres -n stablerisk
   ```

2. **Out of Memory**

   **Check memory usage:**
   ```bash
   kubectl top pod -n stablerisk -l app=api
   ```

   **Fix:** Increase memory limits in deployment:
   ```bash
   kubectl edit deployment api -n stablerisk
   # Update resources.limits.memory to 2Gi
   ```

3. **Raphtory Timeout**

   **Symptoms in logs:**
   ```
   context deadline exceeded
   ```

   **Fix:** Check Raphtory service and increase timeout in ConfigMap

**Resolution:**

```bash
# Scale up API replicas temporarily
kubectl scale deployment/api -n stablerisk --replicas=5

# Restart API pods
kubectl rollout restart deployment/api -n stablerisk

# Monitor recovery
kubectl logs -n stablerisk -l app=api -f
```

---

#### Problem: Authentication Failures

**Symptoms:**
- Users cannot login
- "Invalid token" errors
- 401 responses on authenticated endpoints

**Diagnosis:**

```bash
# Check JWT secret consistency
kubectl get secret stablerisk-secrets -n stablerisk -o json | \
  jq -r '.data.JWT_SECRET' | base64 -d | wc -c
# Should be >= 32 characters

# Test login endpoint
kubectl exec -n stablerisk -l app=api -- curl -X POST \
  http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"test"}'
```

**Common Causes:**

1. **JWT Secret Mismatch**
   - API pods have different JWT_SECRET values
   - Secret was rotated without restarting pods

   **Fix:**
   ```bash
   # Restart all API pods to pick up consistent secret
   kubectl rollout restart deployment/api -n stablerisk
   ```

2. **Clock Skew**
   - JWT expiry validation fails due to time drift

   **Check:**
   ```bash
   # Check pod time
   kubectl exec -n stablerisk -l app=api -- date -u
   # Compare with actual time
   ```

   **Fix:** Ensure NTP is configured on nodes

3. **Bcrypt Hash Issues**
   - Password hash in database doesn't match algorithm

   **Fix:** Recreate user with correct hash

---

### 3. Raphtory Service Issues

#### Problem: Raphtory Service Unavailable

**Symptoms:**
- Monitor fails to store transactions
- API graph queries fail
- `/readiness` shows raphtory: not ok

**Diagnosis:**

```bash
# Check Raphtory pod status
kubectl get pods -n stablerisk -l app=raphtory

# Check logs
kubectl logs -n stablerisk -l app=raphtory --tail=100

# Test health endpoint
kubectl exec -n stablerisk -l app=raphtory -- curl http://localhost:8000/health
```

**Common Causes:**

1. **Out of Memory**

   Raphtory is memory-intensive for large graphs.

   **Check:**
   ```bash
   kubectl top pod -n stablerisk -l app=raphtory
   ```

   **Fix:**
   ```bash
   kubectl edit deployment raphtory -n stablerisk
   # Increase resources.limits.memory to 8Gi
   ```

2. **Python Dependencies Missing**

   **Symptoms in logs:**
   ```
   ModuleNotFoundError: No module named 'raphtory'
   ```

   **Fix:** Rebuild Docker image with correct requirements.txt

3. **Port Binding Failed**

   **Symptoms in logs:**
   ```
   Address already in use
   ```

   **Fix:**
   ```bash
   kubectl delete pod -n stablerisk -l app=raphtory
   # Pod will be recreated automatically
   ```

---

### 4. PostgreSQL Issues

#### Problem: Database Connection Refused

**Symptoms:**
- API cannot connect to database
- "connection refused" in logs
- Pods show Init:Error status

**Diagnosis:**

```bash
# Check PostgreSQL pod
kubectl get pod -n stablerisk postgres-0

# Check logs
kubectl logs -n stablerisk postgres-0

# Test connectivity
kubectl exec -n stablerisk postgres-0 -- pg_isready -U stablerisk
```

**Common Causes:**

1. **PostgreSQL Not Ready**

   **Wait for initialization:**
   ```bash
   kubectl wait --for=condition=ready pod -l app=postgres -n stablerisk --timeout=300s
   ```

2. **Password Mismatch**

   **Check secret:**
   ```bash
   kubectl get secret stablerisk-secrets -n stablerisk -o json | \
     jq -r '.data.DATABASE_PASSWORD' | base64 -d
   ```

   **Fix:** Ensure ConfigMap and Secret have matching credentials

3. **Disk Full**

   **Check PVC usage:**
   ```bash
   kubectl exec -n stablerisk postgres-0 -- df -h /var/lib/postgresql/data
   ```

   **Fix:** Expand PVC or clean up old data

---

#### Problem: Slow Query Performance

**Symptoms:**
- API responses slow
- Outlier list takes >5 seconds
- High CPU on postgres pod

**Diagnosis:**

```bash
# Check running queries
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c \
  "SELECT pid, query, state, query_start FROM pg_stat_activity WHERE state = 'active';"

# Check slow queries
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c \
  "SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

**Fix:**

```bash
# Create missing indexes
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk << EOF
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_outliers_timestamp ON outliers(timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_outliers_severity ON outliers(severity);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_outliers_acknowledged ON outliers(acknowledged);
EOF

# Analyze tables
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c "ANALYZE;"
```

---

### 5. Web Dashboard Issues

#### Problem: Dashboard Not Loading

**Symptoms:**
- Blank page or loading spinner
- Browser console shows errors
- Ingress returns 502/504

**Diagnosis:**

```bash
# Check web pod status
kubectl get pods -n stablerisk -l app=web

# Check logs
kubectl logs -n stablerisk -l app=web --tail=50

# Test web service
kubectl exec -n stablerisk -l app=web -- curl http://localhost:3000/
```

**Common Causes:**

1. **Build Errors**

   **Symptoms in logs:**
   ```
   SyntaxError: Unexpected token
   ```

   **Fix:** Rebuild web image with proper build process

2. **API URL Misconfigured**

   **Check environment:**
   ```bash
   kubectl exec -n stablerisk -l app=web -- env | grep PUBLIC_API_URL
   ```

   **Fix:** Update deployment with correct PUBLIC_API_URL

3. **WebSocket Connection Failed**

   **Check browser console:**
   ```
   WebSocket connection to 'wss://...' failed
   ```

   **Fix:** Verify ingress WebSocket configuration

---

### 6. Ingress/Network Issues

#### Problem: Cannot Access via HTTPS

**Symptoms:**
- Connection timeout or refused
- SSL certificate errors
- 404 on all routes

**Diagnosis:**

```bash
# Check ingress status
kubectl describe ingress stablerisk-ingress -n stablerisk

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=100

# Test from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v http://api-service.stablerisk.svc.cluster.local:8080/health
```

**Common Causes:**

1. **Ingress Controller Not Running**

   **Check:**
   ```bash
   kubectl get pods -n ingress-nginx
   ```

   **Fix:** Install/restart ingress controller

2. **Certificate Issues**

   **Check certificate:**
   ```bash
   kubectl get certificate -n stablerisk
   kubectl describe certificate -n stablerisk
   ```

   **Fix:** Check cert-manager logs and reissue certificate

3. **DNS Not Pointing to Ingress**

   **Check DNS:**
   ```bash
   nslookup stablerisk.yourdomain.com
   ```

   **Fix:** Update DNS A record

---

## Performance Issues

### High Memory Usage

```bash
# Check memory usage across all pods
kubectl top pods -n stablerisk --sort-by=memory

# Identify memory leaks
kubectl exec -n stablerisk <pod-name> -- top -b -n 1
```

**Fix:**
- Increase resource limits
- Enable memory profiling (Go pprof)
- Restart affected pods

### High CPU Usage

```bash
# Check CPU usage
kubectl top pods -n stablerisk --sort-by=cpu

# Check for CPU throttling
kubectl describe pod <pod-name> -n stablerisk | grep -A 5 "Limits"
```

**Fix:**
- Increase CPU limits
- Review detection algorithm efficiency
- Scale horizontally with HPA

---

## Emergency Procedures

### Complete Service Restart

```bash
# Restart all services in order
kubectl rollout restart deployment/raphtory -n stablerisk
sleep 30
kubectl rollout restart deployment/monitor -n stablerisk
sleep 30
kubectl rollout restart deployment/api -n stablerisk
sleep 30
kubectl rollout restart deployment/web -n stablerisk

# Verify all running
kubectl get pods -n stablerisk
```

### Data Corruption Recovery

See `03-backup-restore.md` for database recovery procedures.

---

## Logging and Debugging

### Enable Debug Logging

```bash
# Update ConfigMap
kubectl edit configmap stablerisk-config -n stablerisk
# Change LOGGING_LEVEL: "debug"

# Restart affected services
kubectl rollout restart deployment/api -n stablerisk
kubectl rollout restart deployment/monitor -n stablerisk
```

### Collect Diagnostic Bundle

```bash
#!/bin/bash
# Collect all relevant info for support

mkdir -p stablerisk-diagnostics
cd stablerisk-diagnostics

kubectl get all -n stablerisk > resources.txt
kubectl describe pods -n stablerisk > pods-describe.txt
kubectl logs -n stablerisk -l app=api --tail=1000 > api-logs.txt
kubectl logs -n stablerisk -l app=monitor --tail=1000 > monitor-logs.txt
kubectl logs -n stablerisk -l app=raphtory --tail=1000 > raphtory-logs.txt
kubectl get events -n stablerisk --sort-by='.lastTimestamp' > events.txt
kubectl top pods -n stablerisk > resource-usage.txt

tar -czf stablerisk-diagnostics-$(date +%Y%m%d-%H%M%S).tar.gz *.txt
```

---

## Escalation

If issues persist after following this runbook:

1. Collect diagnostic bundle (above)
2. Check GitHub Issues: https://github.com/yourusername/stablerisk/issues
3. Contact support with:
   - Issue description
   - Steps to reproduce
   - Diagnostic bundle
   - Environment details (cluster version, node count, etc.)
