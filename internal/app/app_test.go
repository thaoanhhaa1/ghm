package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseSSHAlias(t *testing.T) {
	cases := []struct {
		remote string
		login  string
		slug   string
		ok     bool
	}{
		{"git@github-anhthaodev:anhthaodev/misa.git", "anhthaodev", "anhthaodev/misa", true},
		{"git@github-thaoanhhaa1:nguyenhuong9x/FoodShare.git", "thaoanhhaa1", "nguyenhuong9x/FoodShare", true},
		{"git@github.com:owner/repo.git", "", "", false},
		{"https://github.com/owner/repo.git", "", "", false},
		{"git@github-x:noslug.git", "", "", false},
	}
	for _, c := range cases {
		login, slug, ok := ParseSSHAlias(c.remote)
		if ok != c.ok || login != c.login || slug != c.slug {
			t.Errorf("ParseSSHAlias(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.remote, login, slug, ok, c.login, c.slug, c.ok)
		}
	}
}

func TestParseHTTPSGitHub(t *testing.T) {
	cases := []struct {
		remote string
		slug   string
		ok     bool
	}{
		{"https://github.com/anhthaodev/misa.git", "anhthaodev/misa", true},
		{"https://github.com/namphuongso/MyLibrary", "namphuongso/MyLibrary", true},
		{"https://github.com/org/repo/tree/main", "", false},
		{"https://dev.azure.com/x/y/_git/z", "", false},
		{"git@github-anhthaodev:a/b.git", "", false},
	}
	for _, c := range cases {
		slug, ok := ParseHTTPSGitHub(c.remote)
		if ok != c.ok || slug != c.slug {
			t.Errorf("ParseHTTPSGitHub(%q) = (%q,%v), want (%q,%v)", c.remote, slug, ok, c.slug, c.ok)
		}
	}
}

func TestAccountForDir(t *testing.T) {
	base := t.TempDir()
	cfg := &Config{
		Root: base,
		Accounts: []Account{
			{Login: "one", Dir: filepath.Join(base, "one")},
			{Login: "two", Dir: filepath.Join(base, "two")},
		},
	}
	cases := []struct {
		dir  string
		want string
	}{
		{filepath.Join(base, "one", "repo"), "one"},
		{filepath.Join(base, "one"), "one"},
		{filepath.Join(base, "two", "deep", "repo"), "two"},
		{filepath.Join(base, "three", "repo"), ""},
		{base, ""},
	}
	for _, c := range cases {
		a, ok := cfg.AccountForDir(c.dir)
		got := ""
		if ok {
			got = a.Login
		}
		if got != c.want {
			t.Errorf("AccountForDir(%q) = %q, want %q", c.dir, got, c.want)
		}
	}
}

func TestAccountGitConfig(t *testing.T) {
	a := Account{
		Login:     "alpha",
		Name:      "Alpha",
		Email:     "1+alpha@users.noreply.github.com",
		Key:       "~/.ssh/id_ed25519_alpha",
		HostAlias: "github-alpha",
	}
	got := AccountGitConfig(a)
	for _, want := range []string{
		"email = 1+alpha@users.noreply.github.com",
		"sshCommand = ssh -i ~/.ssh/id_ed25519_alpha -o IdentitiesOnly=yes",
		`insteadOf = https://github.com/`,
		`[url "git@github-alpha:"]`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AccountGitConfig missing %q\n%s", want, got)
		}
	}
}

func TestRemoteURLFor(t *testing.T) {
	a := Account{HostAlias: "github-alpha"}
	if got := RemoteURLFor(a, "owner/repo"); got != "git@github-alpha:owner/repo.git" {
		t.Errorf("RemoteURLFor = %q", got)
	}
}

func TestPositionalSkipsFlagValues(t *testing.T) {
	args := []string{"owner/repo", "--account", "alpha", "--json", "--jobs=8"}
	got := positional(args)
	if len(got) != 1 || got[0] != "owner/repo" {
		t.Errorf("positional = %v, want [owner/repo]", got)
	}
}

func TestTildeRoundTrip(t *testing.T) {
	home := Home()
	p := filepath.Join(home, "work", "alpha")
	if got := TildePath(p); got != "~/work/alpha" {
		t.Errorf("TildePath = %q", got)
	}
	if got := ExpandPath("~/work/alpha"); got != p {
		t.Errorf("ExpandPath = %q, want %q", got, p)
	}
	if _, err := os.Stat(home); err != nil {
		t.Skip("no home dir")
	}
}

func TestTUIViewRenders(t *testing.T) {
	cfg := &Config{Accounts: []Account{
		{Login: "alpha", Email: "a@x", Dir: "/tmp/alpha"},
		{Login: "beta", Email: "b@x", Dir: "/tmp/beta"},
	}}
	m := &tuiModel{
		r:        NewRunner(cfg),
		accounts: cfg.Accounts,
		all: []Repo{
			{Path: "/tmp/alpha/one", Login: "alpha", Slug: "alpha/one", Branch: "main", Dirty: 2},
			{Path: "/tmp/alpha/two", Login: "alpha", Slug: "alpha/two", Branch: "dev", Behind: 3},
			{Path: "/tmp/beta/three", Login: "beta", Slug: "beta/three", Branch: "main"},
		},
		width:  120,
		height: 30,
		status: "ready",
	}
	m.applyFilter()
	view := m.View()
	for _, want := range []string{"ghm " + appVersion, "alpha", "REPO", "alpha/one", "2 dirty", "behind"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q\n%s", want, view)
		}
	}
	if strings.Contains(view, "beta/three") {
		t.Error("View() leaks repos from the other account")
	}
	m.filter = "two"
	m.applyFilter()
	if len(m.visible) != 1 || m.visible[0].Slug != "alpha/two" {
		t.Errorf("filter = %v", m.visible)
	}
	m.acct = 1
	m.filter = ""
	m.applyFilter()
	if len(m.visible) != 1 || m.visible[0].Slug != "beta/three" {
		t.Errorf("account switch = %v", m.visible)
	}
}

func TestLogOverlayRenders(t *testing.T) {
	cfg := &Config{Accounts: []Account{{Login: "alpha", Dir: "/tmp/alpha"}}}
	m := &tuiModel{
		r:        NewRunner(cfg),
		accounts: cfg.Accounts,
		all:      []Repo{{Path: "/tmp/alpha/one", Login: "alpha", Slug: "alpha/one", Branch: "main"}},
		width:    100,
		height:   24,
		status:   "3 commits",
		overlay:  "log",
		logTitle: "alpha/one  main",
		logLines: []string{
			"a1b2c3d 2026-11-24 Alice first commit",
			"d4e5f6a 2026-11-23 Bob second commit",
			"7g8h9i0 2026-11-22 Alice third commit",
		},
	}
	m.applyFilter()
	view := m.View()
	for _, want := range []string{"log  alpha/one  main", "a1b2c3d", "third commit", "j/k scroll", "esc close"} {
		if !strings.Contains(view, want) {
			t.Errorf("log view missing %q\n%s", want, view)
		}
	}
	if strings.Contains(view, "REPO") {
		t.Error("log overlay should replace the repo table")
	}

	// esc closes the overlay and keeps the status
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := model.(*tuiModel)
	if got.overlay != "" {
		t.Errorf("overlay = %q, want empty", got.overlay)
	}
	if !strings.Contains(got.View(), "REPO") {
		t.Error("table did not come back after closing the log")
	}
}

func TestLastURL(t *testing.T) {
	cases := map[string]string{
		"Creating pull request for x\nhttps://github.com/a/b/pull/7\n": "https://github.com/a/b/pull/7",
		"warning: something\nhttps://github.com/a/b/pull/9":            "https://github.com/a/b/pull/9",
		"no url here": "",
	}
	for in, want := range cases {
		if got := lastURL(in); got != want {
			t.Errorf("lastURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPRCmdRequiresRemote(t *testing.T) {
	msg := prCmd(nil, Repo{Path: "/tmp"})()
	am, ok := msg.(actionMsg)
	if !ok {
		t.Fatalf("prCmd msg = %T", msg)
	}
	if am.err == nil || !strings.Contains(am.err.Error(), "github remote") {
		t.Errorf("err = %v, want a github remote error", am.err)
	}
}

func TestStickyStatusSurvivesReload(t *testing.T) {
	cfg := &Config{Accounts: []Account{{Login: "alpha", Dir: "/tmp/alpha"}}}
	m := &tuiModel{
		r:        NewRunner(cfg),
		accounts: cfg.Accounts,
		width:    100,
		height:   24,
		sticky:   true,
		status:   "PR created: https://github.com/a/b/pull/7",
		all:      []Repo{{Path: "/tmp/alpha/one", Login: "alpha", Slug: "alpha/one"}},
	}
	m.applyFilter()
	model, _ := m.Update(reposMsg{{Path: "/tmp/alpha/one", Login: "alpha", Slug: "alpha/one"}})
	got := model.(*tuiModel)
	if !strings.Contains(got.status, "PR created") {
		t.Errorf("status = %q, want the PR message to survive the rescan", got.status)
	}
}

func TestPRCmdPushesAndCreatesPR(t *testing.T) {
	work := t.TempDir()
	remote := filepath.Join(work, "remote.git")
	repo := filepath.Join(work, "repo")

	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}

	git(work, "init", "--bare", remote)
	git(work, "init", "-b", "main", repo)
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(repo, "config", "user.email", "a@x")
	git(repo, "config", "user.name", "A")
	git(repo, "add", "a.txt")
	git(repo, "commit", "-m", "feat: first commit")
	git(repo, "remote", "add", "origin", remote)
	git(repo, "checkout", "-b", "feature")

	stub := filepath.Join(work, "gh")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = auth ] && [ \"$2\" = token ]; then echo tok123; exit 0; fi\n" +
		"if [ \"$1\" = pr ]; then echo https://github.com/acme/demo/pull/1; exit 0; fi\n" +
		"echo unexpected args: $* >&2; exit 1\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		GhBinary: stub,
		Accounts: []Account{{Login: "alpha", HostAlias: "github-alpha", Dir: "/tmp/alpha"}},
	}
	msg := prCmd(NewRunner(cfg), Repo{Path: repo, Login: "alpha", Slug: "acme/demo", Branch: "feature"})()
	am, ok := msg.(actionMsg)
	if !ok {
		t.Fatalf("prCmd msg = %T", msg)
	}
	if am.err != nil {
		t.Fatalf("prCmd err = %v", am.err)
	}
	if !strings.Contains(am.desc, "https://github.com/acme/demo/pull/1") {
		t.Errorf("desc = %q", am.desc)
	}
	if out := git(remote, "branch", "--list", "feature"); !strings.Contains(out, "feature") {
		t.Errorf("branch was not pushed to origin: %q", out)
	}
}

func TestFindRepoRoot(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	deep := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findRepoRoot(deep); got != repo {
		t.Errorf("findRepoRoot = %q, want %q", got, repo)
	}
}
