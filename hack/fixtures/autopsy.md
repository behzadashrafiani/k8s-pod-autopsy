# Pod Autopsy: demo/web-1

**Verdict:** Container "web": OOMKilled — Container exceeded its own memory limit. Raise the limit or investigate a memory leak.

- Phase: Running
- Signal: OOMKilled
- Container: web (app), restarts: 6
- Exit code: 137 (SIGKILL — OOM or forced kill)

## Why OOM
Container exceeded its own memory limit. Raise the limit or investigate a memory leak.

## Logs (previous)
**Detected errors:**
```
2026-08-17T10:05:12.002Z ERROR runtime: out of memory: cannot allocate 8192-byte block
```
```
2026-08-17T10:04:52.019Z INFO  http server listening addr=:8080
2026-08-17T10:05:11.884Z WARN  memory usage high heap_mb=248 rss_mb=255
2026-08-17T10:05:12.002Z ERROR runtime: out of memory: cannot allocate 8192-byte block
```

## Warning Events
- **BackOff** (×14): Back-off restarting failed container web in pod web-1_demo(6f7d9c5b)
- **Unhealthy** (×9): Liveness probe failed: Get "http://10.0.3.201:8080/healthz": dial tcp 10.0.3.201:8080: connect: connection refused
