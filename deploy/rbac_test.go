package deploy

import (
	"os"
	"strings"
	"testing"
)

func TestRBACIsReadOnly(t *testing.T) {
	b, err := os.ReadFile("rbac-readonly.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"create", "update", "patch", "delete", "deletecollection", "*"} {
		if strings.Contains(string(b), "\""+forbidden+"\"") {
			t.Fatalf("RBAC must be read-only, found verb %q", forbidden)
		}
	}
}
