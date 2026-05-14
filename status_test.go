package preflight

import "testing"

func TestParseStatusPorcelainV2(t *testing.T) {
	out := "" +
		"# branch.head main\n" +
		"# branch.upstream origin/main\n" +
		"# branch.ab +2 -0\n" +
		"1 M. N... 100644 100644 100644 a b file.txt\n" +
		"1 .M N... 100644 100644 100644 a b other.txt\n" +
		"? scratch.txt\n"

	got := parseStatusPorcelainV2(out)
	if got.Head != "main" {
		t.Fatalf("head = %q", got.Head)
	}
	if got.Upstream != "origin/main" {
		t.Fatalf("upstream = %q", got.Upstream)
	}
	if got.Ahead != 2 {
		t.Fatalf("ahead = %d", got.Ahead)
	}
	if !got.Staged {
		t.Fatal("expected staged")
	}
	if !got.Unstaged {
		t.Fatal("expected unstaged")
	}
}

func TestOrderedFindings(t *testing.T) {
	got := orderedFindings(map[Finding]bool{
		FindingOperation: true,
		FindingStaged:    true,
		FindingUnpushed:  true,
	})
	want := []Finding{FindingStaged, FindingUnpushed, FindingOperation}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
