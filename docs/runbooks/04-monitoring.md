# Monitoring and Observability Runbook

## Overview

This runbook covers monitoring, alerting, and observability practices for StableRisk.

## Monitoring Stack

### Components

1. **Prometheus** - Metrics collection and storage
2. **Grafana** - Visualization and dashboards
3. **AlertManager** - Alert routing and management
4. **Loki** (Optional) - Log aggregation
5. **Jaeger** (Optional) - Distributed tracing

---

## Key Metrics

### Application Metrics

All services expose Prometheus metrics on `/metrics` endpoint (port 9090).

#### API Service Metrics

```promql
# HTTP Request Rate
rate(http_requests_total{service="api"}[5m])

# Request Duration (p95)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service="api"}[5m]))

# Error Rate
rate(http_requests_total{service="api",status=~"5.."}[5m])

# Active WebSocket Connections
websocket_connections_active{service="api"}

# Outlier Detection Rate
rate(outliers_detected_total[5m])

# JWT Token Operations
rate(jwt_operations_total{operation="validate"}[5m])

# Database Query Duration
histogram_quantile(0.95, rate(db_query_duration_seconds_bucket[5m]))
```

#### Monitor Service Metrics

```promql
# Transaction Ingestion Rate
rate(transactions_processed_total{service="monitor"}[5m])

# TronGrid Connection Status
trongrid_connection_status{service="monitor"}

# Raphtory Write Success Rate
rate(raphtory_writes_total{status="success"}[5m]) /
rate(raphtory_writes_total[5m])

# Transaction Processing Lag
transaction_processing_lag_seconds{service="monitor"}

# WebSocket Reconnections
rate(websocket_reconnections_total{service="monitor"}[5m])
```

#### Raphtory Service Metrics

```promql
# Graph Node Count
graph_nodes_total{service="raphtory"}

# Graph Edge Count
graph_edges_total{service="raphtory"}

# Query Duration
histogram_quantile(0.95, rate(graph_query_duration_seconds_bucket[5m]))

# Memory Usage (Python)
python_memory_bytes{service="raphtory"}
```

### Infrastructure Metrics

```promql
# Pod CPU Usage
rate(container_cpu_usage_seconds_total{namespace="stablerisk"}[5m])

# Pod Memory Usage
container_memory_working_set_bytes{namespace="stablerisk"}

# Pod Restarts
kube_pod_container_status_restarts_total{namespace="stablerisk"}

# Disk Usage (PostgreSQL)
kubelet_volume_stats_used_bytes{namespace="stablerisk",persistentvolumeclaim="postgres-storage-postgres-0"}
```

---

## Setting Up Prometheus

### Install Prometheus Operator

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

### ServiceMonitor for StableRisk

```yaml
# stablerisk-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: stablerisk-api
  namespace: stablerisk
  labels:
    app: api
spec:
  selector:
    matchLabels:
      app: api
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s

---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: stablerisk-monitor
  namespace: stablerisk
  labels:
    app: monitor
spec:
  selector:
    matchLabels:
      app: monitor
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

Apply ServiceMonitors:

```bash
kubectl apply -f stablerisk-servicemonitor.yaml
```

---

## Grafana Dashboards

### Access Grafana

```bash
# Port-forward Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Get admin password
kubectl get secret -n monitoring prometheus-grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d
```

Open http://localhost:3000 (admin/password)

### StableRisk Overview Dashboard

Create a dashboard with the following panels:

**1. Transaction Ingestion Rate**
```promql
rate(transactions_processed_total{service="monitor"}[5m])
```

**2. Outlier Detection Rate**
```promql
rate(outliers_detected_total[5m])
```

**3. API Request Rate by Endpoint**
```promql
sum(rate(http_requests_total{service="api"}[5m])) by (endpoint)
```

**4. API Error Rate**
```promql
sum(rate(http_requests_total{service="api",status=~"5.."}[5m])) /
sum(rate(http_requests_total{service="api"}[5m])) * 100
```

**5. Active WebSocket Connections**
```promql
websocket_connections_active{service="api"}
```

**6. Database Connection Pool Usage**
```promql
db_connections_active / db_connections_max * 100
```

**7. Pod Memory Usage**
```promql
sum(container_memory_working_set_bytes{namespace="stablerisk"}) by (pod)
```

**8. Pod CPU Usage**
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="stablerisk"}[5m])) by (pod)
```

**9. TronGrid Connection Status**
```promql
trongrid_connection_status{service="monitor"}
```

**10. Response Time (p95)**
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service="api"}[5m]))
```

### Export Dashboard

Save dashboard JSON:

```bash
# Get dashboard UID from Grafana
DASHBOARD_UID="stablerisk-overview"

# Export via API
curl -u admin:password \
  http://localhost:3000/api/dashboards/uid/${DASHBOARD_UID} | \
  jq '.dashboard' > stablerisk-dashboard.json
```

Store in `docs/monitoring/grafana-dashboards/`

---

## Alerting Rules

### PrometheusRule Configuration

```yaml
# stablerisk-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: stablerisk-alerts
  namespace: stablerisk
  labels:
    prometheus: kube-prometheus
spec:
  groups:
    - name: stablerisk.rules
      interval: 30s
      rules:
        # High Error Rate
        - alert: HighAPIErrorRate
          expr: |
            sum(rate(http_requests_total{service="api",status=~"5.."}[5m])) /
            sum(rate(http_requests_total{service="api"}[5m])) > 0.05
          for: 5m
          labels:
            severity: critical
            component: api
          annotations:
            summary: "High API error rate detected"
            description: "API error rate is {{ $value | humanizePercentage }} (threshold: 5%)"

        # Service Down
        - alert: ServiceDown
          expr: up{namespace="stablerisk"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "Service {{ $labels.pod }} is down"
            description: "Service has been down for more than 2 minutes"

        # High Memory Usage
        - alert: HighMemoryUsage
          expr: |
            container_memory_working_set_bytes{namespace="stablerisk"} /
            container_spec_memory_limit_bytes{namespace="stablerisk"} > 0.9
          for: 5m
          labels:
            severity: warning
            component: infrastructure
          annotations:
            summary: "High memory usage on {{ $labels.pod }}"
            description: "Memory usage is {{ $value | humanizePercentage }} of limit"

        # Database Connection Pool Exhausted
        - alert: DatabaseConnectionPoolExhausted
          expr: db_connections_active / db_connections_max > 0.9
          for: 5m
          labels:
            severity: critical
            component: database
          annotations:
            summary: "Database connection pool nearly exhausted"
            description: "{{ $value | humanizePercentage }} of connections in use"

        # TronGrid Connection Lost
        - alert: TronGridDisconnected
          expr: trongrid_connection_status{service="monitor"} == 0
          for: 2m
          labels:
            severity: critical
            component: monitor
          annotations:
            summary: "TronGrid connection lost"
            description: "Monitor service cannot connect to TronGrid"

        # No Transactions Received
        - alert: NoTransactionsReceived
          expr: rate(transactions_processed_total{service="monitor"}[10m]) == 0
          for: 15m
          labels:
            severity: warning
            component: monitor
          annotations:
            summary: "No transactions received from TronGrid"
            description: "Monitor has not processed any transactions in 15 minutes"

        # High Outlier Detection Rate
        - alert: HighOutlierRate
          expr: rate(outliers_detected_total[5m]) > 10
          for: 10m
          labels:
            severity: warning
            component: detection
          annotations:
            summary: "Unusually high outlier detection rate"
            description: "Detecting {{ $value }} outliers per second (may indicate issue)"

        # Slow API Responses
        - alert: SlowAPIResponses
          expr: |
            histogram_quantile(0.95,
              rate(http_request_duration_seconds_bucket{service="api"}[5m])
            ) > 2
          for: 5m
          labels:
            severity: warning
            component: api
          annotations:
            summary: "Slow API response times"
            description: "95th percentile response time is {{ $value }}s (threshold: 2s)"

        # Pod Restart Loop
        - alert: PodRestartLoop
          expr: rate(kube_pod_container_status_restarts_total{namespace="stablerisk"}[15m]) > 0
          for: 15m
          labels:
            severity: warning
            component: infrastructure
          annotations:
            summary: "Pod {{ $labels.pod }} is restarting"
            description: "Pod has restarted {{ $value }} times in 15 minutes"

        # Disk Space Low
        - alert: DiskSpaceLow
          expr: |
            kubelet_volume_stats_available_bytes{namespace="stablerisk"} /
            kubelet_volume_stats_capacity_bytes{namespace="stablerisk"} < 0.1
          for: 5m
          labels:
            severity: critical
            component: infrastructure
          annotations:
            summary: "Low disk space on {{ $labels.persistentvolumeclaim }}"
            description: "Only {{ $value | humanizePercentage }} disk space remaining"
```

Apply alerts:

```bash
kubectl apply -f stablerisk-alerts.yaml
```

---

## AlertManager Configuration

### Configure Alert Routing

```yaml
# alertmanager-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-prometheus-kube-prometheus-alertmanager
  namespace: monitoring
type: Opaque
stringData:
  alertmanager.yaml: |
    global:
      resolve_timeout: 5m
      slack_api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'

    route:
      group_by: ['alertname', 'cluster', 'service']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 12h
      receiver: 'default'
      routes:
        - match:
            severity: critical
          receiver: 'critical-alerts'
          continue: true
        - match:
            component: database
          receiver: 'database-team'

    receivers:
      - name: 'default'
        slack_configs:
          - channel: '#stablerisk-alerts'
            title: '{{ .GroupLabels.alertname }}'
            text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

      - name: 'critical-alerts'
        slack_configs:
          - channel: '#stablerisk-critical'
            title: ':rotating_light: CRITICAL: {{ .GroupLabels.alertname }}'
            text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        pagerduty_configs:
          - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'

      - name: 'database-team'
        email_configs:
          - to: 'dba@yourdomain.com'
            from: 'alertmanager@stablerisk.com'
            smarthost: 'smtp.gmail.com:587'
            auth_username: 'alerts@yourdomain.com'
            auth_password: 'app-password'

    inhibit_rules:
      - source_match:
          severity: 'critical'
        target_match:
          severity: 'warning'
        equal: ['alertname', 'cluster', 'service']
```

Apply configuration:

```bash
kubectl apply -f alertmanager-config.yaml
kubectl rollout restart statefulset/alertmanager-prometheus-kube-prometheus-alertmanager -n monitoring
```

---

## Log Aggregation

### Loki Setup (Optional)

```bash
# Install Loki stack
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set promtail.enabled=true \
  --set grafana.enabled=false
```

### Query Logs in Grafana

Add Loki as a data source, then use LogQL:

```logql
# API Errors
{namespace="stablerisk", app="api"} |= "level=error"

# Monitor Transaction Processing
{namespace="stablerisk", app="monitor"} |= "Transaction stored"

# Authentication Failures
{namespace="stablerisk", app="api"} |= "authentication failed"

# Slow Queries
{namespace="stablerisk", app="api"} | json | duration > 1s
```

---

## Health Checks

### Kubernetes Probes

All services have configured probes:

```bash
# Check probe status
kubectl describe pod -n stablerisk <pod-name> | grep -A 10 "Liveness\|Readiness"
```

### Manual Health Checks

```bash
# API Health
curl -f https://stablerisk.yourdomain.com/health

# API Readiness
curl -f https://stablerisk.yourdomain.com/readiness

# Internal Service Health
kubectl exec -n stablerisk -l app=api -- curl http://localhost:8080/health
```

---

## SLOs and SLIs

### Service Level Indicators (SLIs)

1. **Availability**: % of successful requests (non-5xx)
2. **Latency**: 95th percentile response time < 500ms
3. **Throughput**: Transactions processed per second
4. **Error Rate**: < 1% of requests result in errors

### Service Level Objectives (SLOs)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Availability | 99.9% | Monthly |
| API Latency (p95) | < 500ms | Weekly |
| API Latency (p99) | < 2s | Weekly |
| Error Rate | < 0.1% | Daily |
| Transaction Lag | < 30s | Hourly |

### SLO Dashboard

```promql
# Availability (last 7 days)
sum(rate(http_requests_total{service="api",status!~"5.."}[7d])) /
sum(rate(http_requests_total{service="api"}[7d]))

# Error Budget Remaining
1 - (
  (1 - (sum(rate(http_requests_total{service="api",status!~"5.."}[7d])) /
        sum(rate(http_requests_total{service="api"}[7d]))))
  / (1 - 0.999)
)
```

---

## Incident Response

### Severity Levels

- **P0 (Critical)**: Service down, data loss, security breach
- **P1 (High)**: Partial outage, significant degradation
- **P2 (Medium)**: Minor feature impairment
- **P3 (Low)**: Cosmetic issues, non-urgent bugs

### Incident Response Checklist

When an alert fires:

1. **Acknowledge Alert** (within 5 minutes)
   ```bash
   # Silence alert if investigating
   amtool silence add alertname=HighAPIErrorRate duration=1h
   ```

2. **Assess Impact**
   - Check dashboard for scope
   - Verify affected users/services
   - Determine severity

3. **Investigate**
   ```bash
   # Check recent events
   kubectl get events -n stablerisk --sort-by='.lastTimestamp' | tail -20

   # Review logs
   kubectl logs -n stablerisk -l app=api --tail=200 | grep -i error

   # Check metrics
   # (Use Grafana dashboard)
   ```

4. **Mitigate**
   - Scale up if capacity issue
   - Restart pods if crash loop
   - Rollback if deployment issue

5. **Resolve**
   - Apply permanent fix
   - Verify metrics return to normal
   - Document in incident report

6. **Post-Mortem** (for P0/P1)
   - Root cause analysis
   - Timeline of events
   - Action items to prevent recurrence

---

## Performance Monitoring

### APM Integration

For detailed performance tracing, integrate with an APM:

**OpenTelemetry Setup:**

```yaml
# otel-collector.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: stablerisk
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317

    processors:
      batch:
        timeout: 10s
        send_batch_size: 1024

    exporters:
      jaeger:
        endpoint: jaeger-collector:14250
        tls:
          insecure: true

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [jaeger]
```

---

## Best Practices

1. **Monitor the Four Golden Signals**
   - Latency
   - Traffic
   - Errors
   - Saturation

2. **Set Meaningful Alerts**
   - Avoid alert fatigue
   - Alert on symptoms, not causes
   - Use appropriate thresholds

3. **Regular Review**
   - Weekly dashboard review
   - Monthly SLO assessment
   - Quarterly alert tuning

4. **Documentation**
   - Document all custom metrics
   - Maintain runbooks for alerts
   - Keep dashboard annotations

5. **Capacity Planning**
   - Monitor trends over time
   - Plan for growth
   - Load test regularly

---

## Troubleshooting Monitoring

### Prometheus Not Scraping

```bash
# Check ServiceMonitor
kubectl get servicemonitor -n stablerisk

# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Visit http://localhost:9090/targets
```

### Missing Metrics

```bash
# Test metrics endpoint
kubectl exec -n stablerisk -l app=api -- curl http://localhost:9090/metrics
```

### Grafana Dashboard Empty

- Check data source configuration
- Verify time range
- Check Prometheus query syntax
- Ensure ServiceMonitor is configured

---

## References

- Prometheus Documentation: https://prometheus.io/docs/
- Grafana Documentation: https://grafana.com/docs/
- Google SRE Book: https://sre.google/books/
