package mcp

import (
	"context"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/k8s"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewServer(clients *k8s.Clients) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "k8s-pod-autopsy", Version: "v0.1.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "autopsy_pod",
		Description: "Root-cause a pod: auto-detects OOM/crash/waiting/saturation, includes previous-container logs and warning events as a compact markdown digest.",
	}, makeAutopsyHandler(clients))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "find_crashlooping_pods",
		Description: "List pods that are CrashLoopBackOff, OOMKilled, or exceed 5 restarts, classified by reason. namespace optional (empty = all).",
	}, makeCrashLoopHandler(clients))
	return s
}

func Run(ctx context.Context, clients *k8s.Clients) error {
	return NewServer(clients).Run(ctx, &mcp.StdioTransport{})
}
