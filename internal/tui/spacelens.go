package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

// SpaceLensModel is the standalone Space Lens TUI (used by `spacelens -i`).
type SpaceLensModel struct {
	path         string
	nodes        []scanner.SpaceLensNode
	cursor       int
	scrollOffset int
	loading      bool
	width        int
	height       int
}

func NewSpaceLensModel(path string) SpaceLensModel {
	return SpaceLensModel{path: path, loading: true}
}

func (m SpaceLensModel) Init() tea.Cmd {
	return m.doAnalyze()
}

func (m SpaceLensModel) doAnalyze() tea.Cmd {
	path := m.path
	return func() tea.Msg {
		sl := scanner.NewSpaceLens(path, 1)
		nodes, _ := sl.Analyze(context.Background())
		return spaceLensDoneMsg{nodes: nodes, path: path}
	}
}

func (m SpaceLensModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spaceLensDoneMsg:
		m.loading = false
		m.nodes = msg.nodes
		m.path = msg.path

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down", "j":
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "enter", "right", "l":
			if m.cursor < len(m.nodes) && m.nodes[m.cursor].IsDir {
				m.path = m.nodes[m.cursor].Path
				m.loading = true
				m.cursor = 0
				m.scrollOffset = 0
				return m, m.doAnalyze()
			}
		case "left", "backspace", "h":
			if idx := lastSlash(m.path); idx > 0 {
				m.path = m.path[:idx]
				m.loading = true
				m.cursor = 0
				m.scrollOffset = 0
				return m, m.doAnalyze()
			}
		}
	}
	return m, nil
}

func (m *SpaceLensModel) ensureVisible() {
	visible := m.visibleLines()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
}

func (m SpaceLensModel) visibleLines() int {
	visible := m.height - 7
	if visible < 4 {
		visible = 4
	}
	return visible
}

func (m SpaceLensModel) View() string {
	s := renderHeader("Space Lens")

	if m.loading {
		s += dimStyle.Render(m.path) + "\n\n"
		s += "Analyzing...\n"
		return s
	}

	var totalSize int64
	for _, node := range m.nodes {
		totalSize += node.Size
	}
	s += dimStyle.Render(fmt.Sprintf("%s (%s)", m.path, utils.FormatSize(totalSize))) + "\n\n"

	if len(m.nodes) == 0 {
		s += "Empty directory.\n"
		return s + renderFooter("q quit")
	}

	visible := m.visibleLines()
	s += renderBarList(m.nodes, m.width, visible, m.cursor, m.scrollOffset)

	s += renderFooter("arrows navigate | enter/right drill in | left/h go up | q quit")
	return s
}
