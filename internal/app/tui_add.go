package app

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// addState drives the "add account" wizard inside the dashboard.
type addState struct {
	step   string // menu | token | busy | result
	input  string
	lines  []string
	failed bool
}

type loginDoneMsg struct{ err error }
type addDoneMsg struct {
	logins []string
	lines  []string
	err    error
}

func (m *tuiModel) openAdd() {
	m.add = &addState{step: "menu"}
}

func (m *tuiModel) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	a := m.add
	switch a.step {
	case "menu":
		switch msg.String() {
		case "esc", "q":
			m.add = nil
		case "ctrl+c":
			return m, tea.Quit
		case "b":
			a.step = "busy"
			a.lines = []string{"starting gh auth login (browser)…"}
			cmd := exec.Command(m.r.GhPath(), "auth", "login", "-h", "github.com", "-p", "https", "-w")
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return loginDoneMsg{err: err} })
		case "t":
			a.step = "token"
			a.input = ""
		}
	case "token":
		switch msg.String() {
		case "esc":
			m.add = nil
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			tok := strings.TrimSpace(a.input)
			a.input = ""
			if tok == "" {
				return m, nil
			}
			a.step = "busy"
			a.lines = []string{"saving token with gh auth login --with-token…"}
			return m, tokenLoginCmd(m.r, tok)
		case "backspace":
			if len(a.input) > 0 {
				a.input = a.input[:len(a.input)-1]
			}
		default:
			if len(msg.Runes) > 0 {
				a.input += string(msg.Runes)
			}
		}
	case "busy":
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case "result":
		switch msg.String() {
		case "esc", "enter", "q":
			m.add = nil
			m.sticky = false
			m.status = fmt.Sprintf("%d repositories", len(m.visible))
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// tokenLoginCmd stores a PAT in the gh keyring, then configures the account.
func tokenLoginCmd(r *Runner, token string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(r.GhPath(), "auth", "login", "--with-token", "--hostname", "github.com")
		cmd.Stdin = strings.NewReader(token + "\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := firstLine(strings.TrimSpace(string(out)))
			if msg == "" {
				msg = err.Error()
			}
			return addDoneMsg{err: fmt.Errorf("%s", msg)}
		}
		return discoverAdd(r)
	}
}

// discoverAdd configures logins that are in the gh keyring but not yet in ghm.
func discoverAdd(r *Runner) tea.Msg {
	after, err := KeyringLogins(r.GhPath())
	if err != nil {
		return addDoneMsg{err: err}
	}
	var before []string
	for _, a := range r.Cfg.Accounts {
		before = append(before, a.Login)
	}
	pending := newLogins(before, after)
	if len(pending) == 0 {
		return addDoneMsg{err: fmt.Errorf("no new account found in the gh keyring; it may already be configured (press r to rescan)")}
	}
	var lines []string
	for _, login := range pending {
		l, aerr := AddAccount(r, login)
		lines = append(lines, l...)
		if aerr != nil {
			return addDoneMsg{logins: pending, lines: lines, err: aerr}
		}
	}
	return addDoneMsg{logins: pending, lines: lines}
}

// addDiscoverCmd resumes after an interactive login and configures the new account.
func addDiscoverCmd(r *Runner) tea.Cmd {
	return func() tea.Msg { return discoverAdd(r) }
}

func (m *tuiModel) viewAdd() string {
	a := m.add
	var b strings.Builder
	switch a.step {
	case "menu":
		b.WriteString(styleHeader.Render("add github account") + "\n")
		b.WriteString(styleDim.Render("credentials stay in the gh keyring; ghm writes key + git config") + "\n\n")
		b.WriteString("  b   login with browser   (gh auth login, recommended)\n")
		b.WriteString("  t   paste a token        (scopes: repo, read:org, admin:public_key)\n\n")
		b.WriteString(styleDim.Render("  the browser flow asks: protocol https, and \"Authenticate Git with your GitHub credentials?\" -> No") + "\n")
		m.writeFooter(&b, "b browser  t token  esc cancel")

	case "token":
		b.WriteString(styleHeader.Render("add github account · token") + "\n\n")
		b.WriteString("  " + strings.Repeat("•", len(a.input)) + "█\n\n")
		b.WriteString(styleDim.Render("  scopes: repo, read:org, admin:public_key") + "\n")
		b.WriteString(styleDim.Render("  the token is piped to gh auth login --with-token and never written by ghm") + "\n")
		m.writeFooter(&b, "enter submit  esc cancel")

	case "busy":
		b.WriteString(styleHeader.Render("add github account") + "\n\n")
		for _, l := range a.lines {
			b.WriteString("  " + l + "\n")
		}
		m.writeFooter(&b, "please wait")

	case "result":
		title := "add github account · done"
		if a.failed {
			title = "add github account · failed"
		}
		b.WriteString(styleHeader.Render(title) + "\n\n")
		for _, l := range a.lines {
			line := "  " + l
			switch {
			case strings.HasPrefix(l, "warn:"):
				b.WriteString(styleWarn.Render(line) + "\n")
			case strings.HasPrefix(l, "error:"), strings.HasPrefix(l, "next:"):
				b.WriteString(styleDim.Render(line) + "\n")
			default:
				b.WriteString(line + "\n")
			}
		}
		if !a.failed {
			b.WriteString("\n" + styleOk.Render("  account ready; open a new tab with tab, then ghm doctor") + "\n")
		}
		m.writeFooter(&b, "esc close  r rescan")
	}
	return b.String()
}
