package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitHub emulates the REST and OAuth endpoints the app depends on.
func fakeGitHub(t *testing.T, login string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login/device/code", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("device/code form: %v", err)
		}
		if got := r.Form.Get("client_id"); got != defaultOAuthClientID {
			t.Errorf("client_id = %q", got)
		}
		if !strings.Contains(r.Form.Get("scope"), "admin:public_key") {
			t.Errorf("scope = %q", r.Form.Get("scope"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "dev-123",
			"user_code":        "ABCD-1234",
			"verification_uri": "https://github.com/login/device",
			"expires_in":       900,
			"interval":         5,
		})
	})
	polls := 0
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		polls++
		if polls == 1 {
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "ghp_from_device"})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token ghp_from_device" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"login": login, "id": 4242, "name": "New Bie", "email": "",
		})
	})
	mux.HandleFunc("/user/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]string{})
		case http.MethodPost:
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if !strings.HasPrefix(body["key"], "ssh-ed25519 ") {
				t.Errorf("posted key = %q", body["key"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// stubGh answers the gh calls made while adding an account.
func stubGh(t *testing.T, login string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	script := "#!/bin/sh\n" +
		"case \"$1 $2\" in\n" +
		"  'auth token') echo ghp_from_device; exit 0 ;;\n" +
		"  'auth login') cat >/dev/null; exit 0 ;;\n" +
		"esac\n" +
		"echo unexpected args: $* >&2; exit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAPIDeviceFlowAddsAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// The keyring already knows alpha; beta is the account being added.
	hosts := filepath.Join(home, ".config", "gh", "hosts.yml")
	if err := os.MkdirAll(filepath.Dir(hosts), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hosts, []byte("github.com:\n    users:\n        alpha: {}\n        beta: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := fakeGitHub(t, "beta")
	cfg := &Config{
		Root:     filepath.Join(home, "work"),
		GhBinary: stubGh(t, "beta"),
		Accounts: []Account{{Login: "alpha", Name: "Alpha", Dir: filepath.Join(home, "work", "alpha")}},
	}
	r := NewRunner(cfg)
	r.webURL = srv.URL
	r.apiURL = srv.URL

	start, err := deviceStart(r, defaultOAuthClientID)
	if err != nil {
		t.Fatalf("deviceStart: %v", err)
	}
	if start.UserCode != "ABCD-1234" || start.Interval != 5 {
		t.Fatalf("device code = %+v", start)
	}

	status, res := devicePoll(r, defaultOAuthClientID, start.DeviceCode)
	if status != "pending" || res.Error != "" {
		t.Fatalf("first poll = %q %+v, want pending", status, res)
	}

	status, res = devicePoll(r, defaultOAuthClientID, start.DeviceCode)
	if status != "success" {
		t.Fatalf("second poll = %q %+v, want success", status, res)
	}
	if !res.OK {
		t.Fatalf("result = %+v", res)
	}
	if len(res.Logins) != 1 || res.Logins[0] != "beta" {
		t.Fatalf("logins = %v, want [beta]", res.Logins)
	}

	joined := strings.Join(res.Lines, "\n")
	for _, want := range []string{"account: beta", "key: generated", "key: registered on github.com", "git: includeIf", "next: ghm doctor"} {
		if !strings.Contains(joined, want) {
			t.Errorf("lines missing %q\n%s", want, joined)
		}
	}

	// The account must be persisted, with key, ssh host, gitconfig and folder.
	saved, err := LoadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	beta, ok := saved.Account("beta")
	if !ok {
		t.Fatalf("beta not saved in accounts.yaml: %+v", saved.Accounts)
	}
	if !strings.HasSuffix(beta.Email, "+beta@users.noreply.github.com") {
		t.Errorf("email = %q, want noreply fallback", beta.Email)
	}
	if _, err := os.Stat(ExpandPath(beta.Key)); err != nil {
		t.Errorf("key file: %v", err)
	}
	if _, err := os.Stat(beta.Dir); err != nil {
		t.Errorf("account folder: %v", err)
	}
	sshConf, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sshConf), "Host github-beta") {
		t.Errorf("~/.ssh/config missing Host github-beta:\n%s", sshConf)
	}
	gitConf, err := os.ReadFile(AccountGitConfigPath("beta"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gitConf), "email = ") {
		t.Errorf("gitconfig = %s", gitConf)
	}

	// Adding the same login again must be refused.
	if _, err := AddAccount(r, "beta"); err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Errorf("second add err = %v", err)
	}
}

func TestAPIDeviceExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
	}))
	defer srv.Close()

	r := NewRunner(&Config{})
	r.webURL = srv.URL
	status, res := devicePoll(r, defaultOAuthClientID, "dev-1")
	if status != "error" || !strings.Contains(res.Error, "expired") {
		t.Errorf("poll = %q %+v", status, res)
	}
}

func TestAPIGitActionValidatesInput(t *testing.T) {
	if res := apiGitAction("", "fetch"); res.Error == "" {
		t.Error("missing path should fail")
	}
	if res := apiGitAction("/tmp", "rebase"); !strings.Contains(res.Error, "unknown action") {
		t.Errorf("unknown action = %+v", res)
	}
}
