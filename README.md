# ghm

One machine, many GitHub accounts, zero `gh auth switch`.

`ghm` binds every GitHub account to its own SSH key **and** its own folder. Git, your IDE, scripts and `gh` then pick the right account from the path you are in — no login, no switching, no token juggling.

```
~/work/anhthaodev/    -> key id_ed25519_anhthaodev    -> git@github-anhthaodev:...
~/work/thaoanhhaa1/   -> key id_ed25519_thaoanhhaa1   -> git@github-thaoanhhaa1:...
~/work/namphuongso/   -> key id_ed25519_namphuongso  -> git@github-namphuongso:...
```

## Why

`gh auth git-credential` returns the token of the **active** account for every `github.com` repo over https. With several accounts that means `gh auth switch` before each push, and silent pushes with the wrong identity if you forget. `ghm` removes that code path:

1. **Transport** – one ed25519 key per account, selected by `core.sshCommand` + `url.insteadOf` from `includeIf gitdir`.
2. **Identity** – per-folder `user.name` / `user.email` (GitHub noreply), so commits are attributed correctly.
3. **API** – a `gh` shim injects `GH_TOKEN` for the folder's account, so `gh pr/issue/api` follow the repo, not the global active account.

Only native config is used. If the `ghm` binary is deleted, git still uses the right account.

## Install

```bash
go build -o ~/.local/bin/ghm ./cmd/ghm
ghm install-shim          # writes ~/.local/bin/gh wrapper + PATH line
ghm doctor                # 23 checks
```

New machine, start to finish:

```bash
gh auth login             # once per account (browser)
gh auth switch -u <next>; gh auth login
ghm init                  # reads ~/.config/gh/hosts.yml, creates keys,
                          # registers them, writes ssh + git config
ghm install-shim
ghm doctor
```

`ghm init` is idempotent and never rewrites files it does not own.

## Daily use

```bash
cd ~/work/thaoanhhaa1/it-tool
git pull                  # key thaoanhhaa1
git push                  # key thaoanhhaa1, correct author
gh pr create              # token thaoanhhaa1 (shim)
ghm status                # accounts + repo health
```

Add a repository:

```bash
ghm clone owner/repo             # resolves the account with push access
ghm adopt ~/old/path --move      # migrate an existing checkout into ~/work/<login>/
ghm adopt --all --dry-run        # preview a bulk migration
```

## Commands

| Command | Purpose |
|---|---|
| `ghm init` | Bootstrap from the accounts already in `gh` (keys, ssh config, git config, folders) |
| `ghm list [--check] [--json]` | Accounts table; `--check` also tests SSH |
| `ghm status` | Accounts + repository health, exit 1 on problems |
| `ghm doctor [--fix]` | 23 checks, `--fix` repairs keys, config, https remotes, identities, shim |
| `ghm repos [--account X] [--json]` | Local repos with branch, ahead/behind, dirty |
| `ghm sync [--account X] [--jobs N]` | Parallel `git fetch --prune` |
| `ghm clone <owner/repo> [--account X] [--dir P]` | Clone with the resolved account |
| `ghm adopt <path...> [--move] [--dry-run]` | Rewrite remote to SSH, set identity, optionally move |
| `ghm exec <login> -- cmd` | Run anything with that account's token + SSH command |
| `eval "$(ghm use <login>)"` | Same, for the current shell |
| `ghm token <login>` | Print the keyring token |
| `ghm tui` | Dashboard: accounts as tabs, repos with ahead/behind, fetch/pull/push |
| `ghm completion zsh\|bash\|fish` | Shell completion script |
| `ghm install-shim` / `uninstall-shim` | Manage the `gh` wrapper and PATH entry |

Flags for the shim: `GHM_SHIM=0` disables it, an explicit `GH_TOKEN` always wins.

## Dashboard

```bash
ghm tui
```

```
ghm 0.3.0                                             gh active: anhthaodev
[1 anhthaodev 14] [2 thaoanhhaa1 15] [3 namphuongso 5]
────────────────────────────────────────────────────────────────────────────
REPO                        BRANCH                AHEAD  BEHIND  STATE
anhthaodev/misa             dev                   0      2       3 dirty
anhthaodev/it-tools-be      master                1      0       clean
────────────────────────────────────────────────────────────────────────────
j/k move  f fetch  p pull  P push  o open  / filter  1-3 account  ? help  q quit
```

`p` runs `git pull --ff-only`, `P` runs `git push`, `f` runs `git fetch --prune`, `o` opens the repo on github.com, `/` filters by name/branch/path, `?` opens help.

## Shell completion

```bash
# zsh
ghm completion zsh > "${fpath[1]}/_ghm"

# bash
ghm completion bash > /usr/local/etc/bash_completion.d/ghm

# fish
ghm completion fish > ~/.config/fish/completions/ghm.fish
```

## Release

- `ci.yml` runs gofmt, `go vet`, `go test`, cross-platform build on Ubuntu + macOS.
- Tagging `v*` runs goreleaser: darwin/linux (amd64/arm64) tarballs, checksums, Homebrew formula push to `thaoanhhaa1/homebrew-tap` (needs `HOMEBREW_TAP_GITHUB_TOKEN`).

```bash
git tag v0.3.0 && git push --tags
```


## Files written

| Path | Content |
|---|---|
| `~/.config/ghm/accounts.yaml` | accounts, folder roots, default account (no secrets) |
| `~/.config/ghm/git/<login>.gitconfig` | `user.*`, `core.sshCommand`, `url.insteadOf` |
| `~/.gitconfig` | `includeIf gitdir:~/work/<login>/` blocks |
| `~/.ssh/config` | `Host github-<login>` blocks |
| `~/.local/bin/gh` | shim that maps folder → `GH_TOKEN` |

Backups are written as `<file>.bak-<timestamp>` before any edit.

## Account resolution

`ghm clone` / `ghm adopt` ask the API per account: the account with `permissions.push` wins, preferring the account that owns the repository. If nobody has write access, the repo is cloned read-only with the first account that can read it. Results are cached in `~/.config/ghm/repo-accounts.tsv`.

## Troubleshooting

**`cannot exec .../askpass/askpass.sh`** — a broken IDE extension (Git Graph) exports `GIT_ASKPASS` to a missing file, and some repo is still on `https`. Fix the repo with `ghm adopt --all`, then reload the IDE window so it drops the stale env.

**Org repos deny the key** — the org enforces SAML SSO: authorize the SSH key in the org's settings. `ghm doctor` shows `Permission denied (publickey)` for that account.

**`gh auth status` shows the wrong active account** — that is harmless for git now; it only matters for `gh` outside `~/work`. Inside a folder the shim overrides it.

**New shell does not see `~/.local/bin`** — `ghm install-shim` appends the PATH line once; open a new terminal, or `exec zsh -l`.

## Development

```bash
go test ./...     # unit tests: remote parsing, includeIf matching, config rendering
go vet ./...
go build -o ~/.local/bin/ghm ./cmd/ghm
```
