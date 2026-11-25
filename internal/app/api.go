package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Machine-readable surface for the macOS app. Every action prints one JSON
// object and exits 0, so the UI can render partial failures (warnings, missing
// scopes) instead of guessing from an exit code.

const defaultOAuthClientID = "Iv1.b507a08c87ecfe98" // GitHub CLI public OAuth app

type apiAccount struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Dir       string `json:"dir"`
	Key       string `json:"key"`
	HostAlias string `json:"host_alias"`
	KeyOK     bool   `json:"key_ok"`
	TokenOK   bool   `json:"token_ok"`
	Repos     int    `json:"repos"`
}

type apiLocalRepo struct {
	Path     string `json:"path"`
	Slug     string `json:"slug"`
	Login    string `json:"login"`
	Branch   string `json:"branch"`
	Origin   string `json:"origin"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
	Dirty    int    `json:"dirty"`
	IsGitHub bool   `json:"is_github"`
}

type apiState struct {
	Active   string         `json:"active"`
	Root     string         `json:"root"`
	Accounts []apiAccount   `json:"accounts"`
	Repos    []apiLocalRepo `json:"repos"`
}

type apiCheck struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	Detail string `json:"detail"`
}

type apiDoctor struct {
	Checks []apiCheck `json:"checks"`
	OK     int        `json:"ok"`
	Warn   int        `json:"warn"`
	Fail   int        `json:"fail"`
}

type apiResult struct {
	OK     bool     `json:"ok"`
	Logins []string `json:"logins,omitempty"`
	Lines  []string `json:"lines,omitempty"`
	Error  string   `json:"error,omitempty"`
	Output string   `json:"output,omitempty"`
}

type apiDevice struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Error           string `json:"error,omitempty"`
}

type apiDevicePoll struct {
	Status string    `json:"status"` // pending | slow_down | success | error
	Result apiResult `json:"result"`
}

// cmdAPI dispatches the JSON API used by the desktop app.
func cmdAPI(r *Runner, args []string) int {
	if len(args) == 0 {
		printJSON(apiResult{Error: "usage: ghm api <state|doctor|add|login-token|device-start|device-poll|git>"})
		return 0
	}
	sub, args := args[0], args[1:]
	switch sub {
	case "state":
		repos := scanAll(r.Cfg, true)
		st := apiState{Active: r.GhActiveLogin(), Root: r.Cfg.Root}
		st.Accounts = buildAPIAccounts(r, repos)
		st.Repos = make([]apiLocalRepo, 0, len(repos))
		for _, rp := range repos {
			st.Repos = append(st.Repos, apiLocalRepo{
				Path: rp.Path, Slug: rp.Slug, Login: rp.Login, Branch: rp.Branch,
				Origin: rp.Origin, Ahead: rp.Ahead, Behind: rp.Behind,
				Dirty: rp.Dirty, IsGitHub: rp.IsGitHub,
			})
		}
		printJSON(st)

	case "doctor":
		doFix := hasFlag(args, "--fix")
		checks := buildChecks(r, scanAll(r.Cfg, false))
		out := apiDoctor{}
		for _, c := range checks {
			level, detail := c.run()
			if doFix && level != levelOK && c.fix != nil {
				if msg, err := c.fix(); err != nil {
					detail += fmt.Sprintf(" (fix failed: %v)", err)
				} else {
					level, detail = c.run()
					detail = strings.TrimSpace(msg + " -> " + detail)
				}
			}
			switch level {
			case levelOK:
				out.OK++
			case levelWarn:
				out.Warn++
			default:
				out.Fail++
			}
			out.Checks = append(out.Checks, apiCheck{Name: c.name, Level: level, Detail: detail})
		}
		printJSON(out)

	case "add":
		pos := positional(args)
		if len(pos) != 1 {
			printJSON(apiResult{Error: "usage: ghm api add <login>"})
			return 0
		}
		lines, err := AddAccount(r, pos[0])
		res := apiResult{OK: err == nil, Lines: lines}
		if err != nil {
			res.Error = err.Error()
		}
		printJSON(res)

	case "login-token":
		raw, _ := io.ReadAll(os.Stdin)
		printJSON(storeTokenAndAdd(r, strings.TrimSpace(string(raw))))

	case "device-start":
		clientID := flagValue(args, "--client-id")
		if clientID == "" {
			clientID = defaultOAuthClientID
		}
		d, err := deviceStart(r, clientID)
		if err != nil {
			printJSON(apiDevice{Error: err.Error()})
			return 0
		}
		printJSON(d)

	case "device-poll":
		clientID := flagValue(args, "--client-id")
		if clientID == "" {
			clientID = defaultOAuthClientID
		}
		code := flagValue(args, "--device-code")
		if code == "" {
			printJSON(apiDevicePoll{Status: "error", Result: apiResult{Error: "missing --device-code"}})
			return 0
		}
		status, res := devicePoll(r, clientID, code)
		printJSON(apiDevicePoll{Status: status, Result: res})

	case "git":
		path := flagValue(args, "--path")
		action := flagValue(args, "--action")
		printJSON(apiGitAction(path, action))

	default:
		printJSON(apiResult{Error: "unknown api action " + sub})
	}
	return 0
}

func buildAPIAccounts(r *Runner, repos []Repo) []apiAccount {
	counts := map[string]int{}
	for _, rp := range repos {
		if rp.Login != "" {
			counts[rp.Login]++
		}
	}
	out := make([]apiAccount, 0, len(r.Cfg.Accounts))
	for _, a := range r.Cfg.Accounts {
		_, terr := r.Token(a.Login)
		out = append(out, apiAccount{
			Login: a.Login, Name: a.Name, Email: a.Email, Dir: a.Dir,
			Key: a.Key, HostAlias: a.HostAlias,
			KeyOK: KeyExists(a), TokenOK: terr == nil, Repos: counts[a.Login],
		})
	}
	return out
}

func apiGitAction(path, action string) apiResult {
	if path == "" {
		return apiResult{Error: "missing --path"}
	}
	var args []string
	switch action {
	case "fetch":
		args = []string{"fetch", "--prune"}
	case "pull":
		args = []string{"pull", "--ff-only"}
	case "push":
		args = []string{"push"}
	default:
		return apiResult{Error: "unknown action " + action + " (fetch, pull, push)"}
	}
	out, err := Git(path, args...)
	res := apiResult{OK: err == nil, Output: out}
	if err != nil {
		res.Error = err.Error()
	}
	return res
}

// storeTokenAndAdd saves a token in the gh keyring, then configures the account
// it belongs to.
func storeTokenAndAdd(r *Runner, token string) apiResult {
	if token == "" {
		return apiResult{Error: "empty token"}
	}
	cmd := exec.Command(r.GhPath(), "auth", "login", "--with-token", "--hostname", "github.com")
	cmd.Stdin = strings.NewReader(token + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := firstLine(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = err.Error()
		}
		return apiResult{Error: msg}
	}

	after, err := KeyringLogins(r.GhPath())
	if err != nil {
		return apiResult{Error: err.Error()}
	}
	var before []string
	for _, a := range r.Cfg.Accounts {
		before = append(before, a.Login)
	}
	pending := newLogins(before, after)
	if len(pending) == 0 {
		return apiResult{Error: "this token belongs to an account that is already configured"}
	}
	lines := []string{"token stored in the gh keyring"}
	res := apiResult{Logins: pending}
	for _, login := range pending {
		l, aerr := AddAccount(r, login)
		lines = append(lines, l...)
		if aerr != nil {
			res.Lines = lines
			res.Error = aerr.Error()
			return res
		}
	}
	res.OK = true
	res.Lines = lines
	return res
}

func deviceStart(r *Runner, clientID string) (apiDevice, error) {
	form := url.Values{
		"client_id": {clientID},
		"scope":     {"repo read:org admin:public_key workflow gist"},
	}
	req, err := http.NewRequest(http.MethodPost, r.webBase()+"/login/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return apiDevice{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := r.client.Do(req)
	if err != nil {
		return apiDevice{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d apiDevice
	if err := json.Unmarshal(body, &d); err != nil {
		return apiDevice{}, fmt.Errorf("github device/code: %v %s", err, strings.TrimSpace(string(body)))
	}
	if d.Error != "" {
		return apiDevice{}, fmt.Errorf("github device/code: %s", d.Error)
	}
	if d.DeviceCode == "" {
		return apiDevice{}, fmt.Errorf("github device/code: empty response")
	}
	if d.Interval == 0 {
		d.Interval = 5
	}
	return d, nil
}

func devicePoll(r *Runner, clientID, deviceCode string) (string, apiResult) {
	form := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	req, err := http.NewRequest(http.MethodPost, r.webBase()+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "error", apiResult{Error: err.Error()}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := r.client.Do(req)
	if err != nil {
		return "error", apiResult{Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var parsed struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "error", apiResult{Error: fmt.Sprintf("github access_token: %v %s", err, strings.TrimSpace(string(body)))}
	}
	if parsed.AccessToken != "" {
		return "success", storeTokenAndAdd(r, parsed.AccessToken)
	}
	switch parsed.Error {
	case "authorization_pending":
		return "pending", apiResult{}
	case "slow_down":
		return "slow_down", apiResult{}
	case "expired_token":
		return "error", apiResult{Error: "the device code expired; start again"}
	case "access_denied":
		return "error", apiResult{Error: "access denied in the browser"}
	default:
		msg := parsed.Error
		if parsed.Description != "" {
			msg += ": " + parsed.Description
		}
		if msg == "" {
			msg = "unexpected response from github"
		}
		return "error", apiResult{Error: msg}
	}
}

// apiBase and webBase keep the REST and OAuth endpoints overridable so tests can
// run against a local server instead of github.com.
func (r *Runner) apiBase() string {
	if r.apiURL != "" {
		return r.apiURL
	}
	return githubAPIURL
}

func (r *Runner) webBase() string {
	if r.webURL != "" {
		return r.webURL
	}
	return githubWebURL
}
