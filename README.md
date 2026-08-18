# k8s-pod-autopsy

**Read-only Kubernetes pod triage that synthesizes crash / OOM / waiting / saturation root-cause into a token-thrifty markdown digest.** Works as a CLI *and* as an MCP server, so your AI assistant can root-cause a pod without you pasting a wall of `kubectl describe` output.

![autopsy in action: scanning a namespace, then root-causing an OOMKilled and a CrashLoopBackOff pod](docs/demo/autopsy.gif)

> Real output against a live cluster — a namespace scan, an OOMKilled pod, and a crash-looping worker (with its actual error logs). Recorded with [VHS](https://github.com/charmbracelet/vhs); regenerate via `vhs docs/demo/autopsy.tape`.

```
autopsy triage web-1 -n demo
```

```markdown
# Pod Autopsy: demo/web-1

**Verdict:** Container "web": OOMKilled — Container exceeded its own memory limit. Raise the limit or investigate a memory leak.

- Phase: Running
- Signal: OOMKilled
- Container: web (app), restarts: 6
- Exit code: 137 (SIGKILL — OOM or forced kill)

## Why OOM
Container exceeded its own memory limit. Raise the limit or investigate a memory leak.

## Logs (previous)
...
```

## 🔒 100% Read-Only by design

This tool **cannot** mutate your cluster. No code path calls create / update / patch / delete / apply / exec / attach / port-forward — only `get` / `list` / `watch` on `pods`, `pods/log`, `events`, and `metrics.k8s.io`. Ship it with a least-privilege role:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: { name: k8s-pod-autopsy-readonly }
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "events"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["metrics.k8s.io"]
    resources: ["pods"]
    verbs: ["get", "list"]
```

A CI test (`deploy/rbac_test.go`) fails the build if a write verb ever appears in that manifest.

## Why not just `kubectl describe` + `logs`?

Because that's a lot of tokens for an LLM to chew through — and most of it is noise. `autopsy` distills the same signal into a fraction of the tokens:

| Input | Tokens (cl100k BPE) |
|---|---|
| Raw `kubectl describe` + `logs --previous` + `get events` | **1475** |
| `autopsy` digest for the same pod | **322** |
| **Saved** | **~78%** |

> Measured with the cl100k_base BPE (`hack/bench-tokens.go`), which is exact for GPT models and a close proxy for Claude. Reproduce it yourself: `go run -tags tools ./hack/bench-tokens.go`.

## What it detects

- **OOMKilled** — and *why*: own-limit breach vs. node memory pressure, with a concrete next step.
- **Crashing** — non-zero exit codes translated to plain English (137 = SIGKILL, 143 = SIGTERM/liveness, 127 = bad entrypoint, …), plus the **previous** container's logs (auto-falls back to current).
- **Waiting** — `ImagePullBackOff`, config errors, and other non-starting states (no logs to fetch — we say so instead of erroring).
- **Init & ephemeral containers** — an init-container crash or a failed `kubectl debug` container is surfaced, not hidden behind the app container.
- **Resource saturation** — live usage vs. requests/limits, flagging memory ≥90% of limit as an OOM risk and possible CPU throttling (honestly framed as a snapshot).
- **Cluster scan** — `crashloops` lists every CrashLoopBackOff / OOMKilled / high-restart pod, classified by reason.

Logs are tailed (50 lines), timestamp-normalized, and de-duplicated (`… (×N)`) so repeated stack traces don't blow your token budget. Every external call degrades gracefully — a missing metrics-server, a not-found pod, or a forbidden namespace produces a clear message, never a stack trace.

## Install

```bash
# Homebrew
brew install behzadashrafiani/tap/k8s-pod-autopsy

# Go
go install github.com/behzadashrafiani/k8s-pod-autopsy/cmd/autopsy@latest
```

## CLI usage

```bash
autopsy triage <pod> -n <namespace>       # full auto-routed autopsy
autopsy saturation <pod> -n <namespace>   # resource saturation table
autopsy crashloops [-n <namespace>]       # scan a namespace (or all) for crashloops
autopsy mcp                               # start the stdio MCP server
```

Global flags: `--kubeconfig`, `--context`, `--no-color` (color auto-disables when stdout isn't a TTY). In every mode **stdout carries only the digest** (or JSON-RPC in `mcp` mode); all diagnostics go to stderr.

## MCP usage (Claude Desktop / Claude Code)

Add to your MCP config:

```json
{
  "mcpServers": {
    "pod-autopsy": { "command": "autopsy", "args": ["mcp"] }
  }
}
```

Two tools are exposed:

- **`autopsy_pod`** — `{ namespace, pod_name }` → root-cause digest for one pod.
- **`find_crashlooping_pods`** — `{ namespace? }` → cluster/namespace crashloop scan (empty namespace = all).

## License

Apache-2.0 — see [LICENSE](LICENSE).
