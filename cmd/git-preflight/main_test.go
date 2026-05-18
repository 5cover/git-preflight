package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRejectsTooManyPaths(t *testing.T) {
	if code := run([]string{"a", "b"}); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunAcceptsBundledShortOptions(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"-rv", root}); code != 2 {
		t.Fatalf("exit code = %d, want 2 because fake repo inspection fails after parsing succeeds", code)
	}
}

func TestRunAcceptsBundledRecursiveQuiet(t *testing.T) {
	root := t.TempDir()
	if code := run([]string{"-rq", root}); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestRunRejectsUnknownShortOption(t *testing.T) {
	if code := run([]string{"-z"}); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}
