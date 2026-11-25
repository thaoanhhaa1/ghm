package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Account is one GitHub identity bound to one SSH key and one root folder.
type Account struct {
	Login     string `yaml:"login"`
	Name      string `yaml:"name"`
	Email     string `yaml:"email"`
	Key       string `yaml:"key"`
	HostAlias string `yaml:"host_alias"`
	Dir       string `yaml:"dir"`
}

// Config is the persisted state of ghm. It never contains tokens.
type Config struct {
	Version        int       `yaml:"version"`
	Root           string    `yaml:"root"`
	DefaultAccount string    `yaml:"default_account"`
	GhBinary       string    `yaml:"gh_binary"`
	ScanRoots      []string  `yaml:"scan_roots,omitempty"`
	Accounts       []Account `yaml:"accounts"`
}

const (
	appVersion   = "0.3.0"
	defaultRoot  = "~/work"
	githubHost   = "github.com"
	githubAPIURL = "https://api.github.com"
)

// Home returns the user home directory.
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return h
}

// ExpandPath expands a leading ~ and makes the result absolute.
func ExpandPath(p string) string {
	if p == "" {
		return ""
	}
	if p == "~" {
		p = Home()
	} else if strings.HasPrefix(p, "~/") {
		p = filepath.Join(Home(), p[2:])
	}
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	return p
}

// TildePath rewrites the home prefix back to ~ so configs stay portable.
func TildePath(p string) string {
	h := Home()
	if p == "" {
		return ""
	}
	if p == h {
		return "~"
	}
	if strings.HasPrefix(p, h+string(os.PathSeparator)) {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(p, h+string(os.PathSeparator)))
	}
	return p
}

// ConfigDir is ~/.config/ghm.
func ConfigDir() string { return filepath.Join(Home(), ".config", "ghm") }

// ConfigPath is the accounts.yaml location.
func ConfigPath() string { return filepath.Join(ConfigDir(), "accounts.yaml") }

// AccountGitConfigPath is the per-account gitconfig location.
func AccountGitConfigPath(login string) string {
	return filepath.Join(ConfigDir(), "git", login+".gitconfig")
}

// LoadConfig reads accounts.yaml and applies defaults.
func LoadConfig() (*Config, error) {
	raw, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	var c Config
	if err = yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ConfigPath(), err)
	}
	c.applyDefaults()
	if len(c.Accounts) == 0 {
		return nil, fmt.Errorf("%s has no accounts; run 'ghm init'", ConfigPath())
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Root == "" {
		c.Root = defaultRoot
	}
	if c.GhBinary == "" {
		c.GhBinary = "/opt/homebrew/bin/gh"
	}
	if c.DefaultAccount == "" && len(c.Accounts) > 0 {
		c.DefaultAccount = c.Accounts[0].Login
	}
	for i := range c.Accounts {
		a := &c.Accounts[i]
		if a.HostAlias == "" {
			a.HostAlias = "github-" + a.Login
		}
		if a.Key == "" {
			a.Key = "~/.ssh/id_ed25519_" + a.Login
		}
		if a.Dir == "" {
			a.Dir = filepath.Join(c.Root, a.Login)
		}
	}
}

// Save writes the config atomically.
func (c *Config) Save() error {
	c.applyDefaults()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return err
	}
	enc.Close()
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	tmp := ConfigPath() + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ConfigPath())
}

// Account returns the account by login (case-insensitive).
func (c *Config) Account(login string) (*Account, bool) {
	for i := range c.Accounts {
		if strings.EqualFold(c.Accounts[i].Login, login) {
			return &c.Accounts[i], true
		}
	}
	return nil, false
}

// AccountForDir resolves the account owning a directory, longest prefix wins.
func (c *Config) AccountForDir(dir string) (*Account, bool) {
	target := ExpandPath(dir)
	var best *Account
	var bestLen int
	for i := range c.Accounts {
		root := ExpandPath(c.Accounts[i].Dir)
		if target == root || strings.HasPrefix(target, root+string(os.PathSeparator)) {
			if len(root) > bestLen {
				best = &c.Accounts[i]
				bestLen = len(root)
			}
		}
	}
	return best, best != nil
}

// ScanRootsResolved returns the roots searched for repositories.
func (c *Config) ScanRootsResolved() []string {
	raw := c.ScanRoots
	if len(raw) == 0 {
		raw = []string{c.Root, "~/Documents", "~/Desktop"}
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range raw {
		p := ExpandPath(r)
		if seen[p] {
			continue
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// RepoRoot is the expanded per-account repository root.
func (a Account) RepoRoot() string { return ExpandPath(a.Dir) }

// SSHCommand returns the core.sshCommand value for this account.
func (a Account) SSHCommand() string {
	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", ExpandPath(a.Key))
}

// SSHCommandScript is like SSHCommand but keeps ~ for written config files.
func (a Account) SSHCommandScript() string {
	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", a.Key)
}

// SortedLogins returns account logins sorted for stable output.
func (c *Config) SortedLogins() []string {
	logins := make([]string, 0, len(c.Accounts))
	for _, a := range c.Accounts {
		logins = append(logins, a.Login)
	}
	sort.Strings(logins)
	return logins
}
