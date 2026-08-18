# Contributing

Thanks for your interest in improving `k8s-pod-autopsy`!

## Ground rules

1. **Read-only is non-negotiable.** No code path may call create / update / patch / delete / apply / exec / attach / port-forward. Only `get` / `list` / `watch` on `pods`, `pods/log`, `events`, and `metrics.k8s.io`. The RBAC manifest in `deploy/` is guarded by a test — keep it that way.
2. **stdout is sacred.** In `mcp` mode only JSON-RPC goes to stdout; in CLI mode only the digest. All logs and diagnostics use `log/slog` to **stderr**.
3. **Never panic on missing resources.** Every external call must degrade to a typed partial result with a human-readable reason — never a crash or stack trace to the user.
4. **Triage returns structs, rendering makes markdown.** Keep `internal/triage` free of presentation concerns; all markdown lives in `internal/render`.

## Development

```bash
go build ./...
go test ./...
go run -tags tools ./hack/bench-tokens.go   # token benchmark (offline)
```

Tests run against fake clientsets — no cluster required. Please add a test with every behavior change; new triage logic should be exercised via a pure function plus a fake-client path where relevant.

## Commits & PRs

- Conventional-commit style messages (`feat(triage): …`, `fix(render): …`).
- Keep PRs focused; explain the "why" in the description.
- CI must be green (`go test ./...` + `golangci-lint`).

## Regenerating the README demo GIF

The GIF at `docs/demo/autopsy.gif` is recorded from **real** output against a
throwaway [kind](https://kind.sigs.k8s.io/) cluster — never hand-faked. To
regenerate it you need [`vhs`](https://github.com/charmbracelet/vhs) and `kind`:

```bash
kind create cluster --name autopsy-demo
kubectl apply -f docs/demo/workloads.yaml   # OOM / crashloop / imagepull pods
go build -o autopsy ./cmd/autopsy
# wait ~1min for pods to reach their failure states, then:
vhs docs/demo/autopsy.tape
kind delete cluster --name autopsy-demo
```

## Regenerating the tiktoken vocab cache

The benchmark reads a committed cl100k_base vocab so CI stays offline. If it is ever missing:

```bash
TIKTOKEN_CACHE_DIR=hack/vocab go run -tags tools ./hack/warm-cache.go
```
