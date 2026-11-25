package app

import (
	"fmt"
	"os"
)

const ghmCommands = "init list status doctor repos sync adopt clone exec use token tui install-shim uninstall-shim completion version"

// cmdCompletion prints a shell completion script.
func cmdCompletion(args []string) int {
	if len(args) == 0 {
		fmt.Println("usage: ghm completion <zsh|bash|fish>")
		return 2
	}
	switch args[0] {
	case "zsh":
		fmt.Print(zshCompletion)
	case "bash":
		fmt.Print(bashCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Fprintf(os.Stderr, "ghm completion: unsupported shell %q\n", args[0])
		return 2
	}
	return 0
}

const zshCompletion = `#compdef ghm
# install: ghm completion zsh > "${fpath[1]}/_ghm"
_ghm() {
  local -a commands
  commands=(
    'init:bootstrap keys, ssh config and git config from gh accounts'
    'list:accounts overview'
    'status:accounts and repositories health'
    'doctor:run health checks (--fix to repair)'
    'repos:local repositories'
    'sync:fetch every repository'
    'adopt:rewrite a checkout to its account (--move)'
    'clone:clone with the resolved account'
    'exec:run a command with an account token'
    'use:print exports for the current shell'
    'token:print the account token'
    'tui:dashboard'
    'install-shim:install the gh wrapper'
    'uninstall-shim:remove the gh wrapper'
    'completion:print shell completion'
    'version:print version'
  )
  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi
  case $words[2] in
    adopt|repos|sync|clone|exec)
      _files
      ;;
    completion)
      _values 'shell' zsh bash fish
      ;;
    doctor)
      _values 'flag' --fix
      ;;
    list)
      _values 'flag' --check --json
      ;;
  esac
}
_ghm "$@"
`

const bashCompletion = `# install: ghm completion bash > /usr/local/etc/bash_completion.d/ghm
_ghm() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=($(compgen -W "` + ghmCommands + `" -- "$cur"))
    return
  fi
  case "$prev" in
    completion) COMPREPLY=($(compgen -W "zsh bash fish" -- "$cur")) ;;
    adopt|clone|exec|repos|sync) COMPREPLY=($(compgen -f -- "$cur")) ;;
  esac
}
complete -F _ghm ghm
`

const fishCompletion = `# install: ghm completion fish > ~/.config/fish/completions/ghm.fish
complete -c ghm -f
complete -c ghm -n "__fish_use_subcommand" -a init -d "bootstrap keys and config"
complete -c ghm -n "__fish_use_subcommand" -a list -d "accounts overview"
complete -c ghm -n "__fish_use_subcommand" -a status -d "accounts and repos health"
complete -c ghm -n "__fish_use_subcommand" -a doctor -d "health checks"
complete -c ghm -n "__fish_use_subcommand" -a repos -d "local repositories"
complete -c ghm -n "__fish_use_subcommand" -a sync -d "fetch every repository"
complete -c ghm -n "__fish_use_subcommand" -a adopt -d "adopt a checkout"
complete -c ghm -n "__fish_use_subcommand" -a clone -d "clone with the resolved account"
complete -c ghm -n "__fish_use_subcommand" -a exec -d "run with an account token"
complete -c ghm -n "__fish_use_subcommand" -a use -d "print shell exports"
complete -c ghm -n "__fish_use_subcommand" -a token -d "print account token"
complete -c ghm -n "__fish_use_subcommand" -a tui -d "dashboard"
complete -c ghm -n "__fish_use_subcommand" -a install-shim -d "install gh wrapper"
complete -c ghm -n "__fish_use_subcommand" -a uninstall-shim -d "remove gh wrapper"
complete -c ghm -n "__fish_use_subcommand" -a completion -d "shell completion"
complete -c ghm -n "__fish_use_subcommand" -a version -d "print version"
complete -c ghm -n "__fish_seen_subcommand_from completion" -a "zsh bash fish"
`
