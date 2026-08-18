package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/k8s"
	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/mcp"
	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/render"
	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/triage"
	"golang.org/x/term"
)

func main() {
	// All logs go to stderr; stdout is reserved for JSON-RPC / digest.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	ns := fs.String("n", "", "namespace")
	kubeconfig := fs.String("kubeconfig", "", "path to kubeconfig")
	kubeContext := fs.String("context", "", "kube context")
	noColor := fs.Bool("no-color", false, "disable ANSI color")
	_ = fs.Parse(os.Args[2:])

	styler := chooseStyler(*noColor)
	ctx := context.Background()

	if cmd == "mcp" {
		clients := mustClients(*kubeconfig, *kubeContext)
		if err := mcp.Run(ctx, clients); err != nil {
			slog.Error("mcp server exited", "err", err)
			os.Exit(1)
		}
		return
	}

	switch cmd {
	case "triage":
		clients := mustClients(*kubeconfig, *kubeContext)
		name := fs.Arg(0)
		requirePod(name, *ns)
		rep, err := triage.AutopsyPod(ctx, clients.Kube, clients.Metrics, *ns, name)
		exitOnErr(err)
		fmt.Println(render.PodReport(rep, styler))
	case "saturation":
		clients := mustClients(*kubeconfig, *kubeContext)
		name := fs.Arg(0)
		requirePod(name, *ns)
		rep := triage.CheckSaturation(ctx, clients.Kube, clients.Metrics, *ns, name)
		fmt.Println(render.Saturation(rep, styler))
	case "crashloops":
		clients := mustClients(*kubeconfig, *kubeContext)
		rep, err := triage.FindCrashLooping(ctx, clients.Kube, *ns)
		exitOnErr(err)
		fmt.Println(render.CrashLoops(rep, styler))
	default:
		usage()
		os.Exit(2)
	}
}

func mustClients(kubeconfig, kubeContext string) *k8s.Clients {
	c, err := k8s.NewClients(kubeconfig, kubeContext)
	exitOnErr(err)
	return c
}

func chooseStyler(noColor bool) render.Styler {
	if noColor || !term.IsTerminal(int(os.Stdout.Fd())) {
		return render.PlainStyler()
	}
	return render.ColorStyler()
}

func requirePod(name, ns string) {
	if name == "" || ns == "" {
		fmt.Fprintln(os.Stderr, "error: pod name (arg) and -n <namespace> are required")
		os.Exit(2)
	}
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `k8s-pod-autopsy — read-only pod triage
usage:
  autopsy mcp                       start stdio MCP server
  autopsy triage <pod> -n <ns>      autopsy a pod
  autopsy saturation <pod> -n <ns>  resource saturation
  autopsy crashloops [-n <ns>]      list crashlooping pods
flags: --kubeconfig --context --no-color`)
}
