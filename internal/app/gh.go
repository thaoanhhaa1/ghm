package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Runner groups config plus the gh binary used for token access.
type Runner struct {
	Cfg      *Config
	tokens   map[string]string
	apiCache map[string]apiRepo
	client   *http.Client
}

// NewRunner builds a Runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		Cfg:      cfg,
		tokens:   map[string]string{},
		apiCache: map[string]apiRepo{},
		client:   &http.Client{Timeout: 25 * time.Second},
	}
}

// GhPath returns the configured gh binary.
func (r *Runner) GhPath() string {
	if r.Cfg != nil && r.Cfg.GhBinary != "" {
		if _, err := os.Stat(r.Cfg.GhBinary); err == nil {
			return r.Cfg.GhBinary
		}
	}
	for _, p := range []string{"/opt/homebrew/bin/gh", "/usr/local/bin/gh", "/usr/bin/gh"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("gh"); err == nil {
		return p
	}
	return "gh"
}

// Token returns (and memoizes) the gh token of one account.
func (r *Runner) Token(login string) (string, error) {
	if t, ok := r.tokens[login]; ok {
		return t, nil
	}
	cmd := exec.Command(r.GhPath(), "auth", "token", "--user", login)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh auth token --user %s: %v %s", login, err, strings.TrimSpace(errb.String()))
	}
	t := strings.TrimSpace(out.String())
	if t == "" {
		return "", fmt.Errorf("gh auth token --user %s returned empty token", login)
	}
	r.tokens[login] = t
	return t, nil
}

// GhActiveLogin returns the currently active gh account (best effort).
func (r *Runner) GhActiveLogin() string {
	cmd := exec.Command(r.GhPath(), "api", "user", "--jq", ".login")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

type hostsFile map[string]struct {
	User  string                 `yaml:"user"`
	Users map[string]interface{} `yaml:"users"`
}

// KeyringLogins lists gh-authenticated accounts for github.com.
func KeyringLogins(ghPath string) ([]string, error) {
	path := filepath.Join(Home(), ".config", "gh", "hosts.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var hf hostsFile
	if err = yaml.Unmarshal(raw, &hf); err != nil {
		return nil, err
	}
	host, ok := hf[githubHost]
	if !ok {
		return nil, fmt.Errorf("%s: no %s entry", path, githubHost)
	}
	var logins []string
	for login := range host.Users {
		logins = append(logins, login)
	}
	if len(logins) == 0 && host.User != "" {
		logins = append(logins, host.User)
	}
	return logins, nil
}

type apiRepo struct {
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	Permissions struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (r *Runner) api(token, method, path string, body []byte) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, githubAPIURL+path, rd)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ghm/"+appVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

// RepoInfo fetches repo permissions with one account token.
func (r *Runner) RepoInfo(login, slug string) (apiRepo, int, error) {
	key := login + "|" + slug
	if v, ok := r.apiCache[key]; ok {
		return v, v.Status, nil
	}
	tok, err := r.Token(login)
	if err != nil {
		return apiRepo{}, 0, err
	}
	data, status, err := r.api(tok, http.MethodGet, "/repos/"+slug, nil)
	if err != nil {
		return apiRepo{}, status, err
	}
	var info apiRepo
	_ = json.Unmarshal(data, &info)
	info.Status = status
	if status == http.StatusOK {
		r.apiCache[key] = info
	}
	return info, status, nil
}

// ResolveResult describes which account should handle a repository.
type ResolveResult struct {
	Login     string
	Mode      string // "push" or "read-only"
	Preferred bool   // account matched the repository owner
}

// ResolveSlug picks the account with write access, preferring the owner login.
func (r *Runner) ResolveSlug(slug string, override string) (ResolveResult, error) {
	slug = strings.TrimSuffix(slug, ".git")
	owner := slug
	if i := strings.Index(slug, "/"); i > 0 {
		owner = slug[:i]
	}
	if override != "" {
		if _, ok := r.Cfg.Account(override); !ok {
			return ResolveResult{}, fmt.Errorf("unknown account %q", override)
		}
		return ResolveResult{Login: override, Mode: "push", Preferred: strings.EqualFold(override, owner)}, nil
	}
	type cand struct {
		login string
		pull  bool
	}
	var pullers []cand
	for _, a := range r.Cfg.Accounts {
		info, status, err := r.RepoInfo(a.Login, slug)
		if err != nil {
			return ResolveResult{}, err
		}
		if status == http.StatusOK && info.Permissions.Push {
			return ResolveResult{Login: a.Login, Mode: "push", Preferred: strings.EqualFold(a.Login, owner)}, nil
		}
		if status == http.StatusOK && info.Permissions.Pull {
			pullers = append(pullers, cand{a.Login, true})
		}
	}
	if len(pullers) > 0 {
		for _, c := range pullers {
			if strings.EqualFold(c.login, owner) {
				return ResolveResult{Login: c.login, Mode: "read-only", Preferred: true}, nil
			}
		}
		return ResolveResult{Login: pullers[0].login, Mode: "read-only"}, nil
	}
	return ResolveResult{}, fmt.Errorf("no account has access to %s", slug)
}

// UserInfo is the subset of GET /user that ghm needs.
type UserInfo struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserInfo fetches the account profile with its own token.
func (r *Runner) UserInfo(login string) (UserInfo, error) {
	tok, err := r.Token(login)
	if err != nil {
		return UserInfo{}, err
	}
	data, status, err := r.api(tok, http.MethodGet, "/user", nil)
	if err != nil {
		return UserInfo{}, err
	}
	if status != http.StatusOK {
		return UserInfo{}, fmt.Errorf("account %s: GET /user -> HTTP %d", login, status)
	}
	var u UserInfo
	if err := json.Unmarshal(data, &u); err != nil {
		return UserInfo{}, err
	}
	if u.Login == "" {
		return UserInfo{}, fmt.Errorf("account %s: empty /user response", login)
	}
	return u, nil
}

// PublicKeyBody returns the key material without the comment.
func PublicKeyBody(pubPath string) (string, error) {
	raw, err := os.ReadFile(ExpandPath(pubPath))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return "", fmt.Errorf("%s: not a public key", pubPath)
	}
	return fields[0] + " " + fields[1], nil
}

// KeyOnAccount reports whether the public key is already registered.
func (r *Runner) KeyOnAccount(login, pubPath string) (bool, error) {
	want, err := PublicKeyBody(pubPath)
	if err != nil {
		return false, err
	}
	tok, err := r.Token(login)
	if err != nil {
		return false, err
	}
	data, status, err := r.api(tok, http.MethodGet, "/user/keys?per_page=100", nil)
	if err != nil {
		return false, err
	}
	switch status {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, fmt.Errorf("account %s: token lacks scope admin:public_key (run: gh auth switch -u %s && gh auth refresh -h github.com -s admin:public_key)", login, login)
	default:
		return false, fmt.Errorf("account %s: GET /user/keys -> HTTP %d", login, status)
	}
	var keys []struct {
		Key string `json:"key"`
	}
	if err = json.Unmarshal(data, &keys); err != nil {
		return false, err
	}
	for _, k := range keys {
		if strings.TrimSpace(k.Key) == want {
			return true, nil
		}
	}
	return false, nil
}

// AddKeyToAccount registers the public key with the account.
func (r *Runner) AddKeyToAccount(login, pubPath string) error {
	pub, err := os.ReadFile(ExpandPath(pubPath))
	if err != nil {
		return err
	}
	title := "macbook-" + login
	body, _ := json.Marshal(map[string]string{
		"title": title,
		"key":   strings.TrimSpace(string(pub)),
	})
	tok, err := r.Token(login)
	if err != nil {
		return err
	}
	data, status, err := r.api(tok, http.MethodPost, "/user/keys", body)
	if err != nil {
		return err
	}
	if status == http.StatusCreated {
		return nil
	}
	if status == http.StatusUnprocessableEntity && strings.Contains(string(data), "key is already in use") {
		return nil
	}
	if status == http.StatusNotFound || status == http.StatusForbidden {
		return fmt.Errorf("account %s: cannot add key (HTTP %d). Grant the scope first:\n  gh auth switch -u %s && gh auth refresh -h github.com -s admin:public_key", login, status, login)
	}
	return fmt.Errorf("account %s: add key -> HTTP %d %s", login, status, strings.TrimSpace(string(data)))
}

// SSHAuth verifies that the alias authenticates as the expected login.
func (r *Runner) SSHAuth(a Account) (string, error) {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10", "-T", "git@"+a.HostAlias)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // ssh exits 1 even on success
	text := out.String()
	if strings.Contains(text, "successfully authenticated") {
		return text, nil
	}
	return text, fmt.Errorf("%s", strings.TrimSpace(text))
}
