package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SSHDir is ~/.ssh.
func SSHDir() string { return filepath.Join(Home(), ".ssh") }

// SSHConfigPath is ~/.ssh/config.
func SSHConfigPath() string { return filepath.Join(SSHDir(), "config") }

// ZshrcPath is ~/.zshrc.
func ZshrcPath() string { return filepath.Join(Home(), ".zshrc") }

// LocalBinDir is ~/.local/bin.
func LocalBinDir() string { return filepath.Join(Home(), ".local", "bin") }

// ShimPath is the managed gh shim.
func ShimPath() string { return filepath.Join(LocalBinDir(), "gh") }

// GhmInstallPath is where the ghm binary lives for the shim to call.
func GhmInstallPath() string { return filepath.Join(LocalBinDir(), "ghm") }

func backupFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", nil
	}
	dst := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102150405"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(dst, raw, 0o600); err != nil {
		return "", err
	}
	return dst, nil
}

// EnsureSSHConfig adds a Host block per account. It returns changed aliases.
func EnsureSSHConfig(accounts []Account) ([]string, error) {
	if err := os.MkdirAll(SSHDir(), 0o700); err != nil {
		return nil, err
	}
	path := SSHConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	content := string(raw)
	hosts := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			for _, h := range strings.Fields(trimmed)[1:] {
				hosts[h] = true
			}
		}
	}
	var added []string
	for _, a := range accounts {
		if hosts[a.HostAlias] {
			continue
		}
		block := fmt.Sprintf("\nHost %s\n    HostName %s\n    User git\n    IdentityFile %s\n    IdentitiesOnly yes\n",
			a.HostAlias, githubHost, a.Key)
		content = strings.TrimRight(content, "\n") + "\n" + block
		added = append(added, a.HostAlias)
	}
	if len(added) == 0 {
		return nil, nil
	}
	if _, err = backupFile(path); err != nil {
		return nil, err
	}
	if err = os.WriteFile(path, []byte(content), 0o600); err != nil {
		return nil, err
	}
	return added, nil
}

// AccountGitConfig renders the per-account gitconfig.
func AccountGitConfig(a Account) string {
	return fmt.Sprintf(`[user]
	name = %s
	email = %s
[core]
	sshCommand = %s
[url "git@%s:"]
	insteadOf = https://%s/
`, a.Name, a.Email, a.SSHCommandScript(), a.HostAlias, githubHost)
}

// EnsureAccountGitConfig writes the per-account gitconfig when missing/different.
func EnsureAccountGitConfig(a Account) (bool, error) {
	path := AccountGitConfigPath(a.Login)
	want := AccountGitConfig(a)
	if raw, err := os.ReadFile(path); err == nil && string(raw) == want {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// EnsureIncludeIf registers includeIf gitdir blocks globally.
func EnsureIncludeIf(a Account) (bool, error) {
	dir := a.RepoRoot()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	key := "includeIf.gitdir:" + dir + string(os.PathSeparator) + ".path"
	out, err := Git("", "config", "--global", "--get-all", key)
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) == AccountGitConfigPath(a.Login) {
				return false, nil
			}
		}
	}
	if _, err := Git("", "config", "--global", "--add", key, AccountGitConfigPath(a.Login)); err != nil {
		return false, err
	}
	return true, nil
}

// ResetCredentialHelper stops git from silently using gh's active-account token.
func ResetCredentialHelper() (bool, error) {
	changed := false
	for _, host := range []string{githubHost, "gist." + githubHost} {
		key := "credential.https://" + host + ".helper"
		out, err := Git("", "config", "--global", "--get-all", key)
		if err == nil && out == "" {
			continue
		}
		if _, err = Git("", "config", "--global", "--replace-all", key, ""); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

// KeyExists reports whether the private key file exists.
func KeyExists(a Account) bool {
	_, err := os.Stat(ExpandPath(a.Key))
	return err == nil
}

// EnsureKey generates the ed25519 keypair if missing.
func EnsureKey(a Account) (bool, error) {
	if KeyExists(a) {
		_ = os.Chmod(ExpandPath(a.Key), 0o600)
		return false, nil
	}
	if err := os.MkdirAll(SSHDir(), 0o700); err != nil {
		return false, err
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "macbook-"+a.Login,
		"-f", ExpandPath(a.Key), "-N", "", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("ssh-keygen: %v %s", err, strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(ExpandPath(a.Key), 0o600)
	return true, nil
}

// SSHConfigHasAlias reports whether the alias is present.
func SSHConfigHasAlias(alias string) bool {
	raw, err := os.ReadFile(SSHConfigPath())
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			for _, h := range strings.Fields(trimmed)[1:] {
				if h == alias {
					return true
				}
			}
		}
	}
	return false
}

// IncludeIfRegistered reports whether the includeIf entry exists.
func IncludeIfRegistered(a Account) bool {
	key := "includeIf.gitdir:" + a.RepoRoot() + string(os.PathSeparator) + ".path"
	out, err := Git("", "config", "--global", "--get-all", key)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == AccountGitConfigPath(a.Login) {
			return true
		}
	}
	return false
}

// CredentialHelperReset reports whether gh's https helper is disabled.
func CredentialHelperReset() bool {
	out, err := Git("", "config", "--global", "--get-all", "credential.https://"+githubHost+".helper")
	return err == nil && out == ""
}

// ShimInstalled reports whether ~/.local/bin/gh is the ghm shim.
func ShimInstalled() bool {
	raw, err := os.ReadFile(ShimPath())
	return err == nil && strings.Contains(string(raw), "managed by ghm")
}

// PathHasLocalBin reports whether ~/.local/bin is on PATH for the current process.
func PathHasLocalBin() bool {
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if filepath.Clean(p) == filepath.Clean(LocalBinDir()) {
			return true
		}
	}
	return false
}

// ZshrcHasLocalBin reports whether ~/.local/bin is exported in ~/.zshrc.
func ZshrcHasLocalBin() bool {
	raw, err := os.ReadFile(ZshrcPath())
	return err == nil && strings.Contains(string(raw), ".local/bin")
}

const shimMarkerStart = "# >>> ghm shim >>>"
const shimMarkerEnd = "# <<< ghm shim <<<"

// InstallShim copies the ghm binary and writes the gh wrapper.
func InstallShim(r *Runner) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(LocalBinDir(), 0o755); err != nil {
		return "", err
	}
	if filepath.Clean(self) != filepath.Clean(GhmInstallPath()) {
		raw, rerr := os.ReadFile(self)
		if rerr != nil {
			return "", rerr
		}
		if werr := os.WriteFile(GhmInstallPath(), raw, 0o755); werr != nil {
			return "", werr
		}
	}
	_ = os.Chmod(GhmInstallPath(), 0o755)

	real := r.GhPath()
	if filepath.Clean(real) == filepath.Clean(ShimPath()) {
		return "", fmt.Errorf("refusing to wrap the shim itself (%s); reinstall gh or set gh_binary in accounts.yaml", real)
	}
	shim := fmt.Sprintf(`#!/bin/sh
# managed by ghm - injects the per-folder GitHub account.
REAL="%s"
GHM="%s"
case "${1:-}" in
  auth|config|extension|alias|help|--help|-h|--version|-v) exec "$REAL" "$@" ;;
esac
if [ -z "${GH_TOKEN:-}" ] && [ "${GHM_SHIM:-1}" != "0" ] && [ -x "$GHM" ]; then
  tok=$("$GHM" shim-token 2>/dev/null)
  if [ -n "$tok" ]; then GH_TOKEN="$tok" exec "$REAL" "$@" ; fi
fi
exec "$REAL" "$@"
`, real, GhmInstallPath())
	if err := os.WriteFile(ShimPath(), []byte(shim), 0o755); err != nil {
		return "", err
	}

	pathChanged := false
	if !ZshrcHasLocalBin() {
		if _, err := backupFile(ZshrcPath()); err != nil {
			return "", err
		}
		f, ferr := os.OpenFile(ZshrcPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if ferr != nil {
			return "", ferr
		}
		_, _ = fmt.Fprintf(f, "\n%s\nexport PATH=\"$HOME/.local/bin:$PATH\"\n%s\n", shimMarkerStart, shimMarkerEnd)
		_ = f.Close()
		pathChanged = true
	}
	msg := fmt.Sprintf("shim: %s -> %s", ShimPath(), real)
	if pathChanged {
		msg += "\nPATH: added ~/.local/bin to ~/.zshrc (open a new terminal)"
	}
	return msg, nil
}

// UninstallShim removes the wrapper and the PATH block.
func UninstallShim() error {
	if _, err := os.Stat(ShimPath()); err == nil {
		if err := os.Remove(ShimPath()); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(ZshrcPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(string(raw), "\n")
	var out []string
	skip := false
	for _, l := range lines {
		switch strings.TrimSpace(l) {
		case shimMarkerStart:
			skip = true
			continue
		case shimMarkerEnd:
			skip = false
			continue
		}
		if !skip {
			out = append(out, l)
		}
	}
	_ = os.WriteFile(ZshrcPath(), []byte(strings.Join(out, "\n")), 0o644)
	return nil
}
