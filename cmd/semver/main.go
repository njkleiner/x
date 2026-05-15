package main

import (
	"bytes"
	"cmp"
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

	cfg.args = flag.Args()
	cfg.dir, _ = os.Getwd()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type config struct {
	branch string
	remote string

	args []string
	dir  string
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

		v, err = parse(latest)

		if err != nil {
			return fmt.Errorf("parse latest tag (%q): %w", latest, err)
		}
	}

	switch arg := cmp.Or(cfg.args...); arg {
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
		return fmt.Errorf("syntax: %s <major|minor|patch>", os.Args[0])
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

	log, err := cfg.git("log", "-1", "--patch")

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
		if _, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("git: %s", strings.TrimSpace(stderr.String()))
		}

		return "", fmt.Errorf("git: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
