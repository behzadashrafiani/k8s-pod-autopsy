package mcp

import (
	"context"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/k8s"
	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/render"
	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/triage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AutopsyInput struct {
	Namespace string `json:"namespace" jsonschema:"the pod's namespace"`
	PodName   string `json:"pod_name" jsonschema:"the pod name to autopsy"`
}

type CrashLoopInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"namespace to scan; empty scans all namespaces"`
}

type MarkdownOutput struct {
	Markdown string `json:"markdown" jsonschema:"the synthesized markdown digest"`
}

func makeAutopsyHandler(clients *k8s.Clients) func(context.Context, *mcp.CallToolRequest, AutopsyInput) (*mcp.CallToolResult, MarkdownOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in AutopsyInput) (*mcp.CallToolResult, MarkdownOutput, error) {
		rep, err := triage.AutopsyPod(ctx, clients.Kube, clients.Metrics, in.Namespace, in.PodName)
		if err != nil {
			return nil, MarkdownOutput{}, err
		}
		return nil, MarkdownOutput{Markdown: render.PodReport(rep, render.PlainStyler())}, nil
	}
}

func makeCrashLoopHandler(clients *k8s.Clients) func(context.Context, *mcp.CallToolRequest, CrashLoopInput) (*mcp.CallToolResult, MarkdownOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CrashLoopInput) (*mcp.CallToolResult, MarkdownOutput, error) {
		rep, err := triage.FindCrashLooping(ctx, clients.Kube, in.Namespace)
		if err != nil {
			return nil, MarkdownOutput{}, err
		}
		return nil, MarkdownOutput{Markdown: render.CrashLoops(rep, render.PlainStyler())}, nil
	}
}
