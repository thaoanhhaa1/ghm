package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type check struct {
	name string
	run  func() (string, string) // level, detail
	fix  func() (string, error)
}

const (
	levelOK   = "ok"
	levelWarn = "warn"
	levelFail = "fail"
)

func cmdDoctor(r *Runner, args []string) int {
	doFix := hasFlag(args, "--fix")
	repos := scanAll(r.Cfg, false)
	checks := buildChecks(r, repos)

	ok, warn, fail := 0, 0, 0
	for _, c := range checks {
		level, detail := c.run()
		if doFix && level != levelOK && c.fix != nil {
			if msg, err := c.fix(); err != nil {
				detail += fmt.Sprintf("\n         fix failed: %v", err)
			} else {
				level, detail = c.run()
				detail = strings.TrimSpace(msg + " -> " + detail)
			}
		}
		switch level {
		case levelOK:
			ok++
		case levelWarn:
			warn++
		default:
			fail++
		}
		fmt.Printf("[%-4s] %-20s %s\n", level, c.name, detail)
	}
	fmt.Printf("\n%d ok, %d warn, %d fail\n", ok, warn, fail)
	if fail > 0 {
		if !doFix {
			fmt.Println("hint: ghm doctor --fix")
		}
		return 1
	}
	return 0
}

func buildChecks(r *Runner, repos []Repo) []check {
	var checks []check

	for _, acct := range r.Cfg.Accounts {
		a := acct
		checks = append(checks, check{
			name: "key:" + a.Login,
			run: func() (string, string) {
				if !KeyExists(a) {
					return levelFail, "missing " + a.Key
				}
				if _, err := os.Stat(ExpandPath(a.Key) + ".pub"); err != nil {
					return levelWarn, "private key present, public key missing"
				}
				return levelOK, TildePath(ExpandPath(a.Key))
			},
			fix: func() (string, error) {
				created, err := EnsureKey(a)
				if err != nil {
					return "", err
				}
				if created {
					return "generated key", nil
				}
				return "key present", nil
			},
		})
		checks = append(checks, check{
			name: "ssh-config:" + a.Login,
			run: func() (string, string) {
				if !SSHConfigHasAlias(a.HostAlias) {
					return levelFail, "Host " + a.HostAlias + " missing in ~/.ssh/config"
				}
				return levelOK, "Host " + a.HostAlias
			},
			fix: func() (string, error) {
				added, err := EnsureSSHConfig([]Account{a})
				if err != nil {
					return "", err
				}
				if len(added) == 0 {
					return "already present", nil
				}
				return "added Host " + a.HostAlias, nil
			},
		})
		checks = append(checks, check{
			name: "ssh-auth:" + a.Login,
			run: func() (string, string) {
				out, err := r.SSHAuth(a)
				if err != nil {
					return levelFail, firstLine(out)
				}
				return levelOK, "Hi " + a.Login + "!"
			},
			fix: func() (string, error) {
				if _, err := EnsureKey(a); err != nil {
					return "", err
				}
				if _, err := EnsureSSHConfig([]Account{a}); err != nil {
					return "", err
				}
				has, err := r.KeyOnAccount(a.Login, a.Key+".pub")
				if err != nil {
					return "", err
				}
				if !has {
					if err := r.AddKeyToAccount(a.Login, a.Key+".pub"); err != nil {
						return "", err
					}
					return "registered public key", nil
				}
				return "key already registered", nil
			},
		})
		checks = append(checks, check{
			name: "gitconfig:" + a.Login,
			run: func() (string, string) {
				raw, err := os.ReadFile(AccountGitConfigPath(a.Login))
				if err != nil {
					return levelFail, "missing " + TildePath(AccountGitConfigPath(a.Login))
				}
				if !strings.Contains(string(raw), a.Email) || !strings.Contains(string(raw), "insteadOf") {
					return levelWarn, "outdated content"
				}
				return levelOK, a.Email
			},
			fix: func() (string, error) {
				changed, err := EnsureAccountGitConfig(a)
				if err != nil {
					return "", err
				}
				if changed {
					return "rewrote gitconfig", nil
				}
				return "gitconfig current", nil
			},
		})
		checks = append(checks, check{
			name: "include-if:" + a.Login,
			run: func() (string, string) {
				if !IncludeIfRegistered(a) {
					return levelFail, "~/work/" + a.Login + "/ not registered in ~/.gitconfig"
				}
				return levelOK, a.Dir
			},
			fix: func() (string, error) {
				changed, err := EnsureIncludeIf(a)
				if err != nil {
					return "", err
				}
				if changed {
					return "registered includeIf", nil
				}
				return "already registered", nil
			},
		})
		checks = append(checks, check{
			name: "gh-token:" + a.Login,
			run: func() (string, string) {
				if _, err := r.Token(a.Login); err != nil {
					return levelFail, "gh auth token failed (re-run: gh auth login)"
				}
				return levelOK, "keyring token available"
			},
		})
	}

	checks = append(checks, check{
		name: "credential-helper",
		run: func() (string, string) {
			if !CredentialHelperReset() {
				return levelFail, "gh https helper active: git could push with the wrong account"
			}
			return levelOK, "github.com https helper disabled"
		},
		fix: func() (string, error) {
			changed, err := ResetCredentialHelper()
			if err != nil {
				return "", err
			}
			if changed {
				return "disabled gh https helper", nil
			}
			return "already disabled", nil
		},
	})

	httpsRepos := func() []Repo {
		var out []Repo
		for _, rp := range repos {
			if rp.Login == "" && rp.IsGitHub {
				out = append(out, rp)
			}
		}
		return out
	}
	checks = append(checks, check{
		name: "https-remotes",
		run: func() (string, string) {
			bad := httpsRepos()
			if len(bad) == 0 {
				return levelOK, "0 github repos on https"
			}
			var names []string
			for i, rp := range bad {
				if i == 3 {
					names = append(names, fmt.Sprintf("+%d more", len(bad)-3))
					break
				}
				names = append(names, TildePath(rp.Path))
			}
			return levelFail, fmt.Sprintf("%d repos: %s", len(bad), strings.Join(names, ", "))
		},
		fix: func() (string, error) {
			bad := httpsRepos()
			fixed := 0
			for _, rp := range bad {
				if err := adoptOne(r, rp.Path, "", false, false); err == nil {
					fixed++
				}
			}
			if fixed != len(bad) {
				return "", fmt.Errorf("adopted %d/%d", fixed, len(bad))
			}
			return fmt.Sprintf("adopted %d repos", fixed), nil
		},
	})

	mismatched := func() []Repo {
		var out []Repo
		for _, rp := range repos {
			if rp.Login == "" {
				continue
			}
			a, ok := r.Cfg.AccountFromAlias(rp.Login)
			if !ok {
				continue
			}
			if local := LocalEmail(rp.Path); local != "" && local != a.Email {
				out = append(out, rp)
			}
		}
		return out
	}
	checks = append(checks, check{
		name: "repo-identity",
		run: func() (string, string) {
			bad := mismatched()
			if len(bad) == 0 {
				return levelOK, "all repos use their account email"
			}
			return levelWarn, fmt.Sprintf("%d repos with a different local user.email", len(bad))
		},
		fix: func() (string, error) {
			bad := mismatched()
			for _, rp := range bad {
				a, _ := r.Cfg.AccountFromAlias(rp.Login)
				if err := ApplyIdentity(rp.Path, a); err != nil {
					return "", err
				}
			}
			return fmt.Sprintf("rewrote identity on %d repos", len(bad)), nil
		},
	})

	checks = append(checks, check{
		name: "gh-shim",
		run: func() (string, string) {
			if !ShimInstalled() {
				return levelWarn, "not installed: gh pr/issue/api follow the global active account"
			}
			if !PathHasLocalBin() || !ZshrcHasLocalBin() {
				return levelWarn, "installed but ~/.local/bin is not on PATH"
			}
			return levelOK, TildePath(ShimPath())
		},
		fix: func() (string, error) {
			_, err := InstallShim(r)
			if err != nil {
				return "", err
			}
			return "shim installed", nil
		},
	})

	checks = append(checks, check{
		name: "askpass",
		run: func() (string, string) {
			ap := os.Getenv("GIT_ASKPASS")
			if ap == "" {
				return levelOK, "not set"
			}
			if _, err := os.Stat(ap); err != nil {
				return levelWarn, "GIT_ASKPASS points to a missing file (stale IDE env): " + ap
			}
			return levelOK, TildePath(ap)
		},
	})

	sort.SliceStable(checks, func(i, j int) bool {
		pi, pj := checkPriority(checks[i].name), checkPriority(checks[j].name)
		if pi != pj {
			return pi < pj
		}
		return checks[i].name < checks[j].name
	})
	return checks
}

func checkPriority(name string) int {
	switch {
	case name == "credential-helper", name == "https-remotes":
		return 0
	case strings.HasPrefix(name, "key:"), strings.HasPrefix(name, "ssh-"):
		return 1
	default:
		return 2
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		s = "no response"
	}
	return s
}

// DirIsUnder reports whether path is inside root (used by tests).
func DirIsUnder(path, root string) bool {
	path, root = filepath.Clean(ExpandPath(path)), filepath.Clean(ExpandPath(root))
	return path == root || strings.HasPrefix(path, root+string(os.PathSeparator))
}
