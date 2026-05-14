package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	preflight "github.com/5cover/git-preflight"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts := preflight.DefaultOptions()
	var jsonOut bool
	var showVersion bool

	fs := flag.NewFlagSet("git-preflight", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&opts.Recursive, "r", false, "recursively scan for Git repositories")
	fs.BoolVar(&opts.Recursive, "recursive", false, "recursively scan for Git repositories")
	fs.BoolVar(&opts.Verbose, "v", false, "print clean repositories as well")
	fs.BoolVar(&opts.Verbose, "verbose", false, "print clean repositories as well")
	fs.BoolVar(&jsonOut, "json", false, "print JSON output")
	fs.BoolVar(&opts.Quiet, "q", false, "suppress normal output")
	fs.BoolVar(&opts.Quiet, "quiet", false, "suppress normal output")
	fs.BoolVar(&opts.FailFast, "fail-fast", false, "stop after first repository with local state")
	noStash := fs.Bool("no-stash", false, "do not consider stashes local state")
	noOperations := fs.Bool("no-operations", false, "do not consider in-progress operations local state")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: git preflight [options] [path]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if showVersion {
		fmt.Fprintf(os.Stdout, "git-preflight %s\n", version)
		return 0
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "git-preflight: expected at most one path")
		return 2
	}
	if fs.NArg() == 1 {
		opts.Path = fs.Arg(0)
	}
	opts.IncludeStash = !*noStash
	opts.IncludeOperation = !*noOperations
	if opts.Recursive && !opts.Quiet && !jsonOut && isTerminal(os.Stderr) {
		opts.ProgressWriter = os.Stderr
	}

	result, err := preflight.Run(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-preflight: %v\n", err)
		return 2
	}
	for _, repo := range result.Repositories {
		if repo.Error != "" {
			fmt.Fprintf(os.Stderr, "git-preflight: %s: %s\n", repo.Path, repo.Error)
		}
	}
	if !opts.Quiet {
		if jsonOut {
			if err := preflight.WriteJSON(os.Stdout, result); err != nil {
				fmt.Fprintf(os.Stderr, "git-preflight: %v\n", err)
				return 2
			}
		} else if err := preflight.WriteHuman(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "git-preflight: %v\n", err)
			return 2
		}
	}
	if result.HasError() {
		return 2
	}
	if result.HasLocalState() {
		return 1
	}
	return 0
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
