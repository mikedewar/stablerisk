# Backup and Restore Runbook

## Overview

This runbook covers backup and restore procedures for StableRisk data, including PostgreSQL databases and configuration.

## Backup Strategy

### What to Back Up

1. **PostgreSQL Database** (Critical)
   - User accounts and credentials
   - Audit logs
   - Outlier records
   - Refresh tokens

2. **Raphtory Graph Data** (Important)
   - Transaction graph
   - Node and edge data
   - Temporal data

3. **Configuration** (Important)
   - Kubernetes Secrets
   - ConfigMaps
   - TLS certificates

4. **Not Backed Up**
   - Raw blockchain data (can be re-ingested)
   - Temporary WebSocket connections
   - Prometheus metrics (ephemeral)

### Backup Schedule

- **Full backup**: Daily at 02:00 UTC
- **Incremental backup**: Every 6 hours
- **Retention**: 30 days for daily, 7 days for incremental
- **Off-site replication**: Every 24 hours

---

## PostgreSQL Backup

### Manual Backup

```bash
# Get current timestamp
BACKUP_DATE=$(date +%Y%m%d-%H%M%S)

# Create backup
kubectl exec -n stablerisk postgres-0 -- pg_dump -U stablerisk -Fc stablerisk > \
  stablerisk-backup-${BACKUP_DATE}.dump

# Verify backup
ls -lh stablerisk-backup-${BACKUP_DATE}.dump

# Compress (if not using -Fc format)
gzip stablerisk-backup-${BACKUP_DATE}.dump
```

### Automated Backup with CronJob

Create a Kubernetes CronJob for automated backups:

```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: stablerisk
spec:
  schedule: "0 2 * * *"  # Daily at 02:00 UTC
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16-alpine
            env:
              - name: PGHOST
                value: "postgres-service"
              - name: PGUSER
                value: "stablerisk"
              - name: PGPASSWORD
                valueFrom:
                  secretKeyRef:
                    name: stablerisk-secrets
                    key: DATABASE_PASSWORD
            command:
              - /bin/sh
              - -c
              - |
                BACKUP_FILE="/backups/stablerisk-$(date +%Y%m%d-%H%M%S).dump"
                pg_dump -Fc stablerisk > "$BACKUP_FILE"
                echo "Backup completed: $BACKUP_FILE"
                ls -lh "$BACKUP_FILE"

                # Upload to S3 (if configured)
                # aws s3 cp "$BACKUP_FILE" s3://your-backup-bucket/stablerisk/

                # Clean up old backups (keep last 30 days)
                find /backups -name "stablerisk-*.dump" -mtime +30 -delete
            volumeMounts:
              - name: backup-storage
                mountPath: /backups
          restartPolicy: OnFailure
          volumes:
            - name: backup-storage
              persistentVolumeClaim:
                claimName: backup-pvc
```

Apply the CronJob:

```bash
kubectl apply -f backup-cronjob.yaml

# Verify
kubectl get cronjob -n stablerisk
```

### Backup to S3

For cloud backups:

```bash
# Install AWS CLI in backup container (or use aws-cli image)
BACKUP_DATE=$(date +%Y%m%d-%H%M%S)

kubectl exec -n stablerisk postgres-0 -- pg_dump -U stablerisk -Fc stablerisk | \
  aws s3 cp - s3://your-backup-bucket/stablerisk/stablerisk-${BACKUP_DATE}.dump

# Verify upload
aws s3 ls s3://your-backup-bucket/stablerisk/
```

---

## PostgreSQL Restore

### Pre-Restore Checklist

- [ ] Verify backup file integrity
- [ ] Stop all services that write to database (API, Monitor)
- [ ] Create a backup of current state (if applicable)
- [ ] Notify users of maintenance window

### Restore Procedure

```bash
# 1. Stop services
kubectl scale deployment/api -n stablerisk --replicas=0
kubectl scale deployment/monitor -n stablerisk --replicas=0

# Wait for pods to terminate
kubectl wait --for=delete pod -l app=api -n stablerisk --timeout=60s
kubectl wait --for=delete pod -l app=monitor -n stablerisk --timeout=60s

# 2. Drop existing database (CAUTION!)
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c "DROP DATABASE IF EXISTS stablerisk;"

# 3. Create fresh database
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -c "CREATE DATABASE stablerisk;"

# 4. Restore from backup
cat stablerisk-backup-20260111-020000.dump | \
  kubectl exec -i -n stablerisk postgres-0 -- pg_restore -U stablerisk -d stablerisk -v

# 5. Verify restore
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -d stablerisk -c "SELECT COUNT(*) FROM users;"
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk -d stablerisk -c "SELECT COUNT(*) FROM outliers;"

# 6. Restart services
kubectl scale deployment/api -n stablerisk --replicas=2
kubectl scale deployment/monitor -n stablerisk --replicas=1

# 7. Verify functionality
kubectl logs -n stablerisk -l app=api --tail=50
```

### Point-in-Time Recovery (PITR)

For PostgreSQL with WAL archiving:

```bash
# Configure WAL archiving in PostgreSQL (one-time setup)
kubectl exec -n stablerisk postgres-0 -- psql -U stablerisk << EOF
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET archive_mode = on;
ALTER SYSTEM SET archive_command = 'test ! -f /archive/%f && cp %p /archive/%f';
EOF

# Restart PostgreSQL
kubectl rollout restart statefulset/postgres -n stablerisk

# Restore to specific timestamp
# (Requires base backup + WAL archives)
kubectl exec -n stablerisk postgres-0 -- pg_restore \
  --target-time="2026-01-11 14:30:00" \
  -d stablerisk stablerisk-base-backup.dump
```

---

## Raphtory Graph Backup

Raphtory stores graph data in memory with optional disk persistence.

### Enable Persistence

Update Raphtory deployment to use persistent storage:

```yaml
# Add to raphtory.yaml
spec:
  template:
    spec:
      volumes:
        - name: graph-storage
          persistentVolumeClaim:
            claimName: raphtory-pvc
      containers:
        - name: raphtory
          volumeMounts:
            - name: graph-storage
              mountPath: /data
          env:
            - name: RAPHTORY_STORAGE_PATH
              value: "/data/graph"
```

### Manual Backup

```bash
# Export graph to disk (via Raphtory API)
kubectl exec -n stablerisk -l app=raphtory -- curl -X POST \
  http://localhost:8000/graph/export \
  -H "Content-Type: application/json" \
  -d '{"format":"json","path":"/data/export"}'

# Copy export to local machine
kubectl cp stablerisk/raphtory-pod:/data/export/graph.json \
  ./raphtory-backup-$(date +%Y%m%d).json
```

### Restore Raphtory

```bash
# Copy backup to pod
kubectl cp ./raphtory-backup-20260111.json \
  stablerisk/raphtory-pod:/data/import/graph.json

# Import via API
kubectl exec -n stablerisk -l app=raphtory -- curl -X POST \
  http://localhost:8000/graph/import \
  -H "Content-Type: application/json" \
  -d '{"format":"json","path":"/data/import/graph.json"}'
```

**Note:** For large graphs, consider rebuilding from PostgreSQL audit logs or re-ingesting from blockchain.

---

## Configuration Backup

### Backup Secrets and ConfigMaps

```bash
# Create backup directory
mkdir -p config-backup-$(date +%Y%m%d)
cd config-backup-$(date +%Y%m%d)

# Export all Kubernetes resources
kubectl get configmap stablerisk-config -n stablerisk -o yaml > configmap.yaml
kubectl get secret stablerisk-secrets -n stablerisk -o yaml > secrets.yaml
kubectl get ingress stablerisk-ingress -n stablerisk -o yaml > ingress.yaml
kubectl get all -n stablerisk -o yaml > all-resources.yaml

# Create archive
tar -czf ../stablerisk-config-backup-$(date +%Y%m%d).tar.gz .

# Upload to secure location
# aws s3 cp ../stablerisk-config-backup-$(date +%Y%m%d).tar.gz \
#   s3://your-config-bucket/stablerisk/
```

### Restore Configuration

```bash
# Extract backup
tar -xzf stablerisk-config-backup-20260111.tar.gz

# Apply configuration
kubectl apply -f configmap.yaml
kubectl apply -f secrets.yaml
kubectl apply -f ingress.yaml

# Restart services to pick up changes
kubectl rollout restart deployment/api -n stablerisk
kubectl rollout restart deployment/monitor -n stablerisk
```

---

## TLS Certificate Backup

```bash
# Backup Let's Encrypt certificates
kubectl exec -n stablerisk certbot-pod -- tar -czf /tmp/letsencrypt-backup.tar.gz \
  /etc/letsencrypt

kubectl cp stablerisk/certbot-pod:/tmp/letsencrypt-backup.tar.gz \
  ./letsencrypt-backup-$(date +%Y%m%d).tar.gz

# Store securely
# gpg --encrypt --recipient admin@yourdomain.com letsencrypt-backup-*.tar.gz
```

### Restore Certificates

```bash
# Copy backup to certbot pod
kubectl cp ./letsencrypt-backup-20260111.tar.gz \
  stablerisk/certbot-pod:/tmp/restore.tar.gz

# Extract
kubectl exec -n stablerisk certbot-pod -- tar -xzf /tmp/restore.tar.gz -C /

# Reload Nginx
kubectl exec -n stablerisk nginx-pod -- nginx -s reload
```

---

## Disaster Recovery

### Full System Restore

In case of complete cluster failure:

```bash
# 1. Deploy fresh Kubernetes cluster
# 2. Install prerequisites (ingress controller, cert-manager)

# 3. Restore secrets
kubectl create namespace stablerisk
kubectl apply -f config-backup/secrets.yaml

# 4. Restore ConfigMap
kubectl apply -f config-backup/configmap.yaml

# 5. Deploy infrastructure
kubectl apply -f deployments/kubernetes/postgres.yaml
kubectl apply -f deployments/kubernetes/raphtory.yaml

# 6. Restore PostgreSQL data
kubectl wait --for=condition=ready pod -l app=postgres -n stablerisk --timeout=300s
cat stablerisk-backup-latest.dump | \
  kubectl exec -i -n stablerisk postgres-0 -- pg_restore -U stablerisk -d stablerisk -v

# 7. Deploy application services
kubectl apply -f deployments/kubernetes/monitor.yaml
kubectl apply -f deployments/kubernetes/api.yaml
kubectl apply -f deployments/kubernetes/web.yaml

# 8. Restore ingress
kubectl apply -f deployments/kubernetes/ingress.yaml

# 9. Verify all services
kubectl get all -n stablerisk
```

### Recovery Time Objectives (RTO)

- **Database restore**: 15-30 minutes
- **Full system restore**: 1-2 hours
- **Configuration restore**: 5-10 minutes

### Recovery Point Objectives (RPO)

- **PostgreSQL**: 6 hours (incremental backups)
- **Configuration**: 24 hours
- **Graph data**: Best effort (can rebuild from blockchain)

---

## Backup Verification

### Automated Backup Testing

```bash
# Test restore in isolated namespace
kubectl create namespace stablerisk-test

# Restore backup
cat latest-backup.dump | \
  kubectl exec -i -n stablerisk-test postgres-0 -- pg_restore -U stablerisk -d stablerisk

# Run validation queries
kubectl exec -n stablerisk-test postgres-0 -- psql -U stablerisk -d stablerisk << EOF
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM outliers;
SELECT COUNT(*) FROM audit_logs;
SELECT MAX(created_at) FROM outliers;
EOF

# Cleanup
kubectl delete namespace stablerisk-test
```

### Backup Integrity Checks

```bash
# Check backup file integrity
pg_restore --list stablerisk-backup-20260111.dump | head -20

# Verify backup size (should be reasonable)
ls -lh stablerisk-backup-*.dump

# Calculate checksum
sha256sum stablerisk-backup-20260111.dump > backup.sha256
```

---

## Compliance Notes

### ISO27001
- A.12.3 Information Backup: Automated daily backups with 30-day retention
- A.17.1 Business Continuity: Documented disaster recovery procedures

### PCI-DSS
- Requirement 9.5: Backups stored securely with encryption
- Requirement 10.5: Audit logs protected from alteration and included in backups

---

## Best Practices

1. **Test restores regularly** - Monthly restore drills
2. **Monitor backup success** - Alert on failed backups
3. **Encrypt backups** - Use encryption at rest and in transit
4. **Off-site replication** - Store backups in different region/cloud
5. **Document procedures** - Keep this runbook updated
6. **Verify backup completeness** - Automated integrity checks
7. **Secure backup storage** - Access control and encryption

---

## Troubleshooting

### Backup Fails

```bash
# Check disk space
kubectl exec -n stablerisk postgres-0 -- df -h

# Check PostgreSQL logs
kubectl logs -n stablerisk postgres-0 | grep -i backup

# Test manual backup
kubectl exec -n stablerisk postgres-0 -- pg_dump -U stablerisk stablerisk -f /tmp/test.dump
```

### Restore Fails

```bash
# Check backup file format
file stablerisk-backup.dump

# Try restore with verbose output
pg_restore -l stablerisk-backup.dump

# Check PostgreSQL version compatibility
kubectl exec -n stablerisk postgres-0 -- psql --version
```

---

## Emergency Contacts

- Database Administrator: dba@yourdomain.com
- Infrastructure Team: infra@yourdomain.com
- On-call Engineer: +1-XXX-XXX-XXXX
