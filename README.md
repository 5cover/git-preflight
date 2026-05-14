# git-preflight

`git-preflight` checks whether Git repositories contain local state that would be lost or stranded if the checkout disappeared.

It is intended for moments when you are about to leave for a long time, switch machines, wipe a directory, travel, shut down a workstation, or otherwise stop relying on local disk state. Instead of manually checking each repository with `git status`, `git-preflight` recursively discovers repositories and reports only the ones that need attention.

The core question is:

> If this repository were deleted right now, would I lose anything or leave work unavailable elsewhere?

`git-preflight` is not a replacement for `git status`. It is a workspace-level safety check.

## Build

Build the CLI from the repository root:

```sh
go build -o git-preflight ./cmd/git-preflight
```

Release builds can inject a version string:

```sh
go build -ldflags "-X main.version=v0.1.0" -o git-preflight ./cmd/git-preflight
```

## Test

Run the test suite:

```sh
go test ./...
```

Tests create temporary real Git repositories and require `git` on `$PATH`.

If the default Go build cache is not writable, set `GOCACHE`:

```sh
GOCACHE=/tmp/git-preflight-go-cache go test ./...
```

## Install

Install the latest CLI with Go:

```sh
go install github.com/5cover/git-preflight/cmd/git-preflight@latest
```

Go installs the executable into `$GOBIN`, or `$GOPATH/bin` when `$GOBIN` is unset. Ensure that directory is on `$PATH`.

This installs the binary only. To install the manpage manually on Unix-like systems:

```sh
install -Dm644 docs/man/git-preflight.1 /usr/local/share/man/man1/git-preflight.1
mandb 2>/dev/null || true
```

Release archives include the binary, `README.md`, and `man/man1/git-preflight.1`.

## Installation model

The executable is named `git-preflight`

When `git-preflight` is available on `$PATH`, Git automatically exposes it as `git preflight`
No Git alias is required.

## Distribution

Create release archives from the repository root:

```sh
VERSION=v0.1.0 ./scripts/build-release.sh
```

The script writes archives and `checksums.txt` to `dist/` for:

- `linux-amd64`
- `linux-arm64`
- `darwin-amd64`
- `darwin-arm64`
- `windows-amd64`

GitHub Releases are the primary upstream binary distribution channel.

Other useful distribution channels:

- Homebrew tap for macOS and Linux, including binary and manpage installation.
- Winget, Scoop, or Chocolatey for Windows binary installation.
- Debian `.deb` and Fedora/RHEL `.rpm` packages for system package managers, including manpage installation.
- Arch Linux PKGBUILD/AUR for Arch users.
- Nix package or flake for reproducible installs.

The Go module is published by tagging releases. Users can import `github.com/5cover/git-preflight` as a library or install the CLI with `go install`.

## Basic usage

Check the current repository:

```sh
git preflight
```

Recursively check repositories under the current directory:

```sh
git preflight -r
```

Recursively check repositories under a specific directory:

```sh
git preflight -r ~/repos
```

Show clean repositories as well:

```sh
git preflight -r --verbose
```

Produce machine-readable output:

```sh
git preflight -r --json
```

## Design principle

By default, clean repositories produce no output.

This means:

```sh
git preflight -r ~/repos
```

printing nothing means no local state was found.

Any output means attention is required.

## Definitions

### Repository

A repository is any directory recognized as a Git work tree.

A repository may be discovered by finding `.git/` for a Git indirection file `.git` containing a `gitdir:` entry, as used by submodules and worktrees.

Implementations may use Git itself to validate repository roots, for example:

```sh
git -C <path> rev-parse --show-toplevel
```

### Local state

A repository has local state if any of the following are true:

- the work tree has unstaged tracked changes
- the index has staged but uncommitted changes
- the repository has untracked, non-ignored files
- the current branch has commits ahead of its upstream
- the repository has commits on a local branch with no configured upstream
- the repository is in the middle of an operation such as merge, rebase, cherry-pick, revert, or bisect
- the repository has one or more stashes

The tool needs not to print file or commit lists. It only needs to report that local state exists and classify it briefly.

### Clean repository

A repository is clean if deleting the checkout would not lose local changes or strand unpublished commits, assuming the configured remotes remain available.

A clean repository may still have ignored files. Ignored files are not considered local state.

### Findings

A finding is a short classification describing why a repository has local state.

Findings are intentionally coarse-grained. They exist to direct attention, not to replace `git status`.

Each repository produces the smallest meaningful set of findings.

| Finding     | Meaning                                                                | Report when                                                                                                                               |
| ----------- | ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `dirty`     | Repository contains local modifications not covered by another finding | index/worktree divergence not otherwise classified, or any local mutable state detected that does not map cleanly to another finding      |
| `error`     | Repository could not be inspected reliably                             | Git command failed, repository is corrupted, permissions denied, repository metadata unreadable, or repository state cannot be determined |
| `staged`    | Index contains changes not yet committed                               | Staged entries exist in the index                                                                                                         |
| `detached`  | Commits may become unreachable because HEAD is detached                | `HEAD` is detached and repository contains commits not reachable from a named branch or upstream                                          |
| `noremote`  | Local branch has commits but no upstream publication target            | Current branch has local commits and no configured upstream branch                                                                        |
| `unpushed`  | Local commits exist but are not published to upstream                  | Current branch is ahead of its upstream                                                                                                   |
| `unstaged`  | Worktree contains unstaged changes                                     | Untracked or modified tracked files are present                                                                                           |
| `stash`     | Work has been temporarily hidden in stash entries                      | One or more stash entries exist                                                                                                           |
| `operation` | Repository is in the middle of a Git operation                         | Merge, rebase, cherry-pick, revert, bisect, or similar operation is in progress                                                           |

Findings always appear in this order in output.

Findings are mutually exclusive in signification. Examples:

- unstaged tracked modification → `unstaged`
- staged file only → `staged`
- ahead of upstream → `unpushed`
- local branch with commits and no upstream → `noremote`

A single repository may still report multiple findings if they describe independent conditions, but a single underlying condition does not normally emit multiple synonymous findings.

Example: `unpushed unstaged` is valid because unpublished commits and unstaged changes are distinct categories of local state.

However: `staged dirty` does not occur if the staged condition already fully explains the detected state.

`dirty` is intended as a fallback category for residual mutable repository state not already classified.

## CLI interface

```text
git preflight [options] [path]
```

### Arguments

```text
[path]
```

Directory to inspect.

Default: `.`

If `-r` is not set, `path` is inside a Git repository or be a repository root.

If `-r` is set, `path` is treated as the root of a recursive scan.

## Options

### `-r`, `--recursive`

Recursively scan for Git repositories below `path`.

Behavior:

- walk the filesystem starting at `path`
- discover every Git repository
- inspect each discovered repository independently
- do not stop at the first repository boundary
- continue scanning inside repositories so nested repositories and submodules are also found
- skip Git internal storage directories such as `.git/`
- do not report clean repositories unless `--verbose` is set

Rationale:

A parent repository can be clean while a submodule or nested repository contains unpublished local state. Therefore recursive mode treats every repository as its own failure domain.

### `-v`, `--verbose`

Print clean repositories as well as repositories with local state.

Default behavior is silent for clean repositories.

Example:

```text
          ~/repos/api
unpushed  ~/repos/site
          ~/repos/tooling
```

### `--json`

Print results as JSON.

JSON output includes clean repositories only when `--verbose` is set.

Recommended schema:

```json
{
  "root": "/home/user/repos",
  "recursive": true,
  "repositories": [
    {
      "path": "/home/user/repos/api",
      "findings": ["unpushed"]
    }
  ]
}
```

A clean repo is indicated by an empty findings array.

### `-q`, `--quiet`

Suppress normal output.

Exit codes still apply.

Useful for scripts.

### `--fail-fast`

Stop scanning after the first repository with local state.

Exit immediately with status `1`.

### `--no-stash`

Do not consider stashes local state.

Default is to consider stashes local state.

### `--no-operations`

Do not consider in-progress Git operations local state.

Default is to consider in-progress operations local state.

### `--version`

Print the program version and exit.

### `-h`, `--help`

Print usage information and exit.

## Output behavior

### Default human output

Default output is line-oriented and skimmable.

Recommended format:

```text
<finding>[ <finding>...]  <path>
```

Example:

```text
unpushed unstaged  ~/repos/site
unpushed           ~/repos/api
noremote           ~/repos/plugin
stash              ~/repos/experiment
```

Filenames stay aligned with the longest status line.

The tool does not print file names by default.

The tool does not print clean repositories by default.

### Verbose output

When `--verbose` is set:

```text
                ~/repos/api
dirty unpushed  ~/repos/site
noremote        ~/repos/plugin
```

~/repos/api is clean, so no findings are shown

### Error output

Operational errors is printed to stderr.

Examples:

```text
git-preflight: cannot read /home/user/repos/private: permission denied
git-preflight: /home/user/tmp is not a git repository
git-preflight: git not found on PATH
```

In recursive mode, a repository-level error does not abort the whole scan unless `--fail-fast` is set. The repository is reported with an `error` finding.

## Exit codes

```text
0    no local state found
1    local state found
2    operational error
```

If both local state and operational errors occur, exit code is `2`.

Examples:

```sh
git preflight -r ~/repos
```

returns 0

when every discovered repository is clean.

It returns 1

when at least one repository contains local state.

It returns 2

when the scan could not be completed reliably due to an operational error.

## Repository inspection behavior

An implementation uses Git as the source of truth wherever possible.

Recommended command:

```sh
git -C <repo> status --porcelain=v2 --branch --untracked-files=all
```

This can detect:

- staged changes
- unstaged tracked changes
- untracked non-ignored files
- current branch
- upstream branch
- ahead/behind counts
- detached HEAD

Relevant porcelain v2 lines:

```text
# branch.head <name>
# branch.upstream <name>
# branch.ab +<ahead> -<behind>
```

Any non-header status line indicates local work tree or index state.

A positive ahead count indicates unpushed commits.

If a branch has commits but no upstream, the repository is reported as `noremote`.

## Stash detection

Default behavior treats stashes as local state.

Recommended command:

```sh
git -C <repo> stash list
```

If output is non-empty, report finding `stash`

## In-progress operation detection

Default behavior treats in-progress Git operations as local state.

The implementation may check for repository state using Git commands or by inspecting Git administrative files.

Operations to detect include:

- merge
- rebase
- cherry-pick
- revert
- bisect

Finding: `operation`

## Recursive discovery behavior

Recursive discovery finds nested repositories.

Given:

- ~/repos/parent
- ~/repos/parent/submodule
- ~/repos/parent/vendor/tool

all three is checked independently if each is a Git repository.

The scanner does not assume that a clean parent implies clean submodules.

The scanner skips Git internal storage directories, including: `.git/`

For `.git` files, the scanner does not recurse into the referenced Git administrative directory.

## Ignored files

Ignored files do not count as local state.

Inside a repository, ignored file handling is delegated to Git.

The tool does not implement its own `.gitignore` parser.

Outside repositories, recursive mode performs normal filesystem traversal. It does not hardcode ecosystem-specific exclusions.

## Progress output

Since recursive scanning takes noticeable time, when -r is provided and stderr is a terminal, the tool prints progress to stderr by keeping a progress line updated, rewriting it with \r.

Examples:

```text
scanning: ~/repos/archive
checked 136 repositories\rchecked 137 repositories
```

Progress output is disabled when `--quiet` is set.

For \r flushing, use a logic analogue to this C program:

```c
#include <stdio.h>
#include <unistd.h>

int main() {
  int l = 0;
  for (int i = 0; i < 100; ++i) {
    if (i % 8) {
      // progress line
      l = printf(".%d", i);
      fflush(stdout);
      putchar('\r');
    } else {
      // diagnostics line
      // has to be at least as long as the progress line to overwrite it
      l -= printf("#%d", i);
      // the putchar loop could replaced by a padded string printf
      while (l-- > 0)
        putchar(' ');
      putchar('\n');
    }
    usleep(100 * 1000);
  }
}
```

## Path handling

Human output may use paths relative to the scan root when practical.

Example:

```text
unpushed  api
dirty     site
```

JSON output use absolute paths.

The tool handles paths with spaces.

## Submodules

Submodules is inspected as independent repositories in recursive mode.

A parent repository may not reveal unpublished commits inside a submodule. Therefore `git preflight -r` discovers and inspects submodule working trees directly.

The tool also detects parent-level submodule pointer changes as dirty state in the parent repository.

## Worktrees

Git worktrees is supported.

A `.git` file with a `gitdir:` pointer is treated as a valid repository marker.

The worktree is inspected using normal Git commands with `git -C <worktree>`.

## Non-recursive behavior

Without `-r`, the tool inspects exactly one repository: the repository containing `path`.

Examples:

```sh
git preflight
git preflight .
git preflight ~/repos/api
```

If `path` is not inside a Git repository, the tool exits with code `2`.

## Smoke test

The CLI implements

```sh
git preflight
git preflight -r
git preflight -r <path>
git preflight --verbose
git preflight --json
```

It detects

- dirty
- unstaged
- unpushed
- noremote

It

- prints nothing for clean repositories by default
- returns 0 when all checked repositories are clean
- returns 1 when any checked repository has local state
- returns 2 on operational error
- discovers nested repositories in recursive mode
- supports .git directories and .git files
- avoids reporting ignored files as local state

## Non-goals

`git-preflight` is not intended to:

- replace `git status`
- show file-level diffs
- auto-commit, auto-push, or mutate repositories
- lint repositories
- validate branch naming
- enforce workflow policy
- inspect remote availability beyond normal Git metadata
- determine whether pushed commits have been reviewed or merged

## Example session

```sh
$ git preflight -r ~/repos

unpushed          ~/repos/api
unpushed unstaged ~/repos/site
noremote          ~/repos/plugin
```

The user can then manually inspect each repository:

```sh
cd ~/repos/api
git status
git push
```

When everything is safe:

```sh
$ git preflight -r ~/repos
$
```

## Testing strategy

`git-preflight` is tested against real Git repositories created dynamically during test execution.

The tool exists specifically to reason about real repository state. Mocking Git or storing static fixture repositories is insufficient for many important cases, especially transient repository states such as rebases, merges, stashes, unpublished commits, detached HEADs, and submodules.

Tests construct repositories procedurally using actual Git commands.

### Guiding principle

Tests describe repository situations, not filesystem snapshots.

A good test reads like:

- create repository
- create unpublished commit
- run git-preflight
- assert repository is reported as unpushed

instead of:

- unpack mysterious fixture directory
- hope its internal Git metadata still behaves correctly

### Recommended test structure

Each test

1. creates temporary directories
2. initializes repositories using real Git commands
3. constructs the desired repository state
4. executes `git-preflight`
5. asserts output and exit codes

Tests avoid sharing repositories between test cases.

### Prefer end-to-end subprocess tests

`git-preflight` should primarily be tested as an executable rather than as direct Go function calls.

The real interface of the program is:

```text
filesystem + git executable + stdout/stderr + exit code
```

not internal package APIs.

Most behavior depends on interactions between:

- repository discovery
- filesystem traversal
- Git command execution
- porcelain parsing
- CLI argument handling
- output formatting
- exit status behavior

Therefore the most valuable tests are end-to-end integration tests that:

1. create temporary repositories
2. construct real Git states
3. execute the `git-preflight` binary as a subprocess
4. assert stdout, stderr, and exit codes

Example:

```go
cmd := exec.Command(preflightPath, "-R", root)
out, err := cmd.CombinedOutput()
```

This approach verifies the complete observable behavior of the tool.

Internal unit tests are still useful for isolated logic such as:

- porcelain parsing
- JSON formatting
- path rendering
- progress rendering
- argument parsing

However, repository state behavior should primarily be validated through real subprocess execution against real Git repositories.

### Temporary directories

e2e tests create isolated temporary directories.

In Go:

```go
dir := t.TempDir()
```

All repositories used by the test exist under this directory.

### Using real Git commands

Tests invoke the real Git executable rather than mocking repository state.

Example helper:

```go
func git(t *testing.T, dir string, args ...string) string
```

This helper

- runs `git`
- fails the test on nonzero exit
- returns stdout
- includes stderr in failure messages

Example usage:

```go
git(t, repo, "init")
git(t, repo, "add", ".")
git(t, repo, "commit", "-m", "initial")
```

### Recommended baseline configuration

Tests configure repositories explicitly so they do not depend on the developer machine configuration.

Example:

```sh
git config user.name test
git config user.email test@example.com
```

This is done for every created repository.

### Testing unpushed commits

Unpushed commit detection uses a real remote.

Example tree:

```text
tmp/
  remote.git/
  work/
```

Procedure:

```sh
git init --bare remote.git
git clone remote.git work
```

Then:

```sh
git commit
```

without pushing.

Expected result: `unpushed`

### Testing dirty repositories

Modify a tracked file without committing:

```sh
echo hello >> file.txt
```

Expected result: dirty

### Testing untracked files

Create a new file without adding it:

```sh
touch scratch.txt
```

Expected result: unstaged

### Testing ignored files

Create a `.gitignore`:

```gitignore
build/
```

Then create ignored files:

```sh
mkdir build
touch build/output.bin
```

Expected result:

repository remains clean.

Ignored files do not count as local state.

### Testing stashes

Procedure:

```sh
echo test >> file.txt
git stash
```

Expected result: `stash`

unless stash detection is disabled.

### Testing repositories without upstreams

Create a local branch with commits but no upstream tracking branch.

Expected result: `noremote`

### Testing merge or rebase state

Tests intentionally create incomplete operations.

Example merge conflict:

```sh
git merge conflicting-branch
```

without resolving conflicts.

Expected result:

```text
local state: operation
```

### Testing recursive discovery

Construct nested repositories:

```text
root/
  a/
  b/
  parent/
    nested/
```

All repositories is discovered independently.

Tests verify that nested repositories are still inspected even when their parent repository is clean.

### Testing submodules

Submodules is tested using actual Git submodule commands.

Example:

```sh
git submodule add ../child child
```

Tests verify:

- parent repository detection
- submodule detection
- unpublished commits inside submodule
- recursive submodule discovery

Important case:

A parent repository may appear clean while a submodule contains unpublished commits. Recursive scanning still reports the submodule.

### Testing worktrees

Tests create real Git worktrees:

```sh
git worktree add ../worktree feature
```

Expected behavior:

- worktree is discovered as a repository
- `.git` indirection files are handled correctly
- local state inside worktrees is detected

### Testing progress output

Progress output is tested separately from repository findings.

Tests verify:

- progress output appears only on terminals
- progress output is written to stderr
- stdout remains parseable
- `--json` suppresses progress output
- `--quiet` suppresses progress output

Tests does not depend on ANSI escape support.

### Testing output behavior

Tests verify:

#### default mode

- clean repositories produce no output
- repositories with local state produce output

#### verbose mode

- clean repositories are printed

#### JSON mode

- output is valid JSON
- stdout contains no progress text
- repository findings are correctly encoded

### Testing exit codes

Tests verify:

```text
0    all repositories clean
1    local state detected
2    operational error
```

Important cases:

- recursive scan with one dirty repository
- recursive scan with inaccessible directory
- invalid repository path
- missing Git executable

### Fixture repositories

Static fixture repositories may be used sparingly for simple topology tests.

If fixture repositories are stored in the source tree, their `.git` directories is escaped to avoid interference from the repository containing `git-preflight`.

Example:

```text
fixtures/basic/.git_escaped
```

During test setup:

```text
.git_escaped -> .git
```

However, static fixtures does not be relied upon for transient Git states such as:

- rebases
- merges
- stashes
- worktrees
- unpublished commits
- submodule state

These states is created dynamically during test execution.

### Avoid mocking Git

Tests prefer real Git behavior over mocks.

`git-preflight` is fundamentally an orchestration layer around Git state inspection. Correctness depends on matching actual Git semantics.

Mocking is generally limited to operational failure scenarios such as:

- Git executable missing
- command execution failure
- permission errors
- timeouts
- corrupted repositories

Repository state behavior is tested using real repositories.
