package main

import (
	"flag"
	"testing"
)

// parseInterspersed must accept flags and positionals in any order, matching
// the syntax the README advertises (e.g. `triage web-1 -n demo`), which the
// stdlib flag.Parse alone does not support (it stops at the first positional).
func TestParseInterspersed(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantNs   string
		wantArgs []string
	}{
		{"flag after positional", []string{"web-1", "-n", "demo"}, "demo", []string{"web-1"}},
		{"flag before positional", []string{"-n", "demo", "web-1"}, "demo", []string{"web-1"}},
		{"equals form after positional", []string{"web-1", "-n=demo"}, "demo", []string{"web-1"}},
		{"positional only", []string{"web-1"}, "", []string{"web-1"}},
		{"flags only", []string{"-n", "demo"}, "demo", nil},
		{"multiple positionals interspersed", []string{"web-1", "-n", "demo", "extra"}, "demo", []string{"web-1", "extra"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			ns := fs.String("n", "", "namespace")
			got, err := parseInterspersed(fs, tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *ns != tc.wantNs {
				t.Errorf("ns = %q, want %q", *ns, tc.wantNs)
			}
			if len(got) != len(tc.wantArgs) {
				t.Fatalf("positionals = %v, want %v", got, tc.wantArgs)
			}
			for i := range got {
				if got[i] != tc.wantArgs[i] {
					t.Errorf("positional[%d] = %q, want %q", i, got[i], tc.wantArgs[i])
				}
			}
		})
	}
}
