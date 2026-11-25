package app

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reposMsg []Repo
type ghActiveMsg string
type actionMsg struct {
	desc string
	err  error
}
type reloadMsg struct{}

type tuiModel struct {
	r        *Runner
	all      []Repo
	visible  []Repo
	accounts []Account
	acct     int
	cursor   int
	filter   string
	filterOn bool
	help     bool
	status   string
	err      bool
	loading  bool
	width    int
	height   int
	active   string
}

var (
	styleHeader  = lipgloss.NewStyle().Bold(true)
	styleDim     = lipgloss.NewStyle().Faint(true)
	styleTab     = lipgloss.NewStyle().Padding(0, 1)
	styleTabOn   = lipgloss.NewStyle().Padding(0, 1).Reverse(true).Bold(true)
	styleSel     = lipgloss.NewStyle().Reverse(true)
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleOk      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleRule    = lipgloss.NewStyle().Faint(true)
	styleHelpBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

// RunTUI starts the dashboard.
func RunTUI(r *Runner) error {
	m := &tuiModel{r: r, accounts: r.Cfg.Accounts, active: r.GhActiveLogin()}
	if len(m.accounts) > 0 {
		if a, ok := r.Cfg.AccountForDir("."); ok {
			for i := range m.accounts {
				if m.accounts[i].Login == a.Login {
					m.acct = i
				}
			}
		}
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), func() tea.Msg { return reloadMsg{} })
}

func (m *tuiModel) loadCmd() tea.Cmd {
	return func() tea.Msg { return reposMsg(scanAll(m.r.Cfg, true)) }
}

func (m *tuiModel) applyFilter() {
	m.visible = m.visible[:0]
	login := ""
	if m.acct < len(m.accounts) {
		login = m.accounts[m.acct].Login
	}
	q := strings.ToLower(m.filter)
	for _, rp := range m.all {
		if login != "" && !strings.EqualFold(rp.Login, login) {
			continue
		}
		if q != "" {
			hay := strings.ToLower(rp.Slug + " " + rp.Path + " " + rp.Branch)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		m.visible = append(m.visible, rp)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *tuiModel) selected() (Repo, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return Repo{}, false
	}
	return m.visible[m.cursor], true
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case reposMsg:
		m.all = msg
		m.loading = false
		m.applyFilter()
		m.status = fmt.Sprintf("%d repositories", len(m.all))

	case ghActiveMsg:
		m.active = string(msg)

	case reloadMsg:
		m.loading = true
		return m, m.loadCmd()

	case actionMsg:
		m.loading = false
		if msg.err != nil {
			m.err = true
			m.status = msg.desc + ": " + firstLine(msg.err.Error())
		} else {
			m.err = false
			m.status = msg.desc
		}
		return m, m.loadCmd()

	case tea.KeyMsg:
		if m.filterOn {
			return m.updateFilter(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.visible) > 0 {
				m.cursor = len(m.visible) - 1
			}
		case "tab", "right", "l":
			if len(m.accounts) > 0 {
				m.acct = (m.acct + 1) % len(m.accounts)
				m.cursor = 0
				m.applyFilter()
			}
		case "shift+tab", "left", "h":
			if len(m.accounts) > 0 {
				m.acct = (m.acct - 1 + len(m.accounts)) % len(m.accounts)
				m.cursor = 0
				m.applyFilter()
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.accounts) {
				m.acct = idx
				m.cursor = 0
				m.applyFilter()
			}
		case "/":
			m.filterOn = true
			m.filter = ""
			m.status = "filter: "
		case "esc":
			m.filter = ""
			m.applyFilter()
		case "?":
			m.help = !m.help
		case "r":
			return m, m.loadCmd()
		case "f":
			if rp, ok := m.selected(); ok {
				m.loading = true
				return m, gitAction(rp, "fetch", "fetch", "--prune", "--quiet")
			}
		case "p":
			if rp, ok := m.selected(); ok {
				m.loading = true
				return m, gitAction(rp, "pull "+rp.Slug, "pull", "--ff-only")
			}
		case "P":
			if rp, ok := m.selected(); ok {
				m.loading = true
				return m, gitAction(rp, "push "+rp.Slug, "push")
			}
		case "o":
			if rp, ok := m.selected(); ok {
				if rp.Slug == "" {
					m.err = true
					m.status = "no github remote"
					break
				}
				url := "https://github.com/" + rp.Slug
				if err := exec.Command("open", url).Start(); err != nil {
					m.err = true
					m.status = err.Error()
				} else {
					m.err = false
					m.status = "opened " + url
				}
			}
		}
	}
	return m, nil
}

func (m *tuiModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.filterOn = false
		if msg.String() == "esc" {
			m.filter = ""
		}
		m.applyFilter()
		m.status = fmt.Sprintf("%d repositories", len(m.visible))
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
		m.applyFilter()
		m.status = "filter: " + m.filter
	case "ctrl+c":
		return m, tea.Quit
	default:
		if len(msg.Runes) > 0 {
			m.filter += string(msg.Runes)
			m.applyFilter()
			m.status = "filter: " + m.filter
		}
	}
	return m, nil
}

func gitAction(rp Repo, desc string, args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := Git(rp.Path, args...)
		_ = out
		return actionMsg{desc: desc, err: err}
	}
}

func (m *tuiModel) View() string {
	if m.width == 0 {
		return "loading..."
	}
	var b strings.Builder
	head := styleHeader.Render("ghm " + appVersion)
	right := styleDim.Render("gh active: " + orDash(m.active))
	gap := m.width - lipgloss.Width(head) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	b.WriteString(head + strings.Repeat(" ", gap) + right + "\n")

	for i, a := range m.accounts {
		label := fmt.Sprintf("%d %s %d", i+1, a.Login, m.countFor(a.Login))
		if i == m.acct {
			b.WriteString(styleTabOn.Render(label))
		} else {
			b.WriteString(styleTab.Render(label))
		}
	}
	b.WriteString("\n")
	b.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	if m.help {
		b.WriteString(m.helpView())
		return b.String()
	}

	listHeight := m.height - 7
	if listHeight < 3 {
		listHeight = 3
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	if start+listHeight > len(m.visible) {
		start = len(m.visible) - listHeight
	}
	if start < 0 {
		start = 0
	}
	widths := m.columns()
	fmt.Fprintf(&b, "%s  %s  %s  %s  %s\n",
		pad("REPO", widths[0]), pad("BRANCH", widths[1]), pad("AHEAD", 5), pad("BEHIND", 6), "STATE")
	for i := start; i < start+listHeight && i < len(m.visible); i++ {
		rp := m.visible[i]
		state := "clean"
		switch {
		case rp.Dirty > 0:
			state = fmt.Sprintf("%d dirty", rp.Dirty)
		case rp.Behind > 0:
			state = "behind"
		}
		line := fmt.Sprintf("%s  %s  %s  %s  %s",
			pad(repoLabel(rp, widths[0]), widths[0]),
			pad(orDash(rp.Branch), widths[1]),
			pad(fmt.Sprint(rp.Ahead), 5),
			pad(fmt.Sprint(rp.Behind), 6),
			state)
		if i == m.cursor {
			b.WriteString(styleSel.Render(pad(line, m.width-1)) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	if len(m.visible) == 0 {
		b.WriteString(styleDim.Render("  no repositories for this account") + "\n")
	}

	b.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")
	status := m.status
	if m.loading {
		status = "working…  " + status
	}
	if m.err {
		b.WriteString(styleErr.Render(status) + "\n")
	} else {
		b.WriteString(styleOk.Render(status) + "\n")
	}
	b.WriteString(styleDim.Render("j/k move  f fetch  p pull  P push  o open  / filter  1-3 account  ? help  q quit"))
	return b.String()
}

func (m *tuiModel) helpView() string {
	help := strings.Join([]string{
		"j / k          move selection",
		"g / G          first / last",
		"/              filter repos (enter to apply, esc to clear)",
		"1..9, tab      switch account",
		"f              git fetch --prune",
		"p              git pull --ff-only",
		"P              git push",
		"o              open repo on github.com",
		"r              rescan repositories",
		"?              close help",
		"q              quit",
	}, "\n")
	return styleHelpBox.Render(help) + "\n"
}

func (m *tuiModel) countFor(login string) int {
	n := 0
	for _, rp := range m.all {
		if strings.EqualFold(rp.Login, login) {
			n++
		}
	}
	return n
}

func (m *tuiModel) columns() [2]int {
	w := m.width
	name := w / 2
	if name < 18 {
		name = 18
	}
	if name > 48 {
		name = 48
	}
	branch := w - name - 5 - 6 - 12
	if branch < 10 {
		branch = 10
	}
	return [2]int{name, branch}
}

func repoLabel(rp Repo, width int) string {
	label := rp.Slug
	if label == "" {
		label = TildePath(rp.Path)
	}
	if rp.Login == "" {
		label = TildePath(rp.Path)
	}
	if len(label) > width {
		label = "…" + label[len(label)-width+1:]
	}
	return label
}

func pad(s string, width int) string {
	n := width - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

// helper so TUI can surface warnings without breaking layout
func (m *tuiModel) warn(msg string) { m.status = msg }
