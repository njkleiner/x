package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repository is a helper for setting up and manipulating a temporary Git repository.
type repository struct {
	t *testing.T

	local string
}

// create creates a temporary bare Git repository.
func create(t *testing.T) *repository {
	t.Helper()

	r := &repository{t: t, local: t.TempDir()}
	r.git("init", "--bare", "--initial-branch", "main")

	return r
}

// clone clones a remote repository into a temporary local working copy.
func clone(t *testing.T, remote *repository) *repository {
	t.Helper()

	r := &repository{t: t, local: t.TempDir()}

	r.git("clone", remote.local, ".")
	r.git("branch", "-M", "main")

	r.git("config", "user.email", "author@domain.invalid")
	r.git("config", "user.name", "Test Author")

	return r
}

func (r *repository) git(args ...string) string {
	r.t.Helper()

	out, err := git(r.local, args...)

	if err != nil {
		r.t.Fatal(err)
	}

	return out
}

// commit creates an empty commit in the local working copy.
func (r *repository) commit(message string) {
	r.t.Helper()
	r.git("commit", "--allow-empty", "--message", message)
}

// tag creates an annotated tag in the local working copy.
func (r *repository) tag(name string) {
	r.t.Helper()
	r.git("tag", "--annotate", name, "--message", name)
}

// push pushes the local working copy to the remote.
func (r *repository) push() {
	r.t.Helper()
	r.git("push", "--follow-tags", "origin", "main")
}

// config returns a config for the test repository.
func (r *repository) config(arg string) config {
	return config{branch: "main", remote: "origin", arg: arg, dir: r.local}
}

func TestRun(t *testing.T) {
	t.Run("successful version bump", func(t *testing.T) {
		tests := []struct {
			name   string
			latest string // empty means no prior tag
			arg    string
			want   string
		}{
			{name: "patch", latest: "v1.2.3", arg: "patch", want: "v1.2.4"},
			{name: "minor", latest: "v1.2.3", arg: "minor", want: "v1.3.0"},
			{name: "major", latest: "v1.2.3", arg: "major", want: "v2.0.0"},

			{name: "patch without previous tag", latest: "", arg: "patch", want: "v0.0.1"},
			{name: "minor without previous tag", latest: "", arg: "minor", want: "v0.1.0"},
			{name: "major without previous tag", latest: "", arg: "major", want: "v1.0.0"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := clone(t, create(t))
				r.commit("initial commit")

				if tt.latest != "" {
					r.tag(tt.latest)
				}

				r.push()

				if err := run(r.config(tt.arg)); err != nil {
					t.Fatalf("should bump version: err=%v", err)
				}

				out := r.git("tag", "--list", tt.want)

				if got := strings.TrimSpace(out); got != tt.want {
					t.Fatalf("should have created tag: got=%q", got)
				}
			})
		}
	})

	t.Run("invalid repository state", func(t *testing.T) {
		tests := []struct {
			name string

			setup func(t *testing.T, r, remote *repository)

			arg string

			// TODO: compare actual error against expected message
		}{
			{
				name: "wrong branch",

				setup: func(t *testing.T, r, remote *repository) {
					r.git("checkout", "-b", "feature/foo")
				},

				arg: "patch",
			},
			{
				name: "dirty worktree",

				setup: func(t *testing.T, r, remote *repository) {
					name := filepath.Join(r.local, "dirty.txt")
					data := []byte("dirty")

					if err := os.WriteFile(name, data, 0644); err != nil {
						t.Fatalf("write file (name=%q): err=%v", name, err)
					}
				},

				arg: "patch",
			},
			{
				name: "non-empty index",

				setup: func(t *testing.T, r, remote *repository) {
					name := filepath.Join(r.local, "dirty.txt")
					data := []byte("dirty")

					if err := os.WriteFile(name, data, 0644); err != nil {
						t.Fatalf("write file (name=%q): err=%v", name, err)
					}

					// NOTE: we stage the file (but do not commit).
					r.git("add", name)
				},

				arg: "patch",
			},
			{
				name: "out of date",

				setup: func(t *testing.T, r, remote *repository) {
					r2 := clone(t, remote)
					r2.commit("remote commit")
					r2.push()
				},

				arg: "patch",
			},
			{
				name: "conflicting tags",

				setup: func(t *testing.T, r, remote *repository) {
					r.tag("v0.1.0")
					r.tag("v0.2.0")
					r.push()
				},

				arg: "patch",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				remote := create(t)

				r := clone(t, remote)
				r.commit("initial commit")
				r.push()

				if tt.setup != nil {
					tt.setup(t, r, remote)
				}

				if err := run(r.config(tt.arg)); err == nil {
					t.Fatalf("should return error: err=%v", err)
				}
			})
		}
	})
}
