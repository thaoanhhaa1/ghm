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

## Add an account

Inside the dashboard press `a`:

```
add github account
  b   login with browser   (gh auth login, recommended)
  t   paste a token        (scopes: repo, read:org, admin:public_key)
```

- `b` suspends the dashboard, runs `gh auth login -h github.com -p https -w`, then ghm detects the new keyring account and configures it: SSH key, key registration, `~/.ssh/config` host alias, per-account gitconfig, `includeIf`, folder.
- `t` accepts a PAT, pipes it to `gh auth login --with-token` (never stored by ghm), then runs the same setup.

The result screen lists every step plus warnings (for example a token without `admin:public_key`). The new account gets its own tab; run `ghm doctor` afterwards.

CLI equivalent for an account that is already logged in to `gh`:

```bash
ghm add <login>
```

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
| `ghm add <login>` | Configure one more account that is already logged in to `gh` |
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
| `ghm tui` | Dashboard: accounts as tabs, repos with ahead/behind, fetch/pull/push, PR, log |
| `ghm api <action>` | JSON API for the desktop app (`state`, `doctor`, `add`, `login-token`, `device-start`, `device-poll`, `git`) |
| `ghm completion zsh\|bash\|fish` | Shell completion script |
| `ghm install-shim` / `uninstall-shim` | Manage the `gh` wrapper and PATH entry |

Flags for the shim: `GHM_SHIM=0` disables it, an explicit `GH_TOKEN` always wins.

## Dashboard

```bash
ghm tui
```

```
ghm 0.7.0                                             gh active: anhthaodev
[1 anhthaodev 14] [2 thaoanhhaa1 16] [3 namphuongso 5]
────────────────────────────────────────────────────────────────────────────
REPO                        BRANCH                AHEAD  BEHIND  STATE
anhthaodev/misa             dev                   0      2       3 dirty
anhthaodev/it-tools-be      master                1      0       clean
────────────────────────────────────────────────────────────────────────────
j/k move  f fetch  p pull  P push  c PR  l log  a add  o open  / filter  tab account  ? help  q quit
```

`p` runs `git pull --ff-only`, `P` runs `git push`, `f` runs `git fetch --prune`, `o` opens the repo on github.com, `/` filters by name/branch/path, `?` opens help.

`c` creates a pull request: it pushes the current branch with the repo's account (`git push -u origin HEAD`), then runs `gh pr create` with that account's token. The PR URL is shown in the status line; failures (read-only repo, detached HEAD, `main` into `main`) stay in the dashboard.

`l` opens the last 60 commits of the selected repo (`j/k` scroll, `g/G` top/bottom, `esc` close).

## macOS app

A native SwiftUI window over the same engine (`ghm api …`), for people who would rather click than remember keys. Accounts, repositories, fetch/pull/push, health checks, and adding an account all work without a terminal.

```bash
make app                  # builds ~/.local/bin/ghm, bundles it, signs ad-hoc
open app/build/ghm.app
```

The UI follows the **iOS 26 Liquid Glass** language: a wallpaper of soft blue/indigo/purple blooms, three floating glass islands (top bar, accounts, repositories), a floating bottom action bar, capsule buttons and a selection lens that slides between accounts. Colour is dimmed behind the glass so white text stays readable; a generated `.icns` icon ships with the bundle (`make icon`). Requirements: **macOS 26+**, Xcode command line tools (`swift`), and Go. `make app` copies the `ghm` binary inside the bundle, so Finder launches work even though `~/.local/bin` is not on the GUI `PATH`, and embeds `Sparkle.framework` for updates.

- **Accounts island** — vivid avatar per login, repo counts, missing key/token badges, a glass selection lens, and Manage actions.
- **Repositories island** — custom list with repository, account avatar, branch, ahead/behind, dirty state. Click to select, double-click to open on GitHub, right-click for fetch/pull/push, reveal in Finder, copy path. `⌘R` rescans.
- **Top bar** — search, dirty/behind counters, rescan, health, *Add account*.
- **Bottom bar** — the selected checkout with Fetch / Pull / Push / GitHub / Health and a quick add button.
- **Health checks** — `ghm doctor`, plus a *Fix issues* button (`ghm doctor --fix`).
- **Add account…** (`⇧⌘N`) — two paths, both keeping credentials in the `gh` keyring:
  - *Browser*: device flow. The app shows the code, opens `github.com/login/device`, polls until you approve. No terminal, no copy-pasting commands.
  - *Token*: paste a PAT (masked field, piped to `gh auth login --with-token`, never stored by ghm). The *Create a token* link pre-fills `repo,read:org,admin:public_key`.
  - After login, ghm detects the new keyring account and writes SSH key, key registration, `Host github-<login>`, per-account gitconfig, `includeIf`, and `~/work/<login>`. Results are listed step by step.

The device flow defaults to the public GitHub CLI OAuth app client id so it works out of the box. For a distributed build, create your own OAuth App with **Device Flow** enabled and paste its client id under *OAuth client id*; the token flow needs no OAuth app at all.

On first launch macOS asks for access to **Documents/Desktop** because ghm also scans those folders for repositories; allow it to see every checkout (or set `scan_roots` in `accounts.yaml`).

`GHM_GITHUB_API` / `GHM_GITHUB_WEB` redirect the REST and OAuth endpoints (tests, GitHub Enterprise).

### Updates

The app ships with Sparkle 2:

- `SUFeedURL` → `https://github.com/thaoanhhaa1/ghm/releases/latest/download/appcast.xml`.
- `SUPublicEDKey` is pinned in `Info.plist`; every archive is signed with the Ed25519 key stored in the login keychain (`generate_keys`).
- Background checks run once a day (`SUScheduledCheckInterval`) with automatic download/install; *Check for Updates…* is in the app menu (`⇧⌘U`) and the sidebar.
- A staging feed can be tested with `GHM_UPDATE_FEED=http://127.0.0.1:8123/appcast.xml`; `GHM_UPDATE_CHANNEL` selects a Sparkle channel.

Publish a release (CI is optional here; this is the manual path):

```bash
GHM_VERSION=0.7.1 GHM_MAKE_RELEASE=1 ./app/scripts/bundle.sh release
gh release create v0.7.1 dist/ghm-0.7.1.zip dist/appcast.xml \
  --title "ghm 0.7.1" --notes "…"
```

`appcast.xml` must be uploaded with **every** release because the feed URL resolves through `releases/latest`. Back up the signing key once with `app/.build/artifacts/sparkle/Sparkle/bin/generate_keys -x ~/ghm-sparkle.key` (never commit it). Auto-update needs a stable signing identity for Gatekeeper; ad-hoc builds work locally but a Developer ID + notarization is required for distribution.

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

CI is **disabled** on purpose (`gh workflow disable ci.yml release.yml`): the GitHub
account is locked for a billing issue, so every push produced a red X in 3 seconds
without starting a single job. Run the same checks locally before pushing:

```bash
make check        # gofmt, go vet, go test, go build
make app          # rebuild app/build/ghm.app after Go changes
```

The app release is manual, because goreleaser cannot produce the `.app` (icon,
Sparkle framework, signature, appcast):

```bash
GHM_VERSION=0.8.0 GHM_MAKE_RELEASE=1 make app
gh release create v0.8.0 dist/ghm-0.8.0.zip dist/appcast.xml \
  --title "ghm 0.8.0" --notes-file /tmp/notes.md
```

`.github/workflows/release.yml` (goreleaser: darwin/linux tarballs, checksums,
Homebrew formula) is kept for the day billing is fixed; enable it with
`gh workflow enable release.yml`.


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
make check        # gofmt check + go vet + go test + build
make test         # unit tests: remote parsing, includeIf matching, config rendering
make build        # install to ~/.local/bin/ghm
```
