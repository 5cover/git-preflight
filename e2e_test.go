package preflight_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type cliResult struct {
	code   int
	stdout string
	stderr string
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "git-preflight")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/git-preflight")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build cli: %v\n%s", err, out)
	}
	return bin
}

func runCLI(t *testing.T, bin string, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("run cli: %v", err)
		}
	}
	return cliResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "user.email", "test@example.com")
}

func commitFile(t *testing.T, dir, name, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", name)
	git(t, dir, "commit", "-m", message)
}

func remoteClone(t *testing.T) (root string, remote string, work string) {
	t.Helper()
	root = t.TempDir()
	remote = filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	work = filepath.Join(root, "work")

	git(t, root, "init", "--bare", remote)
	initRepo(t, seed)
	commitFile(t, seed, "file.txt", "initial\n", "initial")
	git(t, seed, "branch", "-M", "main")
	git(t, seed, "remote", "add", "origin", remote)
	git(t, seed, "push", "-u", "origin", "main")
	git(t, remote, "symbolic-ref", "HEAD", "refs/heads/main")
	git(t, root, "clone", remote, work)
	git(t, work, "config", "user.name", "test")
	git(t, work, "config", "user.email", "test@example.com")
	return root, remote, work
}

func TestCleanRepositoryIsSilent(t *testing.T) {
	bin := buildCLI(t)
	_, _, work := remoteClone(t)

	got := runCLI(t, bin, work)
	if got.code != 0 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	if got.stdout != "" {
		t.Fatalf("stdout = %q, want empty", got.stdout)
	}
}

func TestUntrackedFileReportsUnstaged(t *testing.T) {
	bin := buildCLI(t)
	repo := filepath.Join(t.TempDir(), "repo")
	initRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "scratch.txt"), []byte("scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, bin, repo)
	if got.code != 1 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "unstaged") {
		t.Fatalf("stdout = %q, want unstaged", got.stdout)
	}
}

func TestIgnoredFilesDoNotCount(t *testing.T) {
	bin := buildCLI(t)
	repo := filepath.Join(t.TempDir(), "repo")
	initRepo(t, repo)
	exclude := filepath.Join(repo, ".git", "info", "exclude")
	if err := os.WriteFile(exclude, []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "build", "output.bin"), []byte("ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, bin, repo)
	if got.code != 0 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	if got.stdout != "" {
		t.Fatalf("stdout = %q, want empty", got.stdout)
	}
}

func TestUnpushedWithRealRemote(t *testing.T) {
	bin := buildCLI(t)
	_, _, work := remoteClone(t)
	commitFile(t, work, "file.txt", "changed\n", "local")

	got := runCLI(t, bin, work)
	if got.code != 1 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "unpushed") {
		t.Fatalf("stdout = %q, want unpushed", got.stdout)
	}
}

func TestRecursiveJSONDiscoversNestedRepositories(t *testing.T) {
	bin := buildCLI(t)
	root := t.TempDir()
	clean := filepath.Join(root, "clean")
	dirty := filepath.Join(root, "parent", "nested")
	initRepo(t, clean)
	initRepo(t, dirty)
	if err := os.WriteFile(filepath.Join(dirty, "scratch.txt"), []byte("scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, bin, "-r", "--json", root)
	if got.code != 1 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	var decoded struct {
		Repositories []struct {
			Path     string   `json:"path"`
			Findings []string `json:"findings"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &decoded); err != nil {
		t.Fatalf("json: %v\n%s", err, got.stdout)
	}
	if len(decoded.Repositories) != 1 {
		t.Fatalf("repositories = %#v", decoded.Repositories)
	}
	if decoded.Repositories[0].Path != dirty {
		t.Fatalf("path = %q, want %q", decoded.Repositories[0].Path, dirty)
	}
	if strings.Join(decoded.Repositories[0].Findings, " ") != "unstaged" {
		t.Fatalf("findings = %#v", decoded.Repositories[0].Findings)
	}
}

func TestInvalidNonRecursivePathExitsOperationalError(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	got := runCLI(t, bin, dir)
	if got.code != 2 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
}

func TestRecursivePermissionErrorContinuesWithoutFailFast(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read chmod 000 directories")
	}
	bin := buildCLI(t)
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	blocked := filepath.Join(root, "blocked")
	initRepo(t, repo)
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blocked, 0o755)
	})

	got := runCLI(t, bin, "-r", "-v", root)
	if got.code != 2 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "permission denied") {
		t.Fatalf("stderr = %q, want permission denied", got.stderr)
	}
	if !strings.Contains(got.stdout, "repo") {
		t.Fatalf("stdout = %q, want successfully scanned repo", got.stdout)
	}
}
