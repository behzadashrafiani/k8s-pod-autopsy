package triage

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const tailLines = 50

var (
	tsPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\s*`), // RFC3339
		regexp.MustCompile(`^\[?\d{2}:\d{2}:\d{2}(\.\d+)?\]?\s*`),                                     // clock / [clock]
	}
	errPattern = regexp.MustCompile(`(?i)(error|panic|fatal|exception|traceback|stacktrace|fail)`)
)

// normalizeLogLine strips a leading timestamp so lines that differ only by time
// compare equal. The original text is what we display; this is the compare key.
func normalizeLogLine(line string) string {
	out := line
	for _, re := range tsPatterns {
		out = re.ReplaceAllString(out, "")
	}
	return strings.TrimSpace(out)
}

// dedupLines collapses consecutive lines equal after normalization, keeping the
// first occurrence's original text and appending " (×N)" when N > 1.
func dedupLines(lines []string) []string {
	var out []string
	i := 0
	for i < len(lines) {
		key := normalizeLogLine(lines[i])
		n := 1
		for i+n < len(lines) && normalizeLogLine(lines[i+n]) == key {
			n++
		}
		if n > 1 {
			out = append(out, fmt.Sprintf("%s (×%d)", lines[i], n))
		} else {
			out = append(out, lines[i])
		}
		i += n
	}
	return out
}

func detectErrors(lines []string) []string {
	var out []string
	for _, l := range lines {
		if errPattern.MatchString(l) {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}

// processLogs tails, categorizes, and dedups raw log text.
func processLogs(raw string) (errs, lines []string) {
	all := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(all) > tailLines {
		all = all[len(all)-tailLines:]
	}
	return detectErrors(all), dedupLines(all)
}

// fetchContainerLog reads a container's log tail. previous=true reads the
// crashed instance's logs. Thin by design; parsing is tested via processLogs.
func fetchContainerLog(ctx context.Context, kube kubernetes.Interface, ns, pod, container string, previous bool) (string, error) {
	tl := int64(tailLines)
	req := kube.CoreV1().Pods(ns).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
		TailLines: &tl,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
