package preflight

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

	repositories, err := runScan(ctx, opts, root)

	return Result{
		Root:         root,
		Recursive:    opts.Recursive,
		Repositories: repositories,
	}, err
}

func runScan(ctx context.Context, opts Options, root string) ([]RepositoryResult, error) {
	if opts.Recursive {
		scanned, err := recurseRepositories(ctx, opts, root)
		return scanned, err
	}
	repo, err := repositoryRoot(ctx, opts.GitPath, root)
	if err != nil {
		return nil, err
	}
	scanned := inspectRepository(ctx, opts, repo)
	if opts.Verbose || len(scanned.Findings) > 0 {
		return []RepositoryResult{scanned}, err
	} else {
		return []RepositoryResult{}, err
	}
}

func recurseRepositories(ctx context.Context, opts Options, root string) ([]RepositoryResult, error) {
	var result []RepositoryResult
	n := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result = append(result, RepositoryResult{
				Path:     path,
				Findings: []Finding{FindingError},
				Error:    walkErr.Error(),
			})
			if opts.FailFast {
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
			if opts.ProgressWriter != nil {
				fmt.Fprintf(opts.ProgressWriter, "checked %d repositories\r", n)
			}
			repo := inspectRepository(ctx, opts, parent)
			n++
			if opts.Verbose || len(repo.Findings) > 0 {
				result = append(result, repo)
			}
			if opts.FailFast && len(repo.Findings) > 0 {
				return fs.SkipAll
			}

		}
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		if opts.FailFast {
			return result, nil
		}
		return result, err
	}
	return result, nil
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
