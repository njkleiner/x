package main

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// zero is the base reference for the initial push to a new branch.
	zero = "0000000000000000000000000000000000000000"
)

func main() {
	var cfg config

	flag.StringVar(&cfg.branch, "branch", "main", "default branch name")
	flag.StringVar(&cfg.remote, "remote", "origin", "local remote name")

	flag.Parse()

	cfg.from = flag.Arg(0)
	cfg.until = cmp.Or(flag.Arg(1), "HEAD")

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type config struct {
	branch string
	remote string

	from  string
	until string

	// dir allows overriding the current work directory,
	// which we make use of exclusively during testing.
	dir string
}

func (cfg config) git(args ...string) (string, error) {
	return git(cfg.dir, args...)
}

func run(cfg config) error {
	if cfg.from == "" {
		return fmt.Errorf("missing base reference")
	}

	base, empty, err := resolve(cfg)

	if err != nil {
		return fmt.Errorf("resolve diff base: %w", err)
	}

	if err := check(cfg, base, empty); err != nil {
		return fmt.Errorf("check diff base: %w", err)
	}

	delta, err := cfg.git("diff", "-z", "--no-renames", "--name-only", base, cfg.until)

	if err != nil {
		return fmt.Errorf("calculate path delta: %w", err)
	}

	bs := bufio.NewScanner(strings.NewReader(delta))
	bs.Split(scanNullLines)

	for bs.Scan() {
		fmt.Println(bs.Text())
	}

	if err := bs.Err(); err != nil {
		return fmt.Errorf("scan path delta: %w", err)
	}

	return nil
}

func check(cfg config, base string, empty bool) error {
	if empty {
		// NOTE: IMPORTANT: the Git command we use below to check for merge commits
		// does not work when diffing against an empty tree -- but the initial push
		// to the default branch is also the only case when the check is redundant.
		return nil
	}

	merges, err := cfg.git("rev-list", "--count", "--merges",
		fmt.Sprintf("%s..%s", base, cfg.until))

	if err != nil {
		return fmt.Errorf("check for merge commits: %w", err)
	}

	if merges != "0" {
		return fmt.Errorf("found %s merge commits", merges)
	}

	return nil
}

func resolve(cfg config) (string, bool, error) {
	branch, err := cfg.git("branch", "--show-current")

	if err != nil {
		return "", false, fmt.Errorf("resolve current branch: %w", err)
	}

	if cfg.from != zero {
		// NOTE: non-zero base reference (i.e., handling a push to an existing branch).

		// NOTE: verify that the diff base is a reachable ancestor of the current HEAD.
		// ... necessary to ensure that we detect history rewrites after force-pushing.

		if _, err := cfg.git("merge-base", "--is-ancestor", cfg.from, cfg.until); err != nil {
			return "", false, fmt.Errorf("verify that the base is a reachable ancestor: %w", err)
		}

		return cfg.from, false, nil
	}

	// NOTE: find a suitable diff base for the initial push to a new branch.

	if branch == cfg.branch {
		// NOTE: if we're handling the initial push to the default branch,
		// we diff against an empty tree (i.e., return all tracked files).

		base, err := cfg.git("hash-object", "-t", "tree", "/dev/null")

		if err != nil {
			return "", false, fmt.Errorf("compare against empty tree: %w", err)
		}

		return base, true, nil
	}

	// NOTE: if we're handling the initial push to some non-default branch,
	// we find the actual merge base and diff against the tree "inbetween".

	base, err := cfg.git("merge-base", cfg.until, fmt.Sprintf("%s/%s", cfg.remote, cfg.branch))

	if err != nil {
		return "", false, fmt.Errorf("compare against merge base: %w", err)
	}

	return base, false, nil
}

// git runs a Git command as a subprocess and returns its stdout,
// or an error wrapping its stderr output (or the process error).
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitError := new(exec.ExitError); errors.As(err, &exitError) {
			return "", fmt.Errorf("git: %s", strings.TrimSpace(stderr.String()))
		}

		return "", fmt.Errorf("git: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// scanNullLines is a [bufio.SplitFunc] that scans lines terminated by a null byte.
// The implementation is based on [bufio.ScanLines] with only minor modifications.
func scanNullLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, 0x00); i >= 0 {
		// NOTE: we have a full null-terminated line.
		return i + 1, data[0:i], nil
	}

	if atEOF {
		// NOTE: we have a final, non-terminated line.
		return len(data), data, nil
	}

	return 0, nil, nil // NOTE: request more data.
}
