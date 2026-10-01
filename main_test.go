package tffumpt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRecursiveWriteDoesNotModifyFilesWhenFormattingFails(t *testing.T) {
	dir := t.TempDir()
	firstFile := filepath.Join(dir, "a.tf")
	invalidFile := filepath.Join(dir, "b.tf")
	firstSource := []byte("locals {\n  value=1\n}\n")

	if err := os.WriteFile(firstFile, firstSource, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidFile, []byte("locals {\n  value = [for n in [1] n]\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	status := Fumpt([]string{dir}, &Options{Write: true, Recursive: true})
	if status != 2 {
		t.Fatalf("Fumpt() status = %d, want 2", status)
	}

	got, err := os.ReadFile(firstFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, firstSource) {
		t.Fatalf("first file was modified before the formatting error:\n%s", got)
	}
}
