package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
)

const usageText = `ghm - one machine, many GitHub accounts, zero re-login

Usage:
  ghm init                      bootstrap keys, ssh config, git config from gh accounts
  ghm list [--check] [--json]   accounts overview
  ghm status                    accounts + repositories health (exit 1 on problems)
  ghm doctor [--fix]            run health checks, optionally repair
  ghm repos [--account X] [--json]
  ghm sync [--account X] [--jobs N]
  ghm adopt <path...> [--account X] [--move] [--dry-run]
  ghm clone <owner/repo> [--account X]
  ghm exec <login> -- <cmd...>
  ghm use <login>               print exports for the current shell
  ghm token <login>
  ghm tui                       dashboard: accounts, repos, fetch/pull/push
  ghm install-shim | uninstall-shim
  ghm completion <zsh|bash|fish>
  ghm version

Account selection is path based: ~/work/<login>/... -> that account.
Run 'ghm init' on a new machine, then 'ghm install-shim' to make gh follow folders.
`

// Main dispatches a command line.
func Main(argv []string) int {
	if len(argv) == 0 {
		fmt.Print(usageText)
		return 2
	}
	cmd, args := argv[0], argv[1:]
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usageText)
		return 0
	case "version", "--version", "-v":
		fmt.Println("ghm " + appVersion)
		return 0
	case "init":
		return cmdInit(args)
	case "install-shim":
		return cmdInstallShim(args)
	case "uninstall-shim":
		return cmdUninstallShim(args)
	case "shim-token":
		return cmdShimToken(args)
	case "completion":
		return cmdCompletion(args)
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm: %v\n", err)
		return 2
	}
	r := NewRunner(cfg)
	switch cmd {
	case "list":
		return cmdList(r, args)
	case "status":
		return cmdStatus(r, args)
	case "doctor":
		return cmdDoctor(r, args)
	case "repos":
		return cmdRepos(r, args)
	case "sync":
		return cmdSync(r, args)
	case "adopt":
		return cmdAdopt(r, args)
	case "clone":
		return cmdClone(r, args)
	case "exec":
		return cmdExec(r, args)
	case "use":
		return cmdUse(r, args)
	case "token":
		return cmdToken(r, args)
	case "tui":
		return cmdTUI(r, args)
	}
	fmt.Fprintf(os.Stderr, "ghm: unknown command %q\n\n%s", cmd, usageText)
	return 2
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, name+"=") {
			return strings.TrimPrefix(a, name+"=")
		}
	}
	return ""
}

func positional(args []string) []string {
	var out []string
	skip := false
	for i, a := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(a, "--") {
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				switch a {
				case "--account", "--jobs", "--dir":
					skip = true
				}
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func accountOrDie(r *Runner, login string) (Account, int) {
	a, ok := r.Cfg.Account(login)
	if !ok {
		fmt.Fprintf(os.Stderr, "ghm: unknown account %q\n", login)
		return Account{}, 2
	}
	return *a, 0
}

// ---------- init ----------

func cmdInit(args []string) int {
	ghPath := "/opt/homebrew/bin/gh"
	if _, err := os.Stat(ghPath); err != nil {
		if p, lerr := exec.LookPath("gh"); lerr == nil {
			ghPath = p
		}
	}
	cfg, err := LoadConfig()
	if err != nil {
		cfg = &Config{Version: 1, Root: defaultRoot, GhBinary: ghPath}
	}
	if cfg.GhBinary == "" {
		cfg.GhBinary = ghPath
	}
	r := NewRunner(cfg)

	logins, err := KeyringLogins(ghPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm init: cannot read gh accounts: %v\n", err)
		return 1
	}
	if len(logins) == 0 {
		fmt.Fprintln(os.Stderr, "ghm init: no gh accounts found, run 'gh auth login' first")
		return 1
	}
	sort.Strings(logins)

	for _, login := range logins {
		if _, ok := cfg.Account(login); ok {
			continue
		}
		u, uerr := r.UserInfo(login)
		if uerr != nil {
			fmt.Fprintf(os.Stderr, "ghm init: %v\n", uerr)
			return 1
		}
		name := strings.TrimSpace(u.Name)
		if name == "" {
			name = login
		}
		email := strings.TrimSpace(u.Email)
		if email == "" {
			email = fmt.Sprintf("%d+%s@users.noreply.github.com", u.ID, login)
		}
		cfg.Accounts = append(cfg.Accounts, Account{
			Login:     login,
			Name:      name,
			Email:     email,
			Key:       "~/.ssh/id_ed25519_" + login,
			HostAlias: "github-" + login,
			Dir:       filepath.Join(cfg.Root, login),
		})
	}
	cfg.applyDefaults()
	if _, ok := cfg.Account(cfg.DefaultAccount); !ok {
		cfg.DefaultAccount = cfg.Accounts[0].Login
	}
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "ghm init: save config: %v\n", err)
		return 1
	}
	fmt.Printf("config: %s (%d accounts)\n", ConfigPath(), len(cfg.Accounts))

	addedHosts, err := EnsureSSHConfig(cfg.Accounts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm init: ssh config: %v\n", err)
		return 1
	}
	for _, h := range addedHosts {
		fmt.Printf("ssh:    added Host %s\n", h)
	}

	helperChanged, err := ResetCredentialHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm init: credential helper: %v\n", err)
		return 1
	}
	if helperChanged {
		fmt.Println("git:    disabled gh https credential helper (use SSH)")
	}

	exit := 0
	for _, a := range cfg.Accounts {
		if created, kerr := EnsureKey(a); kerr != nil {
			fmt.Fprintf(os.Stderr, "ghm init: %v\n", kerr)
			exit = 1
			continue
		} else if created {
			fmt.Printf("key:    generated %s\n", a.Key)
		}
		has, kerr := r.KeyOnAccount(a.Login, a.Key+".pub")
		if kerr != nil {
			fmt.Printf("key:    %s: %v\n", a.Login, kerr)
			exit = 1
		} else if !has {
			if aerr := r.AddKeyToAccount(a.Login, a.Key+".pub"); aerr != nil {
				fmt.Fprintf(os.Stderr, "key:    %s: %v\n", a.Login, aerr)
				exit = 1
			} else {
				fmt.Printf("key:    registered public key on %s\n", a.Login)
			}
		}
		if changed, gerr := EnsureAccountGitConfig(a); gerr != nil {
			fmt.Fprintf(os.Stderr, "ghm init: %v\n", gerr)
			exit = 1
		} else if changed {
			fmt.Printf("git:    wrote %s\n", AccountGitConfigPath(a.Login))
		}
		if changed, ierr := EnsureIncludeIf(a); ierr != nil {
			fmt.Fprintf(os.Stderr, "ghm init: %v\n", ierr)
			exit = 1
		} else if changed {
			fmt.Printf("git:    includeIf %s/\n", a.Dir)
		}
	}
	fmt.Println("done. next: ghm doctor && ghm install-shim")
	return exit
}

// ---------- list / status ----------

type accountRow struct {
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	KeyOK   bool   `json:"key_ok"`
	TokenOK bool   `json:"token_ok"`
	SSHOK   *bool  `json:"ssh_ok,omitempty"`
	Repos   int    `json:"repos"`
	Dir     string `json:"dir"`
}

func buildRows(r *Runner, checkSSH bool, repos []Repo) []accountRow {
	rows := make([]accountRow, 0, len(r.Cfg.Accounts))
	counts := map[string]int{}
	for _, rp := range repos {
		if rp.Login != "" {
			counts[rp.Login]++
		}
	}
	for _, a := range r.Cfg.Accounts {
		row := accountRow{
			Login: a.Login, Name: a.Name, Email: a.Email,
			KeyOK: KeyExists(a), Repos: counts[a.Login], Dir: a.Dir,
		}
		_, err := r.Token(a.Login)
		row.TokenOK = err == nil
		if checkSSH {
			ok := false
			if _, serr := r.SSHAuth(a); serr == nil {
				ok = true
			}
			row.SSHOK = &ok
		}
		rows = append(rows, row)
	}
	return rows
}

func cmdList(r *Runner, args []string) int {
	repos := scanAll(r.Cfg, false)
	rows := buildRows(r, hasFlag(args, "--check"), repos)
	if hasFlag(args, "--json") {
		return printJSON(rows)
	}
	active := r.GhActiveLogin()
	fmt.Printf("gh active account: %s\n\n", orDash(active))
	withSSH := hasFlag(args, "--check")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if withSSH {
		fmt.Fprintln(tw, "LOGIN\tNAME\tEMAIL\tKEY\tTOKEN\tSSH\tREPOS\tDIR")
	} else {
		fmt.Fprintln(tw, "LOGIN\tNAME\tEMAIL\tKEY\tTOKEN\tREPOS\tDIR")
	}
	for _, row := range rows {
		if withSSH {
			ssh := "-"
			if row.SSHOK != nil {
				ssh = okMark(*row.SSHOK)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				row.Login, row.Name, row.Email, okMark(row.KeyOK), okMark(row.TokenOK), ssh, row.Repos, row.Dir)
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			row.Login, row.Name, row.Email, okMark(row.KeyOK), okMark(row.TokenOK), row.Repos, row.Dir)
	}
	tw.Flush()
	return 0
}

func cmdStatus(r *Runner, args []string) int {
	repos := scanAll(r.Cfg, true)
	rows := buildRows(r, false, repos)
	fmt.Println("accounts")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  LOGIN\tEMAIL\tKEY\tTOKEN\tREPOS")
	for _, row := range rows {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%d\n", row.Login, row.Email, okMark(row.KeyOK), okMark(row.TokenOK), row.Repos)
	}
	tw.Flush()

	var httpsCount, dirty, unmatched int
	perLogin := map[string]int{}
	for _, rp := range repos {
		if rp.Login == "" && rp.IsGitHub {
			httpsCount++
		}
		if rp.Dirty > 0 {
			dirty++
		}
		perLogin[orDash(rp.Login)]++
		if rp.Login != "" {
			if a, ok := r.Cfg.AccountFromAlias(rp.Login); ok && LocalEmail(rp.Path) != "" && LocalEmail(rp.Path) != a.Email {
				unmatched++
			}
		}
	}
	fmt.Printf("\nrepositories: %d (dirty: %d, https not migrated: %d, wrong local email: %d)\n",
		len(repos), dirty, httpsCount, unmatched)

	problems := 0
	if httpsCount > 0 {
		fmt.Printf("warn: %d github repos still use https; run 'ghm adopt --all'\n", httpsCount)
		problems++
	}
	if !CredentialHelperReset() {
		fmt.Println("warn: gh https credential helper is active; run 'ghm init'")
		problems++
	}
	for _, a := range r.Cfg.Accounts {
		if !KeyExists(a) {
			fmt.Printf("fail: missing key for %s\n", a.Login)
			problems++
		}
		if !IncludeIfRegistered(a) {
			fmt.Printf("warn: includeIf missing for %s\n", a.Login)
			problems++
		}
	}
	if !ShimInstalled() {
		fmt.Println("warn: gh shim not installed; 'gh pr/issue/api' still follows the global active account")
		problems++
	}
	if problems == 0 {
		fmt.Println("\nall good")
		return 0
	}
	return 1
}

func scanAll(cfg *Config, full bool) []Repo {
	seen := map[string]bool{}
	var out []Repo
	for _, group := range [][]string{accountRoots(cfg), cfg.ScanRootsResolved()} {
		for _, p := range ScanRepos(group, 6) {
			if seen[p] {
				continue
			}
			seen[p] = true
			if full {
				out = append(out, LoadRepoFull(p))
			} else {
				out = append(out, LoadRepo(p))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func accountRoots(cfg *Config) []string {
	var roots []string
	for _, a := range cfg.Accounts {
		roots = append(roots, a.RepoRoot())
	}
	return roots
}

// ---------- repos / sync ----------

func cmdRepos(r *Runner, args []string) int {
	filter := flagValue(args, "--account")
	repos := scanAll(r.Cfg, false)
	var rows []Repo
	for _, rp := range repos {
		if filter != "" && !strings.EqualFold(rp.Login, filter) {
			continue
		}
		rows = append(rows, LoadRepoFull(rp.Path))
	}
	if hasFlag(args, "--json") {
		return printJSON(rows)
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ACCOUNT\tBRANCH\tAHEAD\tBEHIND\tDIRTY\tPATH")
	for _, rp := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%s\n",
			orDash(rp.Login), orDash(rp.Branch), rp.Ahead, rp.Behind, rp.Dirty, TildePath(rp.Path))
	}
	tw.Flush()
	fmt.Printf("\n%d repositories\n", len(rows))
	return 0
}

func cmdSync(r *Runner, args []string) int {
	filter := flagValue(args, "--account")
	jobs := 4
	if v := flagValue(args, "--jobs"); v != "" {
		fmt.Sscanf(v, "%d", &jobs)
	}
	if jobs < 1 {
		jobs = 1
	}
	var targets []Repo
	for _, rp := range scanAll(r.Cfg, false) {
		if filter != "" && !strings.EqualFold(rp.Login, filter) {
			continue
		}
		targets = append(targets, rp)
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := 0
	for _, rp := range targets {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if _, err := Git(path, "fetch", "--prune", "--quiet"); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				fmt.Fprintf(os.Stderr, "fail %s: %v\n", TildePath(path), err)
			}
		}(rp.Path)
	}
	wg.Wait()
	fmt.Printf("fetched %d repositories (%d failed)\n", len(targets), failed)
	if failed > 0 {
		return 1
	}
	return 0
}

// ---------- adopt / clone ----------

func cmdAdopt(r *Runner, args []string) int {
	override := flagValue(args, "--account")
	move := hasFlag(args, "--move")
	dry := hasFlag(args, "--dry-run")
	paths := positional(args)
	if hasFlag(args, "--all") || len(paths) == 0 {
		var found []string
		for _, rp := range scanAll(r.Cfg, false) {
			found = append(found, rp.Path)
		}
		paths = found
	}
	exit := 0
	for _, p := range paths {
		if err := adoptOne(r, p, override, move, dry); err != nil {
			fmt.Fprintf(os.Stderr, "adopt %s: %v\n", TildePath(p), err)
			exit = 1
		}
	}
	return exit
}

func adoptOne(r *Runner, path, override string, move, dry bool) error {
	path = ExpandPath(path)
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return fmt.Errorf("not a git repository")
	}
	origin, err := Git(path, "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("no origin remote")
	}
	login, slug, isAlias := ParseSSHAlias(origin)
	if !isAlias {
		s, ok := ParseHTTPSGitHub(origin)
		if !ok {
			return fmt.Errorf("not a github.com remote (%s)", origin)
		}
		slug = s
	}
	var acct Account
	if isAlias && override == "" {
		if a, ok := r.Cfg.AccountFromAlias(login); ok {
			acct = a
		}
	}
	if acct.Login == "" {
		res, rerr := r.ResolveSlug(slug, override)
		if rerr != nil {
			return rerr
		}
		a, _ := r.Cfg.AccountFromAlias(res.Login)
		acct = a
		fmt.Printf("%-14s %s (%s)\n", acct.Login, slug, res.Mode)
	}
	newURL := RemoteURLFor(acct, slug)
	if dry {
		fmt.Printf("would set %s -> %s\n", TildePath(path), newURL)
		return nil
	}
	if origin != newURL {
		if err := SetOrigin(path, newURL); err != nil {
			return err
		}
	}
	if err := ApplyIdentity(path, acct); err != nil {
		return err
	}
	fmt.Printf("%-14s %s\n", acct.Login, TildePath(path))
	if !move {
		return nil
	}
	target := filepath.Join(acct.RepoRoot(), filepath.Base(path))
	if filepath.Clean(target) == filepath.Clean(path) {
		return nil
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("move target already exists: %s", TildePath(target))
	}
	if err := os.MkdirAll(acct.RepoRoot(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(path, target); err != nil {
		return fmt.Errorf("move: %w", err)
	}
	fmt.Printf("%-14s moved %s -> %s\n", "", TildePath(path), TildePath(target))
	return nil
}

func cmdClone(r *Runner, args []string) int {
	pos := positional(args)
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ghm clone <owner/repo> [--account X]")
		return 2
	}
	slug := strings.TrimSuffix(strings.TrimPrefix(pos[0], "https://github.com/"), ".git")
	slug = strings.TrimSuffix(slug, "/")
	res, err := r.ResolveSlug(slug, flagValue(args, "--account"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm clone: %v\n", err)
		return 1
	}
	acct, _ := r.Cfg.AccountFromAlias(res.Login)
	dest := filepath.Join(acct.RepoRoot(), filepath.Base(slug))
	if v := flagValue(args, "--dir"); v != "" {
		dest = ExpandPath(v)
	}
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(os.Stderr, "ghm clone: destination exists: %s\n", TildePath(dest))
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ghm clone: %v\n", err)
		return 1
	}
	url := RemoteURLFor(acct, slug)
	fmt.Printf("%-14s %s (%s)\n", acct.Login, slug, res.Mode)
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return 1
	}
	if err := ApplyIdentity(dest, acct); err != nil {
		fmt.Fprintf(os.Stderr, "ghm clone: applied checkout but identity failed: %v\n", err)
		return 1
	}
	fmt.Printf("cloned -> %s\n", TildePath(dest))
	return 0
}

// ---------- exec / use / token ----------

func envForAccount(r *Runner, a Account) ([]string, error) {
	tok, err := r.Token(a.Login)
	if err != nil {
		return nil, err
	}
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"GH_TOKEN="+tok,
		"GH_HOST="+githubHost,
		"GIT_SSH_COMMAND="+a.SSHCommand(),
		"GIT_AUTHOR_NAME="+a.Name,
		"GIT_AUTHOR_EMAIL="+a.Email,
		"GIT_COMMITTER_NAME="+a.Name,
		"GIT_COMMITTER_EMAIL="+a.Email,
	)
	return env, nil
}

func cmdExec(r *Runner, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: ghm exec <login> -- <command...>")
		return 2
	}
	a, code := accountOrDie(r, args[0])
	if code != 0 {
		return code
	}
	rest := args[1:]
	if rest[0] == "--" {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "ghm exec: missing command")
		return 2
	}
	env, err := envForAccount(r, a)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm exec: %v\n", err)
		return 1
	}
	cmd := exec.Command(rest[0], rest[1:]...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "ghm exec: %v\n", err)
		return 1
	}
	return 0
}

func cmdUse(r *Runner, args []string) int {
	pos := positional(args)
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: eval \"$(ghm use <login>)\"")
		return 2
	}
	a, code := accountOrDie(r, pos[0])
	if code != 0 {
		return code
	}
	tok, err := r.Token(a.Login)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm use: %v\n", err)
		return 1
	}
	fmt.Printf("export GH_TOKEN=%q\n", tok)
	fmt.Printf("export GIT_SSH_COMMAND=%q\n", a.SSHCommand())
	fmt.Printf("export GIT_AUTHOR_NAME=%q\n", a.Name)
	fmt.Printf("export GIT_AUTHOR_EMAIL=%q\n", a.Email)
	fmt.Printf("export GIT_COMMITTER_NAME=%q\n", a.Name)
	fmt.Printf("export GIT_COMMITTER_EMAIL=%q\n", a.Email)
	return 0
}

func cmdToken(r *Runner, args []string) int {
	pos := positional(args)
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ghm token <login>")
		return 2
	}
	a, code := accountOrDie(r, pos[0])
	if code != 0 {
		return code
	}
	tok, err := r.Token(a.Login)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm token: %v\n", err)
		return 1
	}
	fmt.Println(tok)
	return 0
}

// ---------- shim ----------

// RepoLoginForCWD resolves the account for the current directory.
func RepoLoginForCWD(r *Runner, dir string) string {
	root := findRepoRoot(dir)
	if root != "" {
		if origin, err := Git(root, "remote", "get-url", "origin"); err == nil {
			if login, _, ok := ParseSSHAlias(origin); ok {
				if _, ok := r.Cfg.Account(login); ok {
					return login
				}
			}
		}
	}
	if a, ok := r.Cfg.AccountForDir(dir); ok {
		return a.Login
	}
	return ""
}

func findRepoRoot(dir string) string {
	cur := ExpandPath(dir)
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

func cmdShimToken(args []string) int {
	if os.Getenv("GHM_SHIM") == "0" {
		return 0
	}
	cfg, err := LoadConfig()
	if err != nil {
		return 0
	}
	r := NewRunner(cfg)
	dir, err := os.Getwd()
	if err != nil {
		return 0
	}
	login := RepoLoginForCWD(r, dir)
	if login == "" {
		return 0
	}
	tok, err := r.Token(login)
	if err != nil {
		return 0
	}
	fmt.Println(tok)
	return 0
}

func cmdTUI(r *Runner, args []string) int {
	if err := RunTUI(r); err != nil {
		fmt.Fprintf(os.Stderr, "ghm tui: %v\n", err)
		return 1
	}
	return 0
}

func cmdInstallShim(args []string) int {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm: %v\n", err)
		return 2
	}
	msg, err := InstallShim(NewRunner(cfg))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghm install-shim: %v\n", err)
		return 1
	}
	fmt.Println(msg)
	fmt.Println("verify with: type -a gh   (must show ~/.local/bin/gh first)")
	return 0
}

func cmdUninstallShim(args []string) int {
	if err := UninstallShim(); err != nil {
		fmt.Fprintf(os.Stderr, "ghm uninstall-shim: %v\n", err)
		return 1
	}
	fmt.Println("shim removed")
	return 0
}

// ---------- helpers ----------

func okMark(v bool) string {
	if v {
		return "ok"
	}
	return "MISSING"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func printJSON(v interface{}) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "ghm: %v\n", err)
		return 1
	}
	return 0
}
