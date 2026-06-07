# x/cmd/semver

> Program `semver` implements a simple interface for creating Git tags conforming to the [Semantic Versioning](https://semver.org/spec/v2.0.0.html) standard.

## Usage

> `semver [--branch=<default branch> --remote=<remote name>] <major|minor|patch>`

`semver` will perform a sanity check of the local repository state and error out in the following conditions:

- the currently checked out branch is not the default branch (override using the `--branch` flag)
- the local branch is behind its remote tracking branch (checked after fetching from the remote)
- the worktree is in a dirty state (i.e., any non-ignored untracked or changed files are present)
- the index (staging area) is in a "dirty" state (i.e., changes are staged but not yet committed)

## Known Limitations

- `semver` discards any prerelease or build information when parsing existing Git tags
- `semver` does not support adding prerelease or build information when creating new Git tags
- `semver` assumes that the Git commit that carries the latest tag only carries a single tag
- `semver` will sign the newly created Git tag if(f) `tag.gpgSign` is set in the Git config
  - Note that `semver` will always validate the signature (if any) after creating the Git tag
