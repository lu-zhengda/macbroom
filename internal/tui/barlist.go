package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

// renderBarList renders SpaceLens nodes as a vertical bar list (ncdu/dust style).
// Each item gets one line: filled bar + name + size.
// cursor is the selected index, scrollOffset controls which items are visible.
func renderBarList(nodes []scanner.SpaceLensNode, width, height, cursor, scrollOffset int) string {
	if len(nodes) == 0 {
		return "  Empty directory.\n"
	}

	// Find max size for bar scaling.
	var maxSize int64
	for _, n := range nodes {
		if n.Size > maxSize {
			maxSize = n.Size
		}
	}
	if maxSize == 0 {
		maxSize = 1
	}

	// Calculate name column width from visible items.
	nameWidth := 0
	for _, n := range nodes {
		name := n.Name
		if n.IsDir {
			name += "/"
		}
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}
	if nameWidth > 30 {
		nameWidth = 30
	}

	// Layout: "  <bar>  <name>  <size>"
	// Reserve: 2 (prefix) + 2 (gap) + 2 (gap) + 10 (size) = 16
	barWidth := width - nameWidth - 16
	if barWidth < 10 {
		barWidth = 10
	}

	var sb strings.Builder

	end := scrollOffset + height
	if end > len(nodes) {
		end = len(nodes)
	}

	for i := scrollOffset; i < end; i++ {
		n := nodes[i]
		ratio := float64(n.Size) / float64(maxSize)
		filled := int(ratio * float64(barWidth))
		if filled < 1 && n.Size > 0 {
			filled = 1
		}
		empty := barWidth - filled

		// Color by ratio relative to largest item.
		var barColor lipgloss.Color
		pct := ratio * 100
		if pct >= 75 {
			barColor = lipgloss.Color("196") // red
		} else if pct >= 40 {
			barColor = lipgloss.Color("214") // orange
		} else {
			barColor = lipgloss.Color("82") // green
		}

		barStyle := lipgloss.NewStyle().Foreground(barColor)
		bar := barStyle.Render(strings.Repeat("\u2588", filled)) +
			dimStyle.Render(strings.Repeat("\u2591", empty))

		name := n.Name
		if n.IsDir {
			name += "/"
		}
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "\u2026"
		}

		sizeStr := fmt.Sprintf("%10s", utils.FormatSize(n.Size))

		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s  %-*s %s", prefix, bar, nameWidth, name, sizeStr)

		if i == cursor {
			sb.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	// Scroll indicator.
	if len(nodes) > height {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  [%d-%d of %d]", scrollOffset+1, end, len(nodes))) + "\n")
	}

	return sb.String()
}
