package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
