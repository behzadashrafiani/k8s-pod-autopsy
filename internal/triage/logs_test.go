package triage

import (
	"reflect"
	"testing"
)

func TestNormalizeLogLine(t *testing.T) {
	a := normalizeLogLine("2026-08-17T10:00:01Z panic: boom")
	b := normalizeLogLine("2026-08-17T10:00:02Z panic: boom")
	if a != b {
		t.Fatalf("RFC3339 timestamps should normalize equal: %q vs %q", a, b)
	}
	c := normalizeLogLine("[10:00:01] starting")
	d := normalizeLogLine("[10:00:59] starting")
	if c != d {
		t.Fatalf("bracket clock stamps should normalize equal: %q vs %q", c, d)
	}
}

func TestDedupLines(t *testing.T) {
	in := []string{
		"2026-08-17T10:00:01Z panic: boom",
		"2026-08-17T10:00:02Z panic: boom",
		"2026-08-17T10:00:03Z panic: boom",
		"done",
	}
	got := dedupLines(in)
	// First occurrence text preserved; collapsed with count.
	want := []string{"2026-08-17T10:00:01Z panic: boom (×3)", "done"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDetectErrors(t *testing.T) {
	in := []string{"starting up", "ERROR: db unreachable", "panic: nil deref", "ok"}
	got := detectErrors(in)
	if len(got) != 2 {
		t.Fatalf("expected 2 error lines, got %v", got)
	}
}

func TestLogBodyUnavailable(t *testing.T) {
	// The kubelet returns HTTP 200 with this body (not an error status) when a
	// previous container's logs are no longer on disk. It must be treated as a
	// miss so triage falls back to current logs.
	cases := map[string]bool{
		"unable to retrieve container logs for containerd://3f801cacd07f": true,
		"unable to retrieve container logs for docker://abc123":           true,
		"":      true,
		"   \n": true,
		"booting worker\nFATAL: cannot connect to db": false,
		"a normal log line mentioning container logs": false,
	}
	for body, want := range cases {
		if got := logBodyUnavailable(body); got != want {
			t.Errorf("logBodyUnavailable(%q) = %v, want %v", body, got, want)
		}
	}
}
