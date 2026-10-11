// Command cdw changes the shell's working directory to the git worktree that
// has a given branch checked out.
//
// A child process cannot change its parent shell's working directory, so the
// binary only resolves and prints the worktree path. The shell function
// emitted by `cdw init <shell>` performs the actual cd and wires up tab
// completion.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type worktree struct {
	path   string
	branch string // short branch name; empty for detached HEAD or bare
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cdw:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cdw <branch> | cdw init <bash|zsh>")
	}

	switch args[0] {
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("usage: cdw init <bash|zsh>")
		}
		return printInit(args[1])

	case "__complete":
		// Completion must never print errors into the user's prompt.
		wts, err := listWorktrees()
		if err != nil {
			return nil
		}
		for _, c := range completions(wts, args[1:]) {
			fmt.Println(c)
		}
		return nil

	case "-h", "--help", "help":
		fmt.Println(usage)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument (branch name)")
	}
	branch := args[0]

	wts, err := listWorktrees()
	if err != nil {
		return err
	}
	for _, wt := range wts {
		if wt.branch == branch {
			fmt.Println(wt.path)
			return nil
		}
	}
	return fmt.Errorf("no worktree has branch %q checked out", branch)
}

// completions returns the branch names matching the word being completed,
// best match first. args is `[--bash] [word]`.
//
// With --bash, the result is adjusted for readline, which replaces the word
// on the line with the longest common prefix of the candidates. When that
// prefix would discard part of what was typed, the word itself is added as a
// candidate so the common prefix collapses to nothing and readline leaves the
// line alone.
func completions(wts []worktree, args []string) []string {
	bash := len(args) > 0 && args[0] == "--bash"
	if bash {
		args = args[1:]
	}
	var word string
	if len(args) > 0 {
		word = args[0]
	}

	var branches []string
	for _, wt := range wts {
		if wt.branch != "" {
			branches = append(branches, wt.branch)
		}
	}
	matches := fuzzyFilter(word, branches)

	if bash && len(matches) > 1 && !isSubsequence(word, commonPrefix(matches)) {
		matches = append(matches, word)
	}
	return matches
}

// listWorktrees returns the worktrees of the git repo containing the current
// working directory.
func listWorktrees() ([]worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git worktree list: %s", msg)
	}
	return parsePorcelain(out), nil
}

// parsePorcelain parses `git worktree list --porcelain` output: blank-line
// separated records of "worktree <path>", "HEAD <sha>", "branch <ref>", etc.
func parsePorcelain(out []byte) []worktree {
	var (
		wts []worktree
		cur *worktree
	)
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			wts = append(wts, worktree{path: strings.TrimPrefix(line, "worktree ")})
			cur = &wts[len(wts)-1]
		case strings.HasPrefix(line, "branch ") && cur != nil:
			cur.branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	return wts
}

func printInit(shell string) error {
	switch shell {
	case "bash":
		fmt.Print(bashInit)
	case "zsh":
		fmt.Print(zshInit)
	default:
		return fmt.Errorf("unsupported shell %q (want bash or zsh)", shell)
	}
	return nil
}

const usage = `cdw - cd to the git worktree for a branch

Usage:
  cdw <branch>          cd to the worktree with <branch> checked out
  cdw init <bash|zsh>   print shell integration

Setup (add to your shell rc file):
  bash:  eval "$(command cdw init bash)"
  zsh:   eval "$(command cdw init zsh)"`

const shellFunc = `cdw() {
  case "$1" in
    init|__complete|-h|--help|help) command cdw "$@"; return ;;
  esac
  local dir
  dir="$(command cdw "$@")" || return
  # builtin: bypass cd wrappers like ` + "`zoxide init --cmd cd`" + `
  builtin cd -- "$dir"
}
`

const bashInit = shellFunc + `
_cdw_complete() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [ "$COMP_CWORD" -ne 1 ]; then
    COMPREPLY=()
    return
  fi
  local IFS=$'\n'
  COMPREPLY=($(command cdw __complete --bash "$cur" 2>/dev/null))
}
complete -o nosort -F _cdw_complete cdw 2>/dev/null || complete -F _cdw_complete cdw
`

const zshInit = shellFunc + `
_cdw() {
  (( CURRENT == 2 )) || return
  setopt localoptions extendedglob
  local word="$PREFIX$SUFFIX"
  local -a branches
  branches=(${(f)"$(command cdw __complete "$word" 2>/dev/null)"})
  # -U: the candidates are already fuzzy-filtered, so zsh must not re-filter
  # them by prefix. -V: keep the best-match-first order.
  compadd -U -V cdw -a branches
  # zsh replaces the word with the candidates' common prefix. Only allow that
  # when the prefix still fuzzy-matches the word; otherwise just list.
  local pat="*${(j:*:)${(@b)${(s::)word}}}*"
  [[ $word == *[[:upper:]]* ]] || pat="(#i)$pat"
  if (( compstate[nmatches] > 1 )) && [[ $compstate[unambiguous] != ${~pat} ]]; then
    compstate[insert]=
  fi
}
if (( $+functions[compdef] )); then
  compdef _cdw cdw
fi
`
