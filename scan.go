package preflight

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Run scans repositories according to opts.
func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Path == "" {
		opts.Path = "."
	}
	if opts.GitPath == "" {
		opts.GitPath = "git"
	}

	root, err := filepath.Abs(opts.Path)
	if err != nil {
		return Result{}, err
	}

	var repos []string
	var scanErrors []RepositoryResult
	if opts.Recursive {
		repos, scanErrors, err = discoverRepositories(root, opts.FailFast)
		if err != nil {
			return Result{}, err
		}
	} else {
		repo, err := repositoryRoot(ctx, opts.GitPath, root)
		if err != nil {
			return Result{}, err
		}
		repos = []string{repo}
		root = repo
	}
	sort.Strings(repos)
	sort.Slice(scanErrors, func(i, j int) bool {
		return scanErrors[i].Path < scanErrors[j].Path
	})

	result := Result{Root: root, Recursive: opts.Recursive, Repositories: make([]RepositoryResult, 0)}
	for _, scanErr := range scanErrors {
		if opts.Verbose || len(scanErr.Findings) > 0 {
			result.Repositories = append(result.Repositories, scanErr)
		}
		if opts.FailFast {
			return result, nil
		}
	}
	checked := 0
	for _, repo := range repos {
		inspected := inspectRepository(ctx, opts, repo)
		checked++
		if opts.ProgressWriter != nil {
			fmt.Fprintf(opts.ProgressWriter, "checked %d repositories\r", checked)
		}
		if opts.Verbose || len(inspected.Findings) > 0 {
			result.Repositories = append(result.Repositories, inspected)
		}
		if opts.FailFast && len(inspected.Findings) > 0 {
			break
		}
	}
	return result, nil
}

func discoverRepositories(root string, failFast bool) ([]string, []RepositoryResult, error) {
	var repos []string
	var scanErrors []RepositoryResult
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			scanErrors = append(scanErrors, RepositoryResult{
				Path:     path,
				Findings: []Finding{FindingError},
				Error:    walkErr.Error(),
			})
			if failFast {
				return walkErr
			}
			return filepath.SkipDir
		}
		name := d.Name()
		if name != ".git" {
			return nil
		}
		parent := filepath.Dir(path)
		if isGitMarker(path, d) {
			repos = append(repos, parent)
		}
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		if failFast {
			return repos, scanErrors, nil
		}
		return repos, scanErrors, err
	}
	return repos, scanErrors, nil
}

func isGitMarker(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		return true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return len(data) >= len("gitdir:") && string(data[:len("gitdir:")]) == "gitdir:"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
