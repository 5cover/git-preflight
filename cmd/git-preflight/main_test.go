package main

import "testing"

func TestRunRejectsTooManyPaths(t *testing.T) {
	if code := run([]string{"a", "b"}); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}
