package app

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Git runs git in dir and returns trimmed stdout.
func Git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return strings.TrimSpace(out.String()), fmt.Errorf("git %s: %v %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// Repo is one local git repository.
type Repo struct {
	Path     string
	Origin   string
	Login    string // account derived from the remote alias
	Slug     string
	Branch   string
	Ahead    int
	Behind   int
	Dirty    int
	IsGitHub bool
}

// ParseSSHAlias extracts login + slug from git@github-<login>:owner/repo.git.
func ParseSSHAlias(remote string) (login, slug string, ok bool) {
	remote = strings.TrimSpace(remote)
	if !strings.HasPrefix(remote, "git@github-") {
		return "", "", false
	}
	rest := strings.TrimPrefix(remote, "git@github-")
	i := strings.Index(rest, ":")
	if i <= 0 {
		return "", "", false
	}
	login = rest[:i]
	slug = strings.TrimSuffix(rest[i+1:], ".git")
	if slug == "" || !strings.Contains(slug, "/") {
		return "", "", false
	}
	return login, slug, true
}

// ParseHTTPSGitHub extracts the owner/repo slug from a github.com https URL.
func ParseHTTPSGitHub(remote string) (slug string, ok bool) {
	remote = strings.TrimSpace(remote)
	prefixes := []string{"https://github.com/", "http://github.com/", "https://www.github.com/"}
	for _, p := range prefixes {
		if strings.HasPrefix(remote, p) {
			s := strings.TrimSuffix(strings.TrimPrefix(remote, p), ".git")
			s = strings.TrimSuffix(s, "/")
			if strings.Count(s, "/") == 1 {
				return s, true
			}
		}
	}
	return "", false
}

// HTTPSAliasToSSH converts a github https URL to an alias URL for login.
func HTTPSAliasToSSH(slug, login string) string {
	return fmt.Sprintf("git@github-%s:%s.git", login, slug)
}

// SkipDirs are directory names never scanned for repositories.
var SkipDirs = map[string]bool{
	".git": true, "node_modules": true, "Library": true, ".Trash": true,
	".cache": true, "vendor": true, "Pods": true, "DerivedData": true,
	".npm": true, ".nvm": true, ".local": true, "go": true,
}

// ScanRepos walks roots up to maxDepth and returns repository paths.
func ScanRepos(roots []string, maxDepth int) []string {
	var found []string
	seen := map[string]bool{}
	for _, root := range roots {
		rootDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			base := d.Name()
			if base == ".git" {
				repo := filepath.Dir(path)
				if !seen[repo] {
					seen[repo] = true
					found = append(found, repo)
				}
				return filepath.SkipDir
			}
			if path != root && SkipDirs[base] {
				return filepath.SkipDir
			}
			if strings.Count(path, string(os.PathSeparator))-rootDepth >= maxDepth {
				return filepath.SkipDir
			}
			return nil
		})
	}
	sort.Strings(found)
	return found
}

// LoadRepo reads origin only (fast path for scanning many repos).
func LoadRepo(path string) Repo {
	r := Repo{Path: path}
	r.Origin, _ = Git(path, "remote", "get-url", "origin")
	if login, slug, ok := ParseSSHAlias(r.Origin); ok {
		r.Login, r.Slug, r.IsGitHub = login, slug, true
	} else if slug, ok := ParseHTTPSGitHub(r.Origin); ok {
		r.Slug, r.IsGitHub = slug, true
	}
	return r
}

// LoadRepoFull also reads branch, ahead/behind and dirty state.
func LoadRepoFull(path string) Repo {
	r := LoadRepo(path)
	r.Branch, _ = Git(path, "rev-parse", "--abbrev-ref", "HEAD")
	if out, err := Git(path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		parts := strings.Fields(out)
		if len(parts) == 2 {
			r.Ahead, _ = strconv.Atoi(parts[0])
			r.Behind, _ = strconv.Atoi(parts[1])
		}
	}
	if out, err := Git(path, "status", "--porcelain"); err == nil && out != "" {
		r.Dirty = len(strings.Split(out, "\n"))
	}
	return r
}

// AccountFromAlias finds the configured account for a repo remote.
func (c *Config) AccountFromAlias(login string) (Account, bool) {
	if a, ok := c.Account(login); ok {
		return *a, true
	}
	return Account{}, false
}

// RemoteURLFor builds the canonical SSH remote for a repo and account.
func RemoteURLFor(a Account, slug string) string {
	return fmt.Sprintf("git@%s:%s.git", a.HostAlias, slug)
}

// SetOrigin rewrites origin.
func SetOrigin(path, url string) error {
	_, err := Git(path, "remote", "set-url", "origin", url)
	return err
}

// ApplyIdentity writes local user.* and core.sshCommand for a repo.
func ApplyIdentity(path string, a Account) error {
	steps := [][]string{
		{"config", "--local", "user.name", a.Name},
		{"config", "--local", "user.email", a.Email},
		{"config", "--local", "core.sshCommand", a.SSHCommandScript()},
	}
	for _, s := range steps {
		if _, err := Git(path, s...); err != nil {
			return err
		}
	}
	return nil
}

// LocalEmail returns the repo-local (not inherited) user.email.
func LocalEmail(path string) string {
	out, err := Git(path, "config", "--local", "--get", "user.email")
	if err != nil {
		return ""
	}
	return out
}
