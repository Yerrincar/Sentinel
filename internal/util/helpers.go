package helpers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func BorderTitle(box, title string) string {
	if title == "" {
		return box
	}

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	top := []rune(lines[0])
	if len(top) < 3 {
		return box
	}

	label := []rune(" " + title + " ")
	start := 3
	maxEnd := len(top) - 1
	if start >= maxEnd {
		return box
	}
	if start+len(label) > maxEnd {
		label = label[:maxEnd-start]
	}
	copy(top[start:start+len(label)], label)
	lines[0] = string(top)

	return strings.Join(lines, "\n")
}

func ColorPanelBorder(panel string, color lipgloss.TerminalColor) string {
	borderRunes := map[rune]struct{}{
		'┌': {}, '┐': {}, '└': {}, '┘': {}, '─': {}, '│': {},
		'├': {}, '┤': {}, '┬': {}, '┴': {}, '┼': {},
		'+': {}, '-': {}, '|': {},
	}
	borderStyle := lipgloss.NewStyle().Foreground(color)

	lines := strings.Split(panel, "\n")
	for i, line := range lines {
		var b strings.Builder
		for _, r := range line {
			if _, ok := borderRunes[r]; ok {
				b.WriteString(borderStyle.Render(string(r)))
				continue
			}
			b.WriteRune(r)
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

func ColorOuterPanelBorder(panel string, color lipgloss.TerminalColor) string {
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		return panel
	}

	borderRunes := map[rune]struct{}{
		'┌': {}, '┐': {}, '└': {}, '┘': {}, '─': {}, '│': {},
		'├': {}, '┤': {}, '┬': {}, '┴': {}, '┼': {},
		'+': {}, '-': {}, '|': {},
	}
	borderStyle := lipgloss.NewStyle().Foreground(color)

	colorBorderRunes := func(line string) string {
		var b strings.Builder
		for _, r := range line {
			if _, ok := borderRunes[r]; ok {
				b.WriteString(borderStyle.Render(string(r)))
				continue
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	colorSideBorders := func(line string) string {
		rs := []rune(line)
		if len(rs) == 0 {
			return line
		}
		firstIdx := -1
		lastIdx := -1
		for i, r := range rs {
			if _, ok := borderRunes[r]; ok {
				firstIdx = i
				break
			}
		}
		for i := len(rs) - 1; i >= 0; i-- {
			if _, ok := borderRunes[rs[i]]; ok {
				lastIdx = i
				break
			}
		}
		if firstIdx == -1 {
			return line
		}
		if lastIdx == -1 {
			lastIdx = firstIdx
		}
		if firstIdx == lastIdx {
			return string(rs[:firstIdx]) + borderStyle.Render(string(rs[firstIdx])) + string(rs[firstIdx+1:])
		}

		return string(rs[:firstIdx]) +
			borderStyle.Render(string(rs[firstIdx])) +
			string(rs[firstIdx+1:lastIdx]) +
			borderStyle.Render(string(rs[lastIdx])) +
			string(rs[lastIdx+1:])
	}

	for i := range lines {
		if i == 0 || i == len(lines)-1 {
			lines[i] = colorBorderRunes(lines[i])
			continue
		}
		lines[i] = colorSideBorders(lines[i])
	}

	return strings.Join(lines, "\n")
}

func FormatUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	totalSeconds := int64(d.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	parts := make([]string, 0, 4)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if days > 0 || hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if days > 0 || hours > 0 || minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))

	return strings.Join(parts, " ")
}

func CPUPercent(prevUsageUsec, currUsageUsec uint64, prevAt, now time.Time) float64 {
	dt := now.Sub(prevAt).Microseconds()
	du := int64(currUsageUsec) - int64(prevUsageUsec)
	if dt <= 0 || du < 0 {
		return 0.0
	}
	return (float64(du) / float64(dt)) * 100.0
}

func FormatBytes(bytes uint64) string {
	if (bytes / (1024 * 1024)) < 1024 {
		return strconv.FormatFloat(float64(bytes)/(1024.0*1024.0), 'f', 1, 64) + " MiB"
	}
	return strconv.FormatFloat(float64(bytes)/(1024.0*1024.0*1024.0), 'f', 1, 64) + " GiB"
}
