# Assignment 3: Monitoring and Alerting Setup

## Architecture

This setup uses the existing medical appointment system as the monitored application.

Components:

- `app`: Go binary running `doctor-service` and `appointment-service`
- `mongo`: database used by the application
- `prometheus`: metrics collection and alert evaluation
- `grafana`: dashboards
- `node-exporter`: host metrics

Start the stack:

```bash
docker compose --profile observability up --build -d
```

## Prometheus Configuration

Prometheus configuration:

- [prometheus.yml](/Users/myrzanizimbetov/Desktop/med-go/monitoring/prometheus/prometheus.yml)

Alert rules:

- [alerts.yml](/Users/myrzanizimbetov/Desktop/med-go/monitoring/prometheus/alerts.yml)

Targets scraped:

- `doctor-service`
- `appointment-service`
- `node-exporter`
- `prometheus`

## Metrics Collected

Application metrics:

- request rate
- 5xx error ratio
- p95 request latency

Infrastructure metric:

- node memory usage

Note:

- on Docker Desktop, `node-exporter` reflects the Docker VM host view rather than the full macOS host

Metrics endpoints:

- `http://localhost:8081/metrics`
- `http://localhost:8082/metrics`

## Dashboard

Grafana dashboard:

- [med-go-observability.json](/Users/myrzanizimbetov/Desktop/med-go/monitoring/grafana/dashboards/med-go-observability.json)

Panels:

1. `Request Rate by Service`
Question answered:
Are requests reaching the services right now?

2. `5xx Error Ratio by Service`
Question answered:
Are failures happening at a level users would notice?

3. `P95 HTTP Latency by Service`
Question answered:
Are requests getting slow even when the service is still available?

4. `Node Memory Usage`
Question answered:
Is host memory pressure a likely explanation for application behavior?

## Alerting

Alerts configured:

1. `DoctorServiceDown`
- threshold: target `up == 0`
- time condition: `for 1m`
- meaning: Prometheus cannot scrape the doctor service

2. `AppointmentServiceDown`
- threshold: target `up == 0`
- time condition: `for 1m`
- meaning: Prometheus cannot scrape the appointment service

3. `HighAppointmentAPIErrorRate`
- threshold: 5xx responses above `5%`
- time condition: `for 5m`
- meaning: the appointment API is failing persistently, not just spiking

## Interpretation

What the metrics tell me:

- request rate shows whether traffic exists and whether services are active
- error ratio shows whether users are getting failures
- p95 latency shows user-facing slowness before total outage
- memory usage adds infrastructure context

Most useful signal:

- `P95 HTTP Latency`

Reason:

- a service can still be up while the user experience is already degraded

Is the alert too sensitive or too weak:

- the `service down` alerts are conservative and appropriate for a demo
- the `5xx > 5% for 5m` alert is production-oriented and may need tuning in a low-traffic environment

What I would add in a real system:

- MongoDB exporter metrics
- route-level saturation metrics
- CPU throttling and disk pressure panels
- Alertmanager or Grafana notification routing
- SLO-based alerts

## Demo Steps

1. Start the stack:

```bash
docker compose --profile observability up --build -d
```

2. Open:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

3. In Prometheus, verify `Status -> Targets` shows `UP`.

4. Generate traffic:

```bash
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8081/doctors
curl http://localhost:8082/appointments
```

5. In Grafana, open `Med Go Observability`.

6. To simulate the `service down` alert:

```bash
docker compose stop app
```

Wait at least 1 minute, then verify the alert fires.

Bring it back:

```bash
docker compose start app
```
