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
