// Program semver implements a simple interface for creating
// Git tags conforming to the Semantic Versioning standard.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	var cfg config

	flag.StringVar(&cfg.branch, "branch", "main", "default branch name")
	flag.StringVar(&cfg.remote, "remote", "origin", "local remote name")

	flag.Parse()

	cfg.dir, _ = os.Getwd()

	switch arg := flag.Arg(0); arg {
	case "major", "minor", "patch":
		cfg.arg = arg // NOTE: propagate known-valid "bump" operation type.
	default:
		fmt.Fprintln(os.Stderr, "syntax: <major|minor|patch>")
		os.Exit(1) // TODO: THINK: use a different exit code?
	}

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type config struct {
	branch string
	remote string

	arg string
	dir string
}

func (cfg config) git(args ...string) (string, error) {
	return git(cfg.dir, args...)
}

func run(cfg config) error {
	branch, err := cfg.git("symbolic-ref", "--short", "HEAD")

	if err != nil {
		return fmt.Errorf("check current branch: %w", err)
	}

	if branch != cfg.branch {
		return fmt.Errorf("current branch is %q (must be on %q)", branch, cfg.branch)
	}

	if _, err := cfg.git("fetch", cfg.remote); err != nil {
		return fmt.Errorf("fetch from remote: %w", err)
	}

	local, err := cfg.git("rev-parse", "HEAD")

	if err != nil {
		return fmt.Errorf("resolve HEAD: %w", err)
	}

	upstream, err := cfg.git("rev-parse", "@{u}")

	if err != nil {
		return fmt.Errorf("resolve upstream: %w", err)
	}

	if local != upstream {
		return fmt.Errorf("local branch is not up to date with upstream")
	}

	status, err := cfg.git("status", "--porcelain")

	if err != nil {
		return fmt.Errorf("check worktree status: %w", err)
	}

	if status != "" {
		return fmt.Errorf("worktree is dirty")
	}

	// NOTE: pretend "v0.0.0" is the current tag
	// if we find that no Git tags exist at all.
	var v version

	list, err := cfg.git("tag", "--list")

	if err != nil {
		return fmt.Errorf("show existing tags: %w", err)
	}

	if list != "" {
		latest, err := cfg.git("describe", "--tags", "--abbrev=0")

		if err != nil {
			return fmt.Errorf("describe latest tag: %w", err)
		}

		commit, err := cfg.git("rev-list", "-1", latest)

		if err != nil {
			return fmt.Errorf("resolve commit: %w", err)
		}

		all, err := cfg.git("tag", "--points-at", commit)

		if err != nil {
			return fmt.Errorf("show other tags: %w", err)
		}

		if all != latest {
			// NOTE: if the commit references more than just the "latest" tag,
			// we are not guaranteed to have found the correct "latest" tag...
			return fmt.Errorf("commit %q references multiple tags", commit)
		}

		v, err = parse(latest)

		if err != nil {
			return fmt.Errorf("parse latest tag (%q): %w", latest, err)
		}
	}

	switch cfg.arg {
	case "major":
		v.major++
		v.minor = 0
		v.patch = 0
	case "minor":
		v.minor++
		v.patch = 0
	case "patch":
		v.patch++
	default:
		// NOTE: for future-safety (actually unreachable).
		return fmt.Errorf("unknown argument: %q", cfg.arg)
	}

	tag := v.String()

	existing, err := cfg.git("tag", "--list", tag)

	if err != nil {
		return fmt.Errorf("check for conflicting tag: %w", err)
	}

	if existing != "" {
		return fmt.Errorf("tag already exists: %q", tag)
	}

	if _, err := cfg.git("tag", "--annotate", tag, "--message", tag); err != nil {
		return fmt.Errorf("create annotated tag: %w", err)
	}

	log, err := cfg.git("show", "--no-patch", "--show-signature", tag)

	if err != nil {
		return fmt.Errorf("show log for new tag: %w", err)
	}

	fmt.Println(log)

	return nil
}

type version struct {
	major, minor, patch int
}

func (v version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
}

var tagRE = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)([-+]|$)`)

// parse parses a v-prefixed SemVer tag (e.g., "v1.2.3" or "v1.2.3-alpha+build")
// into a struct holding the three numeric components -- major, minor and patch.
func parse(tag string) (version, error) {
	m := tagRE.FindStringSubmatch(tag)

	if len(m) == 0 {
		return version{}, fmt.Errorf("invalid semantic version: %q", tag)
	}

	var v version

	v.major, _ = strconv.Atoi(m[1])
	v.minor, _ = strconv.Atoi(m[2])
	v.patch, _ = strconv.Atoi(m[3])

	if v == (version{}) && !(m[1] == "0" && m[2] == "0" && m[3] == "0") {
		// NOTE: a zero version is treated as a parse failure, unless all three components
		// were explicitly zero in the tag input (e.g., "v0.0.0" or "v0.0.0-alpha+build").
		return version{}, fmt.Errorf("invalid semantic version: %q", tag)
	}

	return v, nil
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
