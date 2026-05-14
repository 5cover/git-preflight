package preflight

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteHumanAlignsRelativePaths(t *testing.T) {
	result := Result{
		Root:      "/tmp/repos",
		Recursive: true,
		Repositories: []RepositoryResult{
			{Path: "/tmp/repos/api", Findings: nil},
			{Path: "/tmp/repos/site", Findings: []Finding{FindingUnpushed, FindingUnstaged}},
		},
	}
	var buf bytes.Buffer
	if err := WriteHuman(&buf, result); err != nil {
		t.Fatal(err)
	}
	want := "                   api\nunpushed unstaged  site\n"
	if buf.String() != want {
		t.Fatalf("output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestWriteJSON(t *testing.T) {
	result := Result{
		Root:      "/tmp/repos",
		Recursive: true,
		Repositories: []RepositoryResult{
			{Path: "/tmp/repos/site", Findings: []Finding{FindingUnpushed}},
		},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Repositories[0].Findings[0] != FindingUnpushed {
		t.Fatalf("finding = %q", decoded.Repositories[0].Findings[0])
	}
}
