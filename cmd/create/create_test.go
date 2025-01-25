// file: cmd/create/create_test.go

package create

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "test.dsk")

	if err := Create(outPath); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("Output file not created: %v", err)
	}

	nestedPath := filepath.Join(tmpDir, "sub", "nested.dsk")
	if err := Create(nestedPath); err != nil {
		t.Errorf("Create with nested path failed: %v", err)
	}
}