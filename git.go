package preflight

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func runGit(ctx context.Context, gitPath string, dir string, args ...string) (string, error) {
	if gitPath == "" {
		gitPath = "git"
	}
	fullArgs := make([]string, 0, len(args)+2)
	if dir != "" {
		fullArgs = append(fullArgs, "-C", dir)
	}
	fullArgs = append(fullArgs, args...)
	cmd := exec.CommandContext(ctx, gitPath, fullArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), errors.New(msg)
	}
	return stdout.String(), nil
}

func repositoryRoot(ctx context.Context, gitPath string, path string) (string, error) {
	out, err := runGit(ctx, gitPath, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repository", path)
	}
	return strings.TrimSpace(out), nil
}

func gitDir(ctx context.Context, gitPath string, repo string) (string, error) {
	out, err := runGit(ctx, gitPath, repo, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
