# cdw

`cd` to the git worktree that has a given branch checked out.

```sh
cdw feature/login   # jumps to whichever worktree has feature/login checked out
cdw main            # back to the main checkout
cdw <TAB>           # completes branch names of the current repo's worktrees
cdw gout<TAB>       # fuzzy: completes to feature/logout
```

Completion is fuzzy: the typed characters only need to appear in order
somewhere in the branch name, and the best matches are listed first. It's
case-insensitive unless the typed text contains an uppercase letter.

Works from anywhere inside the repo, including from inside another worktree.

## Install

```sh
go install github.com/brynbellomy/cdw@latest
```

Then add the shell integration to your rc file:

```sh
# ~/.bashrc
eval "$(command cdw init bash)"

# ~/.zshrc (after compinit)
eval "$(command cdw init zsh)"
```

Open a new shell (or `source` the rc file) and you're set.

## How it works

A program can't change the working directory of the shell that launched it, so
`cdw` has two parts:

- **The binary** runs `git worktree list --porcelain` in the current directory
  and prints the path of the worktree with the requested branch.
- **A shell function**, also named `cdw` and defined by `cdw init <shell>`,
  calls the binary and `cd`s to the path it prints. It also registers tab
  completion, which uses `cdw __complete` to list branch names.

Without the shell integration, running the binary directly just prints the
path, so `cd "$(command cdw my-branch)"` also works.

## Limitations

- Only switches to worktrees that already exist; it doesn't create them. Use
  `git worktree add` first.
- Worktrees on a detached HEAD have no branch name, so they can't be targeted
  or completed.
- Supports bash and zsh only.
