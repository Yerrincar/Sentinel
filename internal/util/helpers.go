package helpers

import "strings"

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
