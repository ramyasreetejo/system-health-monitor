# Health Monitor (Go)

A simple polling-based service health monitor written in Go. This project demonstrates dynamic service registration, bounded-concurrency polling, readiness vs health evaluation, and graceful shutdown. The code is intentionally structured and commented for clarity and learning.

---

## Overview

The monitor periodically polls registered services over HTTP and evaluates their state based on reachability, readiness, and error metrics. Services register themselves at runtime; there is no static configuration or heartbeat mechanism.

Key properties:

* Polling-based monitoring (no push / heartbeat)
* Self-timed polling cycles (no overlapping cycles)
* Bounded worker pool (maximum 5 concurrent polls)
* Snapshot-based polling per cycle
* Clean separation of readiness and health
* Graceful shutdown using context cancellation

---

## Project Structure

```
health-monitor/
├── go.mod
├── monitor/
│   ├── main.go
│   └── internal/
│       ├── models.go      # Core data models
│       ├── store.go       # Thread-safe service store
│       ├── monitor.go     # Polling engine and worker pool
│       └── handlers.go    # HTTP handlers
└── services/
    ├── svc1/
    │   └── main.go        # Example healthy service
    └── svc2/
        └── main.go        # Example degrading service
```

---

## Service Registration

Services register themselves dynamically using:

```
POST /register
```

Example payload:

```json
{
  "id": "svc1",
  "url": "http://localhost:9001",
  "attributes": {
    "service": "user",
    "env": "prod"
  }
}
```

The monitor stores services in a thread-safe in-memory store. Attributes provided during registration are merged with attributes returned by the `/health` endpoint.

---

## Polling Model

The monitor runs a continuous loop with a fixed polling interval (default: 10 seconds).

For each cycle:

1. A snapshot of currently registered services is taken
2. A per-cycle job queue is created
3. A fixed worker pool (max 5 workers) consumes jobs
4. Each service is polled exactly once in the cycle
5. Metrics and health state are updated

If a polling cycle takes longer than the configured interval, the next cycle starts immediately. There are no overlapping cycles and no persistent backlog.

---

## Health and Readiness Semantics

Readiness and health are treated as separate concepts:

### Readiness

* `ready = true` if the service responds with HTTP 200
* `ready = false` if the service is unreachable or returns a non-200 response

### Health

| Condition                 | Health    |
| ------------------------- | --------- |
| Network failure / timeout | dead      |
| HTTP non-200              | unhealthy |
| 200 OK + high error rate  | degraded  |
| 200 OK + low error rate   | healthy   |

This mirrors real-world systems such as Kubernetes readiness and liveness probes.

---

## Metrics Endpoint

```
GET /metrics
```

Returns JSON containing all registered services with:

* readiness
* health status
* uptime
* request count
* error count
* error rate
* dynamic attributes
* last checked age

---

## Example Services

Two example services are included:

* **svc1**: always healthy
* **svc2**: periodically increases error count to demonstrate degradation

Each service:

* exposes `/health`
* tracks its own metrics
* registers itself with the monitor on startup

---

## Running the Project

Start the monitor:

```bash
go run monitor/main.go
```

Start the example services:

```bash
go run services/svc1/main.go
go run services/svc2/main.go
```

View metrics:

```
http://localhost:8080/metrics
```

---

## Design Rationale

* Per-cycle queues are used for coordination, not buffering
* No persistent queue avoids stale health checks
* Snapshot-based polling avoids race conditions with dynamic registration
* Self-timed scheduling adapts naturally to load

The design is intentionally simple but aligns with production monitoring principles used by systems like Prometheus.

---

## Possible Extensions

* Prometheus exporter
* Persistent storage (SQLite / BoltDB)
* Per-service polling intervals
* gRPC health checks
* Alerting and notification hooks

---

## License

This project is licensed under the MIT License.  
See the [LICENSE](LICENSE) file for details.
