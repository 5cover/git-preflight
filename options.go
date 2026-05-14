package preflight

import "io"

// Options controls repository scanning and inspection.
type Options struct {
	Path             string
	Recursive        bool
	Verbose          bool
	Quiet            bool
	FailFast         bool
	IncludeStash     bool
	IncludeOperation bool
	GitPath          string
	ProgressWriter   io.Writer
}

// DefaultOptions returns the default scan options used by the CLI.
func DefaultOptions() Options {
	return Options{
		Path:             ".",
		IncludeStash:     true,
		IncludeOperation: true,
		GitPath:          "git",
	}
}

// Result is the complete output of a preflight scan.
type Result struct {
	Root         string             `json:"root"`
	Recursive    bool               `json:"recursive"`
	Repositories []RepositoryResult `json:"repositories"`
}

// RepositoryResult is the inspection result for one repository.
type RepositoryResult struct {
	Path     string    `json:"path"`
	Findings []Finding `json:"findings"`
	Error    string    `json:"-"`
}

// HasLocalState reports whether any repository has local state findings.
func (r Result) HasLocalState() bool {
	for _, repo := range r.Repositories {
		if len(repo.Findings) > 0 && !hasFinding(repo.Findings, FindingError) {
			return true
		}
		for _, finding := range repo.Findings {
			if finding != FindingError {
				return true
			}
		}
	}
	return false
}

// HasError reports whether any repository has an inspection error.
func (r Result) HasError() bool {
	for _, repo := range r.Repositories {
		if hasFinding(repo.Findings, FindingError) {
			return true
		}
	}
	return false
}
