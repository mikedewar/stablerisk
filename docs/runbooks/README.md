# StableRisk Operational Runbooks

This directory contains operational runbooks for managing and maintaining StableRisk in production.

## Available Runbooks

### [01-deployment.md](01-deployment.md)
**Purpose:** Step-by-step deployment procedures for StableRisk to Kubernetes

**When to use:**
- Initial production deployment
- Deploying to new environment (staging, DR)
- Major version upgrades

**Key sections:**
- Prerequisites and checklists
- Deployment procedure (all 12 steps)
- Post-deployment verification
- Rollback procedures

**Estimated time:** 45-60 minutes for full deployment

---

### [02-troubleshooting.md](02-troubleshooting.md)
**Purpose:** Diagnose and resolve common issues

**When to use:**
- Service outages or degradation
- Investigating alerts
- Performance issues
- Configuration problems

**Key sections:**
- Quick diagnostics scripts
- Issue categories:
  - Monitor service issues
  - API service issues
  - Raphtory service issues
  - PostgreSQL issues
  - Web dashboard issues
  - Ingress/network issues
- Emergency procedures
- Diagnostic bundle collection

**Target resolution time:** 15-30 minutes for common issues

---

### [03-backup-restore.md](03-backup-restore.md)
**Purpose:** Backup and disaster recovery procedures

**When to use:**
- Scheduled backups (daily automated)
- Manual backups before major changes
- Data recovery after incidents
- Disaster recovery scenarios
- Environment cloning/migration

**Key sections:**
- Backup strategy and schedule
- PostgreSQL backup/restore
- Raphtory graph backup/restore
- Configuration backup
- TLS certificate backup
- Full disaster recovery
- Backup verification

**RTO/RPO:**
- Database restore: 15-30 minutes
- Full system restore: 1-2 hours
- RPO: 6 hours (incremental backups)

---

### [04-monitoring.md](04-monitoring.md)
**Purpose:** Monitoring, alerting, and observability

**When to use:**
- Setting up monitoring infrastructure
- Configuring alerts
- Creating dashboards
- Investigating performance issues
- SLO tracking

**Key sections:**
- Key metrics (application & infrastructure)
- Prometheus setup
- Grafana dashboards
- Alerting rules (10+ critical alerts)
- AlertManager configuration
- SLO/SLI definitions
- Incident response procedures
- Performance monitoring

**Alert response time:** < 5 minutes for P0 incidents

---

## Quick Reference

### Common Commands

```bash
# Check overall status
kubectl get all -n stablerisk

# View logs
kubectl logs -n stablerisk -l app=api --tail=100

# Restart service
kubectl rollout restart deployment/api -n stablerisk

# Scale service
kubectl scale deployment/api -n stablerisk --replicas=5

# Execute in pod
kubectl exec -it -n stablerisk <pod-name> -- /bin/sh
```

### Health Check URLs

- Overall health: `https://stablerisk.yourdomain.com/health`
- API readiness: `https://stablerisk.yourdomain.com/readiness`
- API liveness: `https://stablerisk.yourdomain.com/liveness`
- Metrics: `https://stablerisk.yourdomain.com/metrics` (internal only)

### Emergency Contacts

- On-call Engineer: [PagerDuty/Phone]
- Database Team: dba@yourdomain.com
- Infrastructure Team: infra@yourdomain.com
- Security Team: security@yourdomain.com

---

## Runbook Development

### Creating New Runbooks

When creating a new runbook, include:

1. **Overview**: Purpose and scope
2. **Prerequisites**: Required access, tools
3. **Procedure**: Step-by-step instructions with commands
4. **Verification**: How to verify success
5. **Rollback**: How to undo changes
6. **Troubleshooting**: Common issues
7. **References**: Related docs, links

### Runbook Template

```markdown
# [Task Name] Runbook

## Overview
Brief description of what this runbook covers.

## Prerequisites
- [ ] Required access
- [ ] Required tools
- [ ] Required knowledge

## Procedure

### Step 1: [Action]
Description of step.

\`\`\`bash
# Commands here
\`\`\`

**Expected output:**
\`\`\`
Output here
\`\`\`

### Step 2: [Action]
...

## Verification
How to verify the procedure was successful.

## Rollback
How to undo changes if needed.

## Troubleshooting
Common issues and their solutions.

## References
- Related documentation
- External links
```

---

## Runbook Maintenance

### Review Schedule

- **Monthly**: Review for accuracy, update commands
- **Quarterly**: Test all procedures in staging
- **After incidents**: Update based on lessons learned
- **After changes**: Update affected runbooks

### Version Control

All runbooks are version controlled in git. To propose changes:

```bash
# Create branch
git checkout -b update-runbook-deployment

# Make changes
vim docs/runbooks/01-deployment.md

# Commit and push
git add docs/runbooks/01-deployment.md
git commit -m "Update deployment runbook with new health check"
git push origin update-runbook-deployment

# Create pull request
```

---

## Testing Runbooks

### Staging Environment

Test all procedures in staging before using in production:

```bash
# Use staging context
kubectl config use-context staging

# Follow runbook procedures
# Verify results
```

### Disaster Recovery Drills

Quarterly DR drills to test backup/restore procedures:

- Schedule maintenance window
- Follow disaster recovery procedures
- Document issues found
- Update runbooks accordingly

---

## Runbook Usage Guidelines

### Before Starting

1. **Read the entire runbook** before executing commands
2. **Check prerequisites** are met
3. **Notify team** if making changes
4. **Take backup** if making data changes
5. **Have rollback plan** ready

### During Execution

1. **Follow steps in order** unless instructed otherwise
2. **Verify each step** before proceeding
3. **Document deviations** from runbook
4. **Take screenshots** for audit trail
5. **Monitor metrics** during changes

### After Completion

1. **Verify success** using verification steps
2. **Update status** (if using ticketing)
3. **Document lessons learned**
4. **Update runbook** if improvements found
5. **Notify team** of completion

---

## Related Documentation

- **[Deployment Guide](../DEPLOYMENT_GUIDE.md)**: Complete deployment instructions
- **[API Documentation](../api/)**: API reference and examples
- **[Architecture](../ARCHITECTURE.md)**: System architecture and design
- **[Test Report](../TEST_REPORT.md)**: Testing coverage and results

---

## Compliance

These runbooks support:

### ISO27001
- **A.12.1**: Operational procedures documented
- **A.16.1**: Incident management procedures
- **A.17.1**: Business continuity procedures

### PCI-DSS
- **Requirement 12.10**: Incident response plan
- **Requirement 10.5**: Audit trail protection procedures
- **Requirement 9.5**: Media backup procedures

---

## Feedback

To improve these runbooks:

- **Open issue**: https://github.com/mikedewar/stablerisk/issues
- **Submit PR**: For corrections or improvements
- **Email**: ops@stablerisk.example.com

---

**Last Updated:** 2026-01-11
**Runbook Version:** 1.0.0
