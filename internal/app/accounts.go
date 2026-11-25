package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddAccount registers a login that is already authenticated in gh.
// It appends the account to the config, creates and registers its SSH key,
// writes the per-account gitconfig and includeIf, and creates its folder.
//
// The returned lines are meant to be shown to the user. A missing
// admin:public_key scope is reported as a warning: the account is still added,
// the key has to be registered manually or with `gh auth refresh`.
func AddAccount(r *Runner, login string) ([]string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, fmt.Errorf("empty login")
	}
	if _, ok := r.Cfg.Account(login); ok {
		return nil, fmt.Errorf("%s is already configured", login)
	}
	if _, err := r.Token(login); err != nil {
		return nil, fmt.Errorf("%s is not logged in to gh; run: gh auth login -h github.com (or press a in ghm tui)", login)
	}
	u, err := r.UserInfo(login)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(u.Name)
	if name == "" {
		name = login
	}
	email := strings.TrimSpace(u.Email)
	if email == "" {
		email = fmt.Sprintf("%d+%s@users.noreply.github.com", u.ID, login)
	}

	a := Account{
		Login:     login,
		Name:      name,
		Email:     email,
		Key:       "~/.ssh/id_ed25519_" + login,
		HostAlias: "github-" + login,
		Dir:       filepath.Join(r.Cfg.Root, login),
	}

	r.Cfg.Accounts = append(r.Cfg.Accounts, a)
	r.Cfg.applyDefaults()
	if _, ok := r.Cfg.Account(r.Cfg.DefaultAccount); !ok {
		r.Cfg.DefaultAccount = r.Cfg.Accounts[0].Login
	}
	if err := r.Cfg.Save(); err != nil {
		return nil, err
	}

	lines := []string{fmt.Sprintf("account: %s <%s>", login, email)}

	added, err := EnsureSSHConfig(r.Cfg.Accounts)
	if err != nil {
		return lines, err
	}
	for _, h := range added {
		lines = append(lines, "ssh: added Host "+h)
	}
	if changed, err := ResetCredentialHelper(); err != nil {
		return lines, err
	} else if changed {
		lines = append(lines, "git: disabled gh https credential helper")
	}
	if created, err := EnsureKey(a); err != nil {
		return lines, err
	} else if created {
		lines = append(lines, "key: generated "+a.Key)
	} else {
		lines = append(lines, "key: reusing "+a.Key)
	}

	has, err := r.KeyOnAccount(login, a.Key+".pub")
	switch {
	case err != nil:
		lines = append(lines, "warn: "+firstLine(err.Error()))
	case has:
		lines = append(lines, "key: already registered on github.com")
	default:
		if err := r.AddKeyToAccount(login, a.Key+".pub"); err != nil {
			lines = append(lines, "warn: "+firstLine(err.Error()))
			lines = append(lines, "warn: add it manually: cat "+TildePath(ExpandPath(a.Key))+".pub")
		} else {
			lines = append(lines, "key: registered on github.com")
		}
	}

	if changed, err := EnsureAccountGitConfig(a); err != nil {
		return lines, err
	} else if changed {
		lines = append(lines, "git: wrote "+TildePath(AccountGitConfigPath(login)))
	}
	if changed, err := EnsureIncludeIf(a); err != nil {
		return lines, err
	} else if changed {
		lines = append(lines, "git: includeIf "+TildePath(a.Dir)+"/")
	}

	lines = append(lines, "dir: "+TildePath(a.Dir))
	lines = append(lines, "next: ghm doctor")
	return lines, nil
}

// newLogins returns entries in after that are not in before (case-insensitive).
func newLogins(before, after []string) []string {
	seen := map[string]bool{}
	for _, l := range before {
		seen[strings.ToLower(l)] = true
	}
	var out []string
	for _, l := range after {
		l = strings.TrimSpace(l)
		if l == "" || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		out = append(out, l)
	}
	return out
}

// cmdAdd configures an account that is already authenticated in gh.
func cmdAdd(r *Runner, args []string) int {
	pos := positional(args)
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ghm add <login>   (the login must exist in gh; use ghm tui and press a to log in)")
		return 2
	}
	lines, err := AddAccount(r, pos[0])
	for _, l := range lines {
		fmt.Println(l)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm add: %v\n", err)
		return 1
	}
	return 0
}
