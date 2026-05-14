package preflight

import (
	"context"
	"path/filepath"
	"strings"
)

func inspectRepository(ctx context.Context, opts Options, repo string) RepositoryResult {
	result := RepositoryResult{Path: repo}
	seen := map[Finding]bool{}

	statusOut, err := runGit(ctx, opts.GitPath, repo, "status", "--porcelain=v2", "--branch", "--untracked-files=all")
	if err != nil {
		result.Findings = []Finding{FindingError}
		result.Error = err.Error()
		return result
	}
	status := parseStatusPorcelainV2(statusOut)

	if status.Dirty {
		seen[FindingDirty] = true
	}
	if status.Staged {
		seen[FindingStaged] = true
	}
	if status.Unstaged {
		seen[FindingUnstaged] = true
	}
	if status.Ahead > 0 && status.Upstream != "" {
		seen[FindingUnpushed] = true
	}
	if status.Head != "" && status.Head != "(detached)" && status.Upstream == "" && branchHasCommits(ctx, opts, repo, status.Head) {
		seen[FindingNoRemote] = true
	}
	if status.Head == "(detached)" && detachedHasUnreachableCommits(ctx, opts, repo) {
		seen[FindingDetached] = true
	}
	operation := opts.IncludeOperation && hasOperation(ctx, opts, repo)
	if operation {
		delete(seen, FindingDirty)
		delete(seen, FindingStaged)
		delete(seen, FindingUnstaged)
		seen[FindingOperation] = true
	}
	if opts.IncludeStash && hasStash(ctx, opts, repo) {
		seen[FindingStash] = true
	}

	result.Findings = orderedFindings(seen)
	return result
}

func branchHasCommits(ctx context.Context, opts Options, repo string, branch string) bool {
	out, err := runGit(ctx, opts.GitPath, repo, "rev-list", "--count", branch)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != "0"
}

func detachedHasUnreachableCommits(ctx context.Context, opts Options, repo string) bool {
	out, err := runGit(ctx, opts.GitPath, repo, "branch", "--contains", "HEAD", "--format=%(refname)")
	if err == nil && strings.TrimSpace(out) != "" {
		return false
	}
	out, err = runGit(ctx, opts.GitPath, repo, "branch", "-r", "--contains", "HEAD", "--format=%(refname)")
	if err == nil && strings.TrimSpace(out) != "" {
		return false
	}
	return true
}

func hasStash(ctx context.Context, opts Options, repo string) bool {
	out, err := runGit(ctx, opts.GitPath, repo, "stash", "list")
	return err == nil && strings.TrimSpace(out) != ""
}

func hasOperation(ctx context.Context, opts Options, repo string) bool {
	gd, err := gitDir(ctx, opts.GitPath, repo)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(repo, gd)
	}
	markers := []string{
		"MERGE_HEAD",
		"CHERRY_PICK_HEAD",
		"REVERT_HEAD",
		"BISECT_LOG",
		"rebase-merge",
		"rebase-apply",
	}
	for _, marker := range markers {
		if exists(filepath.Join(gd, marker)) {
			return true
		}
	}
	return false
}
