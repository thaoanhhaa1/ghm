# ghm — Quản lý nhiều tài khoản GitHub trên 1 máy

Mục tiêu: 3 account GitHub (`anhthaodev`, `thaoanhhaa1`, `namphuongso`) dùng song song, **không bao giờ phải `gh auth login` / `gh auth switch`** khi pull/push. Mỗi thư mục/repo gắn cứng 1 account, git và `gh` tự chọn đúng.

---

## 1. Chẩn đoán hiện trạng (đã kiểm tra máy)

| Hạng mục | Hiện tại | Vấn đề |
|---|---|---|
| `gh` | 2.102.0, 3 account trong keyring, active = `anhthaodev` | `gh` chỉ có 1 account active tại 1 thời điểm |
| `git_protocol` | `https` | Mọi repo push bằng token của account active |
| `~/.gitconfig` | `credential.https://github.com.helper = !gh auth git-credential` | **Nguyên nhân gốc**: helper trả token active account cho *mọi* repo github.com → push sai account |
| SSH | chưa có key nào (`~/.ssh` chỉ có `known_hosts`) | Chưa tận dụng được định danh theo key |
| Remote | `https://github.com/...` rải rác nhiều owner | Không có thông tin account trong remote |
| Repo | nằm rải rác trong `~/Documents/IT/Personal/Projects/` | Không map được account theo đường dẫn |

**Kết luận**: không cần "app login lại". Cần 2 lớp cấu hình native + 1 CLI mỏng:

1. **SSH key riêng cho từng account** → GitHub định danh account bằng key, nên quyền push/pull đúng tuyệt đối, không token, không hết hạn.
2. **`includeIf gitdir`** → mỗi thư mục gốc của account tự nạp `user.name`, `user.email`, `core.sshCommand`, `url.insteadOf`. Git tự đúng account kể cả khi bạn không dùng app (IDE, VS Code, git CLI, script, Docker).
3. **CLI `ghm`** → lớp phủ giúp `gh` CLI (pr/issue/api/run) cũng đúng account, và các thao tác clone/adopt/doctor/tui.

Triết lý thiết kế: **app không giữ secret, không tự login**. App chỉ sinh/sửa config chuẩn của git + ssh + gh, và gọi `gh auth token --user <login>` khi cần. Nếu xoá app, git vẫn chạy đúng account.

---

## 2. Kiến trúc

```
                    ┌─────────────────────────────┐
   bạn gõ lệnh  ──▶ │  ghm (Go binary)            │
                    │  init / list / adopt /      │
                    │  clone / exec / doctor / tui│
                    └───────────┬─────────────────┘
                                │ sinh & kiểm tra config (không giữ secret)
        ┌───────────────────────┼────────────────────────┐
        ▼                       ▼                        ▼
 ~/.ssh/config          ~/.gitconfig              ~/.config/gh/hosts.yml
 Host github-<login>    includeIf gitdir:...      3 account (đã có sẵn)
 IdentityFile <key>     → ~/.config/ghm/git/<login>.gitconfig
                        (user.*, core.sshCommand,
                         url.insteadOf)

        ▼                       ▼                        ▼
   ssh -T github-X        git pull/push             GH_TOKEN=$(gh auth token
   → "Hi <login>!"        → đúng key/account         --user X) gh pr create
```

Ba trục độc lập, hỏng 1 trục vẫn còn 2 trục:

| Trục | Cơ chế | Bảo vệ khi |
|---|---|---|
| Git transport | `core.sshCommand` + `url.insteadOf` theo thư mục | luôn luôn, kể cả app không chạy |
| Git identity | `user.name`/`user.email` theo thư mục + `user.useConfigOnly=true` | tránh commit sai email/attribute |
| GitHub API (`gh`) | `GH_TOKEN` theo thư mục (shim/`ghm exec`) | `gh pr/issue/api` đúng account |

---

## 3. Mô hình dữ liệu

### 3.1 Thư mục

```
~/work/                          # root, cấu hình được
├── anhthaodev/                  # account 1
│   ├── misa/
│   └── it-tools-be/
├── thaoanhhaa1/                 # account 2
│   ├── ai-switcher/
│   ├── it-tool/
│   └── moni-split-bill/
└── namphuongso/                 # account 3
    └── ...
```

Quy tắc: **account là thư mục cấp 1**, repo nằm dưới (có thể thêm cấp owner nếu trùng tên). Repo của người khác mà bạn là collaborator (vd `quocvan1999/AI-Gate`, `nguyenhuong9x/FoodShare`) vẫn đặt dưới account *bạn dùng để push*, không phải theo owner.

### 3.2 State file `~/.config/ghm/accounts.yaml`

```yaml
version: 1
root: ~/work
default_account: anhthaodev
gh_binary: /opt/homebrew/bin/gh
accounts:
  - login: anhthaodev
    name: "Thao Anh (work)"
    email: "12345678+anhthaodev@users.noreply.github.com"
    key: ~/.ssh/id_ed25519_anhthaodev
    host_alias: github-anhthaodev
    dir: ~/work/anhthaodev
  - login: thaoanhhaa1
    name: "Thao Anh"
    email: "98765432+thaoanhhaa1@users.noreply.github.com"
    key: ~/.ssh/id_ed25519_thaoanhhaa1
    host_alias: github-thaoanhhaa1
    dir: ~/work/thaoanhhaa1
  - login: namphuongso
    name: "Nam Phuong"
    email: "11112222+namphuongso@users.noreply.github.com"
    key: ~/.ssh/id_ed25519_namphuongso
    host_alias: github-namphuongso
    dir: ~/work/namphuongso
```

Không chứa token. Email lấy từ `gh api user --jq .email`; nếu null (private) thì dùng noreply `ID+login@users.noreply.github.com` (`ID` = `gh api user --jq .id`).

### 3.3 Config sinh ra

`~/.config/ghm/git/anhthaodev.gitconfig`:

```gitconfig
[user]
    name = Thao Anh (work)
    email = 12345678+anhthaodev@users.noreply.github.com
[core]
    sshCommand = ssh -i ~/.ssh/id_ed25519_anhthaodev -o IdentitiesOnly=yes
[url "git@github-anhthaodev:"]
    insteadOf = https://github.com/
```

`~/.gitconfig` (thêm 1 lần):

```gitconfig
[user]
    useConfigOnly = true          # chặn commit khi repo không khớp account nào
[includeIf "gitdir:~/work/anhthaodev/"]
    path = ~/.config/ghm/git/anhthaodev.gitconfig
[includeIf "gitdir:~/work/thaoanhhaa1/"]
    path = ~/.config/ghm/git/thaoanhhaa1.gitconfig
[includeIf "gitdir:~/work/namphuongso/"]
    path = ~/.config/ghm/git/namphuongso.gitconfig
```

`~/.ssh/config` (3 block):

```sshconfig
Host github-anhthaodev
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_anhthaodev
    IdentitiesOnly yes
    AddKeysToAgent yes
    UseKeychain yes
```

Vì sao cần cả `core.sshCommand` **và** alias:
- alias + `insteadOf` xử lý remote `https://github.com/...` (đa số repo hiện tại).
- `core.sshCommand` xử lý remote `git@github.com:...` (không bị `insteadOf` rewrite) — đảm bảo không rơi về key mặc định.

### 3.4 Bỏ đường dẫn gây lỗi âm thầm

```bash
git config --global --unset-all credential.https://github.com.helper
git config --global --unset-all credential.https://gist.github.com.helper
```

Lý do: giữ helper này thì bất kỳ remote https nào lọt lưới sẽ push bằng account active mà **không báo lỗi**. Bỏ đi, git sẽ fail rõ ràng thay vì push sai. `ghm doctor` cảnh báo nếu `gh auth setup-git` thêm lại.

---

## 4. Spec CLI `ghm`

Go 1.27 + `spf13/cobra`, TUI bằng `charmbracelet/bubbletea`. Binary đơn, không runtime.

| Lệnh | Việc | Output/Exit |
|---|---|---|
| `ghm init` | wizard: đọc `hosts.yml` lấy 3 login → tạo key nếu thiếu → `GH_TOKEN=$(gh auth token -u X) gh ssh-key add` → ghi `~/.ssh/config`, `~/.gitconfig` (backup `.bak`), 3 file `.gitconfig`, `accounts.yaml` → tạo `~/work/<login>` | idempotent, chạy lại an toàn |
| `ghm list` | bảng: login, email, key, ssh ok?, gh auth ok?, số repo, commit gần nhất | `--json` |
| `ghm status` | như `list` + account active của `gh`, cảnh báo helper, remote https chưa migrate | exit 1 nếu có lỗi |
| `ghm use <login>` | in `export GH_TOKEN=...` / hoặc ghi `.ghm/current` cho shim; **không** đổi `gh auth switch` | dùng với `eval "$(ghm use X)"` |
| `ghm token <login>` | `gh auth token --user` | stdout token |
| `ghm exec <login> -- <cmd...>` | chạy cmd với `GH_TOKEN`, `GIT_SSH_COMMAND`, `GIT_AUTHOR_*` đúng account | passthrough exit code |
| `ghm clone <owner/repo> [--account X]` | dò token nào có quyền (`GET /repos/{o}/{r}`), chọn account, `git clone git@github-<login>:o/r.git ~/work/<login>/r` | hỏi nếu nhiều account có quyền |
| `ghm adopt <path> [--account X] [--move]` | đọc `origin`, xác định account, rewrite remote sang SSH alias, `--move` chuyển vào `~/work/<login>/` | `--dry-run` mặc định |
| `ghm repos` | liệt kê repo local theo account, ahead/behind vs upstream | `--json` |
| `ghm sync` | `git fetch` song song tất cả repo | |
| `ghm doctor [--fix]` | 8 check: ssh -T từng alias, key có trong account, includeIf tồn tại, helper bị xoá, remote lạ, email trùng giữa account, `useConfigOnly`, quyền ghi `~/work/*` | exit code theo mức độ |
| `ghm tui` | dashboard: cột account → repo, phím `p` pull, `P` push, `o` mở web, `c` `gh pr create`, `/` search | optional |

### 4.1 Shim `gh` (tuỳ chọn, Phase 3)

`~/.local/bin/gh` đứng trước `/opt/homebrew/bin` trong `PATH`:

```
đọc cwd → nếu nằm trong ~/work/<login>/ và lệnh không phải auth/config
        → GH_TOKEN=$(gh auth token --user <login>) exec /opt/homebrew/bin/gh "$@"
ngược lại → exec /opt/homebrew/bin/gh "$@"
```

Guardrails: không shim `gh auth *`, `gh config *`, `gh extension *`; log account đã chọn ra stderr khi `GHM_DEBUG=1`; tắt bằng `GHM_SHIM=0`.

Thay thế nhẹ hơn (không shim): `direnv` + `.envrc` mỗi thư mục account:

```bash
export GH_TOKEN="$(gh auth token --user anhthaodev)"
export GIT_SSH_COMMAND="ssh -i ~/.ssh/id_ed25519_anhthaodev -o IdentitiesOnly=yes"
```

### 4.2 Cách app "login tất cả account" mà không login lại

Account đã nằm trong keyring. `ghm init` chỉ cần, cho mỗi login:

1. `gh auth token --user <login>` → lấy token có sẵn (không login lại).
2. `gh ssh-key add <key>.pub --title "macbook-<login>"` với token đó → add public key vào đúng account.
3. Test `ssh -T git@github-<login>` → phải trả `Hi <login>!`.

Nếu account nào chưa có trong keyring mới phải `gh auth login` 1 lần. Sau đó không bao giờ nữa.

---

## 5. Cấu trúc mã nguồn

```
ghm/
├── PLAN.md
├── go.mod
├── cmd/ghm/main.go
├── internal/
│   ├── account/        # load/save accounts.yaml, resolve login ↔ dir
│   ├── github/         # gh CLI wrapper: token, api user, ssh-key add
│   ├── sshcfg/         # parse + merge ~/.ssh/config (giữ nguyên block khác)
│   ├── gitcfg/         # ghi *.gitconfig, sửa ~/.gitconfig có backup
│   ├── repo/           # scan repo, remote parse, ahead/behind, rewrite remote
│   ├── doctor/         # các check + fix
│   └── tui/            # bubbletea
└── testdata/           # fixture ssh config, gitconfig, hosts.yml
```

Nguyên tắc code: mọi thao tác ghi file đều **backup + idempotent**; không parse secret từ keyring; chỉ đọc `hosts.yml` phần `users`.

---

## 6. Lộ trình

### Phase 0 — Fix ngay, không cần code (≈45 phút)

Làm tay để có kết quả tức thì; Phase 1 sẽ tự động hoá lại đúng các bước này.

```bash
# 1. key cho từng account (không passphrase, hoặc dùng agent + UseKeychain)
for a in anhthaodev thaoanhhaa1 namphuongso; do
  ssh-keygen -t ed25519 -C "$a@macbook" -f ~/.ssh/id_ed25519_$a -N ""
done

# 2. add public key vào đúng account
for a in anhthaodev thaoanhhaa1 namphuongso; do
  GH_TOKEN=$(gh auth token --user $a) gh ssh-key add ~/.ssh/id_ed25519_$a.pub --title "macbook-$a"
done

# 3. thêm alias vào ~/.ssh/config (nội dung mục 3.3), rồi test — phải ra "Hi <login>!"
for a in anhthaodev thaoanhhaa1 namphuongso; do
  ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519_$a -T git@github.com 2>&1 | head -1
done

# 4. bỏ helper gây push sai
git config --global --unset-all credential.https://github.com.helper
git config --global --unset-all credential.https://gist.github.com.helper
```

Sau đó: tạo `~/work/<login>/`, chuyển repo vào, `ghm`-style rewrite remote:

```bash
git -C <repo> remote set-url origin git@github-anhthaodev:owner/repo.git
```

**Kết quả Phase 0**: hết phải `gh auth switch` cho pull/push. `gh pr create` vẫn cần chọn account → Phase 2.

### Phase 1 — MVP CLI (1.5–2 ngày)

`init`, `list`, `status`, `doctor --fix`, `adopt`, `clone`.
Acceptance: chạy `ghm init` trên máy hiện tại → `ghm doctor` xanh 8/8; `ghm adopt --move` migrate 6 repo local.

### Phase 2 — `gh` đúng account (1 ngày)

`exec`, `use`, `token`, shim `gh` (hoặc `.envrc`), `repos`, `sync`.
Acceptance: trong `~/work/thaoanhhaa1/moni-split-bill`, `gh pr list` trả repo của `thaoanhhaa1` dù `gh auth status` active là `anhthaodev`.

### Phase 3 — TUI (2–3 ngày)

Dashboard account/repo, pull/push, tạo PR, mở web, search, log hoạt động.

### Phase 4 — Đóng gói (0.5 ngày)

`brew` tap hoặc `go install`; shell completion; `ghm doctor` trong CI pre-commit; README + demo.

---

## 7. Kịch bản kiểm thử

| # | Test | Kỳ vọng |
|---|---|---|
| 1 | `ghm doctor` sau init | 8/8 pass, exit 0 |
| 2 | `cd ~/work/thaoanhhaa1/it-tool && git push` | commit author = email thaoanhhaa1; push vào repo bằng key thaoanhhaa1 |
| 3 | `cd ~/work/anhthaodev/misa && git push` khi `gh` active = thaoanhhaa1 | vẫn đúng anhthaodev (SSH quyết định) |
| 4 | Trong repo thaoanhhaa1: `git remote get-url origin` | `git@github-thaoanhhaa1:thaoanhhaa1/it-tool.git` |
| 5 | Clone repo private của namphuongso bằng `ghm clone` | resolve đúng account, clone vào `~/work/namphuongso/` |
| 6 | `ghm exec namphuongso -- gh api user -q .login` | `namphuongso` |
| 7 | Repo ngoài `~/work` | `git commit` fail vì `useConfigOnly` (cố ý) |
| 8 | Xoá `ghm` binary | `git pull/push` vẫn đúng account |
| 9 | `gh auth switch -u namphuongso` | không ảnh hưởng git trong 3 thư mục |
| 10 | Repo Azure DevOps (`super-app`) | không bị ảnh hưởng, ngoài scope |

---

## 8. Rủi ro & xử lý

| Rủi ro | Xử lý |
|---|---|
| Org bật SAML SSO | SSH key phải "Authorize" cho org đó (Settings → SSH keys → Configure SSO); `ghm doctor` phát hiện qua lỗi permission |
| 2 account trùng `user.email` → commit bị gán sai account | bắt buộc email noreply riêng mỗi account + `user.useConfigOnly` |
| `insteadOf` rewrite cả submodule / Go module proxy | đúng như mong muốn vì cùng thư mục = cùng account; test với 1 repo có submodule |
| `gh` shim gây khó debug | pass-through mặc định, chỉ can thiệp khi cwd khớp, tắt bằng `GHM_SHIM=0`, log khi `GHM_DEBUG=1` |
| Keychain hỏi password khi `gh auth token` | `security` unlock 1 lần; hoặc cache token trong process `ghm` (memory, TTL 5') |
| Repo hiện có remote https + credential đã lưu osxkeychain | helper bị bỏ → git fail rõ; `ghm adopt` rewrite; không push sai |
| Di chuyển repo làm hỏng IDE/workspace path | `ghm adopt --move` in danh sách path cũ→mới để sửa workspace; hỗ trợ `--copy` giữ bản cũ |
| Windows/Linux khác | plan này macOS-first (keychain); trừu tượng hoá `secretstore` nếu cần port |

---

## 9. Ngoài scope (ghi nhận)

- `super-app` trên Azure DevOps: giữ nguyên, không đụng.
- Git LFS: không ảnh hưởng (đi qua cùng remote SSH).
- Nhiều GitHub Enterprise host: state file đã có `host_alias`, có thể mở rộng `--hostname`.
- GPG commit signing: có thể thêm Phase 4 (`user.signingkey` per account).

---

## 10. Trạng thái triển khai (2026-11-25)

Phase 0 đã chạy thật trên máy.

| Hạng mục | Trạng thái |
|---|---|
| 3 SSH key ed25519 (`~/.ssh/id_ed25519_<login>`) | tạo xong, đã add vào đúng account |
| `ssh -T git@github-<login>` | `Hi anhthaodev!`, `Hi thaoanhhaa1!`, `Hi namphuongso!` |
| `~/.ssh/config` | 3 Host block (backup file cũ nếu có) |
| `~/.config/ghm/git/<login>.gitconfig` | 3 file: user.*, `core.sshCommand`, `url.insteadOf` |
| `~/.gitconfig` | 3 `includeIf gitdir:~/work/<login>/`; backup tại `~/.gitconfig.bak-20261125075341` |
| `credential.https://github.com.helper` | reset về rỗng (git không còn dùng token active cho https) |
| `~/.config/ghm/accounts.yaml` | 3 account, email noreply lấy từ API |
| 7 repo | rewrite remote sang `git@github-<login>:owner/repo.git`, đã move vào `~/work/<login>/` |
| Verify | `fetch` + `push --dry-run` OK cả 7 repo |

Bố cục hiện tại:

```
~/work/anhthaodev/{AI-Gate, it-tools-be, misa}
~/work/thaoanhhaa1/{FoodShare, ai-switcher, it-tool, moni-split-bill}
~/work/namphuongso/          (trống, chờ repo)
```

Ghi chú triển khai:
- Key **không passphrase** (máy cá nhân, không prompt khi script/IDE chạy). Muốn thêm passphrase: `ssh-keygen -p -f ~/.ssh/id_ed25519_<login>` + `AddKeysToAgent yes`/`UseKeychain yes`.
- Scope token đã thêm `admin:public_key` cho cả 3 account (để `gh ssh-key add` hoạt động), scope cũ giữ nguyên.
- `gh auth switch` giờ **không còn ảnh hưởng git**; chỉ đổi account cho lệnh `gh` API. Phần này xử lý ở Phase 2.

### 10.1 Mở rộng: toàn bộ repo GitHub trên máy (26 repo ngoài `~/work`)

Sau Phase 0 phát hiện máy còn 26 repo GitHub dùng `https://github.com` nằm rải trong `Documents/IT`. Đã xử lý thêm:

- Resolve account theo `permissions.push` thật từ API (không đoán theo owner), cache tại `~/.config/ghm/repo-accounts.tsv`.
- Rewrite toàn bộ remote sang `git@github-<login>:owner/repo.git`; ưu tiên account trùng owner khi nhiều account có quyền.
- Ghi `user.name`, `user.email` (noreply), `core.sshCommand` vào **local config** từng repo (vì nằm ngoài `~/work` nên không có `includeIf`).
- Verify: 26/26 `git ls-remote` OK; 0 remote `https://github.com` còn lại trong `~/Documents`, `~/work`.

| Kết quả resolve | Account |
|---|---|
| repo thuộc `thaoanhhaa1` | `thaoanhhaa1` |
| repo thuộc `anhthaodev` | `anhthaodev` |
| `namphuongso/np-hub`, `namphuongso/ACS-Chat-Flutter`, `Nam-Phuong-Digital/*` | `namphuongso` |
| repo chỉ đọc (makara-dev369, JunSuzuki1973, MyLibrary, pratise_android_app) | key của account resolve được, `push` sẽ fail rõ ràng vì không có quyền |
| `hieutt0910/practise_ios_app` | `thaoanhhaa1` (có quyền push thật) |

Repo Azure DevOps (`dev.azure.com`) **không đổi**, vẫn https + credential osxkeychain.

### 10.2 Sự cố `askpass.sh` (Antigravity IDE)

Triệu chứng:

```
fatal: cannot exec '~/.antigravity-ide/extensions/neophack.git-graph-rs-.../out/askpass/askpass.sh'
fatal: could not read Username for 'https://github.com'
```

Nguyên nhân: extension `neophack.git-graph-rs-1.0.28` cài thiếu file (`out/askpass/` rỗng) nhưng vẫn export `GIT_ASKPASS` trỏ vào đó; cửa sổ IDE đang giữ env cũ. Git chỉ gọi askpass khi remote https cần username → đã hết vì 100% remote GitHub chuyển SSH.

Cách xử lý: reload window (`Cmd+Shift+P` → *Developer: Reload Window*) hoặc thoát hẳn Antigravity; nếu vẫn lỗi thì uninstall → install lại extension `Git Graph (Rust)`, hoặc chạy IDE từ terminal sạch (`env -u GIT_ASKPASS`).


## 11. Phase 1 + 2 — đã hoàn thành (2026-11-25)

Binary `~/.local/bin/ghm` (v0.2.0), shim `~/.local/bin/gh`, source tại thư mục dự án này.

| Lệnh | Trạng thái |
|---|---|
| `init`, `list`, `status`, `doctor --fix` | xong |
| `repos`, `sync`, `clone`, `adopt --move`, `exec`, `use`, `token` | xong |
| `install-shim` / `uninstall-shim`, `shim-token` | xong |

Kiểm chứng trên máy:

- `ghm doctor` → **23 ok, 0 warn, 0 fail**.
- `ghm list` → 34 repo GitHub nhận đúng account (`anhthaodev` 14, `thaoanhhaa1` 15, `namphuongso` 5).
- Shim theo thư mục: `misa` → `anhthaodev`, `it-tool` → `thaoanhhaa1`, `np-hub` → `namphuongso`, `/tmp` → account active.
- `ghm exec thaoanhhaa1 -- gh api user` → `thaoanhhaa1`; `GH_TOKEN=... gh ...` thắng shim; `GHM_SHIM=0` tắt shim.
- `go test ./...` pass (8 test: parse remote, includeIf, render config, positional args, repo root), `go vet` sạch.

Quyết định kỹ thuật đã chốt:

- Token chỉ lấy qua `gh auth token --user`, memoize trong RAM, **không ghi token xuống đĩa**.
- Resolve account bằng `permissions.push` thật, ưu tiên account trùng owner, cache ở `~/.config/ghm/repo-accounts.tsv`.
- Shim bỏ qua `auth/config/extension/alias/help/version`, tôn trọng `GH_TOKEN` có sẵn, tắt bằng `GHM_SHIM=0`.
- Scan root mặc định `~/work` + `~/Documents` + `~/Desktop`, depth 6, bỏ `node_modules/.git/Library/vendor/...`.
- `doctor --fix` idempotent: sửa key, ssh config, gitconfig, includeIf, credential helper, https remote, identity, shim.

Lệch so với plan ban đầu:

- Phase 3 (TUI) chưa làm — hiện CLI đã đủ dùng.
- `ghm init` yêu cầu account đã có trong keyring (không tự chạy `gh auth login`), đúng thiết kế "không login lại".
- Check `repo-identity` chỉ cảnh báo khi local `user.email` lệch, không tự sửa trừ khi `--fix`.

## 12. Phase 3 + 4 — đã hoàn thành (2026-11-25)

`ghm` v0.3.0.

### Phase 3 — TUI

`ghm tui` (bubbletea + lipgloss, tab account, bảng repo):

- `j/k`, `g/G` di chuyển; `1..9`/`tab`/`←→` đổi account; `/` filter (tên, branch, path).
- `f` fetch --prune, `p` pull --ff-only, `P` push, `o` mở github.com, `r` rescan, `?` help, `q` quit.
- Trạng thái lỗi hiển thị đỏ, reload tự động sau mỗi action.
- Test `TestTUIViewRenders` render model ở 120x30, kiểm tra filter + tab account + không lẫn repo account khác (vì TUI cần TTY nên test ở tầng View).

### Phase 4 — đóng gói

| File | Việc |
|---|---|
| `internal/app/completion.go` | `ghm completion zsh\|bash\|fish` |
| `.goreleaser.yaml` | darwin/linux amd64+arm64, checksums, Homebrew formula (`thaoanhhaa1/homebrew-tap`) |
| `.github/workflows/ci.yml` | gofmt, vet, test, build, `ghm version`, completion; Ubuntu + macOS; validate YAML |
| `.github/workflows/release.yml` | tag `v*` → goreleaser release |
| `LICENSE` | MIT |
| `Makefile` | `build`, `install`, `test`, `vet`, `fmt`, `doctor` |

Cài đặt:

```bash
go install github.com/thaoanhhaa1/ghm/cmd/ghm@latest   # khi repo public
brew install thaoanhhaa1/tap/ghm                       # sau lần release đầu
```

### Còn tuỳ chọn

- Commit/push có prompt message trong TUI.
- GPG signing theo account, port Windows.
- `ghm doctor` trong pre-push hook hoặc CI của từng repo.

## 13. TUI PR + log — v0.4.0 (2026-11-25)

Thêm hai phím vào dashboard:

| Phím | Việc |
|---|---|
| `c` | `git push -u origin HEAD` bằng key của repo, rồi `gh pr create --title <commit subject> --body ""` bằng token account đó (`GH_PROMPT_DISABLED=1`, `GHM_SHIM=0`). URL in ở status; lỗi (repo read-only, detached HEAD, `main` → `main`) hiện đỏ và không thoát TUI. |
| `l` | Overlay log 60 commit gần nhất (`%h %ad %an %s`); `j/k` scroll, `g/G` đầu/cuối, `PgUp/PgDn`, `esc` đóng. |

Chi tiết triển khai:

- `c`/`l` tách khỏi phím đổi account: bỏ `h`/`l` khỏi nhóm `tab`/`←→` (giờ chỉ còn `tab`/`shift+tab`/`←→`).
- Status "sticky": message của action (`PR created: <url>`, `pushed`, lỗi) không bị rescan ghi đè bởi "N repositories".
- `prCmd` gọi `r.GhPath()` (gh thật, không qua shim) + `GH_TOKEN` của account; `cmd.Dir` = repo.
- Test mới: `TestLogOverlayRenders`, `TestLastURL`, `TestPRCmdRequiresRemote`, `TestPRCmdPushesAndCreatesPR` (repo + bare remote + stub `gh`, không cần mạng), `TestStickyStatusSurvivesReload`.
- Verify thật trong PTY: mở log, đóng log, `c` trên repo read-only → hiện `PR ...: push: ... ERROR: Write access to repository not granted` đúng account.

## 14. Thêm account bằng UI — v0.5.0 (2026-11-25)

Trước đây thêm account phải chạy 5–6 lệnh tay. Giờ có 2 đường vào, dùng chung một hàm `AddAccount`:

### TUI: phím `a`

```
add github account
  b   login with browser   (gh auth login, recommended)
  t   paste a token        (scopes: repo, read:org, admin:public_key)
  esc cancel
```

- `b`: TUI tạm nhường terminal (`tea.ExecProcess`) cho `gh auth login -h github.com -p https -w`; khi quay lại, ghm đọc keyring, diff với `accounts.yaml` để tìm login mới rồi cấu hình.
- `t`: nhập token (mask `•`, không echo, không ghi đĩa), pipe vào `gh auth login --with-token --hostname github.com`, rồi cấu hình.
- Màn kết quả liệt kê từng bước (`account:`, `key:`, `ssh:`, `git:`, `dir:`, `next:`) + `warn:` nếu token thiếu `admin:public_key`.
- Account mới tự thành tab hiện tại; `esc` đóng, `r` rescan.

### CLI: `ghm add <login>`

Dùng khi login đã có trong `gh` (script/CI): chạy đúng flow trên, in cùng định dạng dòng.

### Các bước `AddAccount` thực hiện

1. `gh api user` lấy name/email (fallback `<id>+<login>@users.noreply.github.com`).
2. Append vào `accounts.yaml`, `applyDefaults`, `Save`.
3. `EnsureSSHConfig` (Host `github-<login>`), `ResetCredentialHelper` (bỏ helper https nếu còn).
4. `EnsureKey` (`~/.ssh/id_ed25519_<login>`), `KeyOnAccount` → `AddKeyToAccount` nếu chưa có.
5. `EnsureAccountGitConfig` (`~/.config/ghm/git/<login>.gitconfig`) + `EnsureIncludeIf` (`~/work/<login>/`, tạo luôn thư mục).
6. In `next: ghm doctor`.

### Test

- `TestNewLogins` (diff keyring, case-insensitive), `TestAddAccountRejectsDuplicate`.
- `TestAddWizardMenuAndToken` (mở `a`, chọn `t`, token mask, esc đóng), `TestAddBrowserStartsLogin` (trả `ExecProcess`, state busy), `TestAddDoneSwitchesToNewAccount` (tab mới + status + màn kết quả).
- PTY thật: `a` → menu → `t` → gõ `ghp_secret` → chỉ thấy `••••`, esc về bảng, `q` thoát 0.
- CLI: `ghm add` (exit 2), `ghm add anhthaodev` → "already configured" (exit 1), `ghm add octocat` → "not logged in" (exit 1); `ghm doctor` vẫn 23/23.

### Giới hạn

- `b` cần browser + quyền tương tác; trong SSH/tmux không có browser thì dùng `t`.
- Token dán vào cần scope `admin:public_key` để tự đăng ký key; thiếu thì account vẫn được thêm, key phải thêm tay (UI in lệnh gợi ý).
- Chưa hỗ trợ GitHub Enterprise host khác `github.com`.

## 15. App macOS native (SwiftUI) — v0.6.0 (2026-11-25)

Yêu cầu: UI thật (cửa sổ, chuột, form), không phải TUI; thêm account bằng cả device flow lẫn PAT.

### Kiến trúc

- `app/` là Swift package (`GhmApp`, macOS 13+), không cần Xcode project.
- App **không** đọc/ghi key, token hay git config. Mọi thứ đi qua `ghm api <action>` (JSON) trong `internal/app/api.go`, nên logic vẫn nằm ở Go và test được.
- `app/scripts/bundle.sh` (gọi bằng `make app`): build release, copy `ghm` vào `Contents/MacOS/ghm`, viết `Info.plist`, ký ad-hoc. Bundle chứa backend nên mở từ Finder vẫn chạy dù GUI không có `~/.local/bin` trong `PATH`.
- `GhmCLI` tìm binary theo thứ tự: trong bundle → `~/.local/bin/ghm` → `PATH` → Homebrew. Khi chạy con, nó xoá `GH_TOKEN`/`GHM_SHIM` để token thừa không đè account theo folder.

### API mới

| Action | Việc |
|---|---|
| `state` | accounts + repos + active login + root (một lần gọi cho toàn bộ dashboard) |
| `doctor [--fix]` | danh sách check kèm level/detail, có sửa |
| `add <login>` | `AddAccount` cho login đã có trong keyring (dùng chung với CLI/TUI) |
| `login-token` | đọc token từ stdin → `gh auth login --with-token` → diff keyring → `AddAccount` |
| `device-start [--client-id]` | gọi `POST /login/device/code` |
| `device-poll --device-code` | poll `POST /login/oauth/access_token`, trả `pending/slow_down/success/error` |
| `git --path --action fetch\|pull\|push` | chạy git trong repo đó |

Mọi action in **một** object JSON và exit 0, để UI hiển thị được cả trường hợp một phần thất bại (warn, thiếu scope) thay vì đoán qua exit code.

### UI

- `NavigationSplitView`: sidebar (All repositories + từng account + Manage) và bảng repo (`Table`) với account/branch/ahead/behind/state, context menu fetch/pull/push, mở GitHub, Finder, copy path, `⌘R`.
- `DoctorView`: chạy `ghm doctor`, hiện ok/warn/fail, nút *Fix issues*.
- `AddAccountView`: segmented Browser/Token.
  - Browser: `device-start` → hiện `user_code` (copy được), mở browser, poll mỗi `interval` giây; `slow_down` thì tăng 5s; hết hạn thì báo chạy lại.
  - Token: `SecureField`, link tạo token với scope điền sẵn, submit qua stdin.
  - Kết quả: danh sách dòng (`account:`, `key:`, `ssh:`, `git:`, `dir:`, `next:`) + warn màu cam + lỗi màu đỏ; *Add another* / *Done*.
- Sau khi thêm: `model.refresh(silently:)`, account mới xuất hiện ở sidebar.
- `NSDocumentsFolderUsageDescription` / `NSDesktopFolderUsageDescription` để prompt TCC có lý do rõ ràng (ghm mặc định scan `~/work`, `~/Documents`, `~/Desktop`).

### Test

- Go: `TestAPIDeviceFlowAddsAccount` dựng `httptest` giả GitHub (device/code, access_token pending→success, `/user`, `/user/keys` GET/POST) + stub `gh`, `HOME` tạm; kiểm tra poll pending→success, account `beta` được lưu, key sinh ra, `Host github-beta`, gitconfig, folder, và add lần 2 bị từ chối. `TestAPIDeviceExpired`, `TestAPIGitActionValidatesInput`.
- `GHM_GITHUB_API` / `GHM_GITHUB_WEB` cho phép trỏ REST/OAuth sang server test (hoặc GitHub Enterprise), mặc định vẫn là github.com.
- E2E thật trong app (bundle `app/build/ghm.app`, `HOME` tạm, server giả + stub `gh`):
  - device flow: mở *Add account* → *Start browser login* → hiện `E2E1-2345` + *Waiting for approval* → result *Account ready* với đủ dòng; `accounts.yaml`, `~/.ssh/config`, key, gitconfig đều có `beta`.
  - token flow: chọn *Token*, gõ token (AX trả `••••…`, không lộ), submit → *Account ready*, thêm `gamma`.
  - App chạy với dữ liệu thật: sidebar 3 account (14/16/5 repo), bảng 88 repo, `gh active: anhthaodev`, doctor 23 ok.

### Giới hạn / ghi chú

- Device flow mặc định dùng client id public của GitHub CLI (`Iv1.b507a08c87ecfe98`) để chạy ngay. Khi phát hành cho người khác nên tạo OAuth App riêng (bật Device Flow) và nhập client id; PAT không cần OAuth app.
- App ký ad-hoc, chưa notarize: máy khác tải về sẽ bị Gatekeeper hỏi. Muốn phát hành cần Developer ID + notarization.
- Lần đầu mở, macOS hỏi quyền Documents/Desktop vì scan repo; từ chối thì chỉ thấy repo trong `~/work`.
- Chưa có icon `.icns`, chưa có auto-update, chưa hỗ trợ GitHub Enterprise UI (backend đã có env URL).

## 16. Icon, Liquid Glass, auto-update — v0.7.0 (2026-11-25)

Ba việc: icon `.icns`, giao diện Liquid Glass, và Sparkle auto-update.

### Icon

- `app/scripts/make-icon.swift` vẽ icon 1024×1024 bằng CoreGraphics: squircle bo 220, gradient indigo→violet→teal, 3 aurora blob, specular sweep, vignette, rồi glyph git-branch 4 node (kim loại glass) + terminal caret.
- `make icon` render ra `app/build/AppIcon.iconset`, `iconutil -c icns` → `app/Resources/AppIcon.icns` (16…512@2x).
- `Info.plist` có `CFBundleIconFile=AppIcon`; bundle copy vào `Contents/Resources`. Corner alpha = 0 nên không có nền vuông.

### Liquid Glass

- Nâng deployment target lên **macOS 26** (`.macOS("26.0")` trong `Package.swift`) để dùng API thật thay vì fake blur.
- `Theme.swift`:
  - `LiquidBackground`: gradient tối + 3 blob màu lớn (indigo/violet/teal) drift chậm 22s, làm nền để glass khúc xạ.
  - `glassPanel(_:tint:interactive:)`: wrapper quanh `.glassEffect(.regular[.tint().interactive()], in: .rect(cornerRadius:))`.
  - `GlassBadge`, `GlassIconButton` (`.buttonStyle(.glass)`), `StatePill`, `Avatar` gradient theo login.
- `RootView` bỏ `NavigationSplitView` + `Table`, thay bằng layout riêng:
  - Sidebar là một glass panel 272pt: logo gradient, hàng "All repositories", từng account với avatar + badge key/token + số repo, selection dùng `.glassEffect(.regular.tint(...).interactive())` + `glassEffectID` để morph, Manage (Add account `.glassProminent`, Health/Refresh/Check updates `.glass`), footer `gh active` + root.
  - Detail là glass panel: header (title, dirty/behind badge, search trong glass capsule, refresh/health/add), danh sách row tự vẽ (hover/selected glass highlight, click chọn, double-click mở GitHub, context menu), status bar glass.
  - Cửa sổ `.windowStyle(.hiddenTitleBar)`, chừa 30pt cho traffic lights.
- Sheet Add account/Doctor: nền gradient riêng, icon trong glass circle, code box/result panel/kết quả doctor đều `glassPanel`, button glass.
- Bỏ badge "0.7.0" hardcode: đọc `CFBundleShortVersionString`.

### Auto-update (Sparkle 2.10.1)

- `Package.swift` thêm `.package(url: "https://github.com/sparkle-project/Sparkle", from: "2.6.0")` (resolve 2.10.1, commit `Package.resolved`).
- `Updater.swift`: `SPUStandardUpdaterController(startingUpdater: true)` + delegate cho `GHM_UPDATE_FEED` / `GHM_UPDATE_CHANNEL` (test/staging, không cần rebuild).
- Menu app có **Check for Updates…** (`⇧⌘U`), sidebar có nút tương ứng.
- `Info.plist`: `SUFeedURL=https://github.com/thaoanhhaa1/ghm/releases/latest/download/appcast.xml`, `SUPublicEDKey` (Ed25519 sinh bằng `generate_keys`, private key trong login keychain), `SUEnableAutomaticChecks`, `SUAllowsAutomaticUpdates`, `SUAutomaticallyUpdate`, `SUScheduledCheckInterval=86400`.
- `bundle.sh`: copy `Sparkle.framework` vào `Contents/Frameworks`, thêm rpath `@executable_path/../Frameworks`, ký framework rồi app, `codesign --verify --deep --strict`.
- Release tay: `GHM_VERSION=x.y.z GHM_MAKE_RELEASE=1 ./app/scripts/bundle.sh release` → `dist/ghm-x.y.z.zip` + `dist/appcast.xml` (EdDSA signature, URL trỏ `releases/download/vX/`). Upload cả hai lên GitHub Release; `latest/download/appcast.xml` là feed.

### Kiểm chứng

- App 0.7.0 chạy thật: sidebar 3 account (14/16/5 repo), 88 repo, `gh active: anhthaodev`; `ghm doctor` 23 ok.
- Icon: `iconutil` OK, `sips` 1024, ASCII render đúng squircle + glyph, 4 góc alpha 0.
- Update e2e: build 0.7.1 + zip + `generate_appcast` (prefix `http://127.0.0.1:8123/`), chạy app 0.7.0 với `GHM_UPDATE_FEED=…`, bấm Check for Updates → Sparkle hiện *“A new version of ghm is available! ghm 0.7.1 … you have 0.7.0”*; bấm Install → *“Updating ghm / Ready to Install”* → app tự thay bundle, chạy lại **0.7.1**.
- `codesign --verify --deep --strict` pass; `otool -l` có rpath `@executable_path/../Frameworks`; AX thấy menu `Check for Updates…` và nút sidebar.

### Giới hạn

- App ký ad-hoc + chưa notarize; auto-update chạy được trên máy này nhưng phát hành cho máy khác cần Developer ID ổn định + notarization.
- Icon hiện là file `.icns` cổ điển; macOS 26 còn hỗ trợ icon layered (Default/Dark/Clear/Tinted) qua `.icon` bundle — chưa làm.
- CI/release GitHub Actions vẫn fail 3s/0 step nên chưa có release tự động; quy trình release đang thủ công.

## 17. Giao diện trầm hơn — v0.7.1 (2026-11-25)

Bản 0.7.0 bị chê "màu mè". Giữ Liquid Glass nhưng bỏ trang trí:

- `LiquidBackground`: nền graphite `#0B0D11 → #11141A`, chỉ còn 2 quầng xám-xanh rất nhạt (opacity 0.20/0.12, blur ~190–200) thay cho 3 blob indigo/violet/teal đậm; thêm vignette. Drift chậm 30s.
- Bảng màu mới: `ghmAccent` (steel `#9DB3CC`), `ghmPositive` (sage), `ghmWarning` (amber), `ghmInfo` (steel), `ghmDanger` (dusty red). Bỏ `ghmIndigo/Violet/Teal/Mint/Sky`.
- Logo "ghm" màu trắng, bỏ gradient cầu vồng; selection sidebar tint `ghmAccent` 0.16 thay vì sky/violet 0.35; avatar dùng 4 tông đá trầm theo login; badge dirty/behind dùng amber/steel; bỏ shadow phát sáng ở chấm trạng thái.
- Nút prominent không còn `.tint(.ghmIndigo)` — dùng accent hệ thống.
- Sheet Add account/Doctor: nền `#0E1116 → #12161D`, icon và các card glass không tint.
- Đo lại: saturation trung bình của cửa sổ 0.55 → 0.18; panel rgb(49,54,63) trên nền rgb(18,22,29) nên vẫn tách lớp rõ.
