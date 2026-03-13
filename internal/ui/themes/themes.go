package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Palette struct {
	Name            string
	Focus           string
	Border          string
	Selected        string
	TitleServices   string
	TitleWorkspace  string
	TitleTypes      string
	TitleFilters    string
	TitleThemes     string
	TitleAddService string
	TitleRemote     string
	StateRunning    string
	StateDegraded   string
	StateStopped    string
	StateInactive   string
	StateError      string
}

var palettes = []Palette{
	{
		Name:            "Sentinel Blue",
		Focus:           "#4F8CFF",
		Border:          "#60708A",
		Selected:        "#7AA2F7",
		TitleServices:   "#8AB4FF",
		TitleWorkspace:  "#B0C4FF",
		TitleTypes:      "#9AD0FF",
		TitleFilters:    "#90E0EF",
		TitleThemes:     "#A0C4FF",
		TitleAddService: "#FFD6A5",
		TitleRemote:     "#CDB4DB",
		StateRunning:    "#22C55E",
		StateDegraded:   "#F59E0B",
		StateStopped:    "#EF4444",
		StateInactive:   "#94A3B8",
		StateError:      "#DC2626",
	},
	{
		Name:            "Nordic",
		Focus:           "#88C0D0",
		Border:          "#81A1C1",
		Selected:        "#5E81AC",
		TitleServices:   "#88C0D0",
		TitleWorkspace:  "#8FBCBB",
		TitleTypes:      "#81A1C1",
		TitleFilters:    "#A3BE8C",
		TitleThemes:     "#B48EAD",
		TitleAddService: "#EBCB8B",
		TitleRemote:     "#D08770",
		StateRunning:    "#A3BE8C",
		StateDegraded:   "#EBCB8B",
		StateStopped:    "#BF616A",
		StateInactive:   "#4C566A",
		StateError:      "#BF616A",
	},
	{
		Name:            "Gruvbox",
		Focus:           "#FABD2F",
		Border:          "#928374",
		Selected:        "#83A598",
		TitleServices:   "#FABD2F",
		TitleWorkspace:  "#B8BB26",
		TitleTypes:      "#83A598",
		TitleFilters:    "#8EC07C",
		TitleThemes:     "#D3869B",
		TitleAddService: "#FE8019",
		TitleRemote:     "#FB4934",
		StateRunning:    "#8EC07C",
		StateDegraded:   "#FABD2F",
		StateStopped:    "#FB4934",
		StateInactive:   "#928374",
		StateError:      "#CC241D",
	},
	{
		Name:            "Dracula",
		Focus:           "#BD93F9",
		Border:          "#6272A4",
		Selected:        "#FF79C6",
		TitleServices:   "#BD93F9",
		TitleWorkspace:  "#8BE9FD",
		TitleTypes:      "#50FA7B",
		TitleFilters:    "#F1FA8C",
		TitleThemes:     "#FF79C6",
		TitleAddService: "#FFB86C",
		TitleRemote:     "#FF5555",
		StateRunning:    "#50FA7B",
		StateDegraded:   "#F1FA8C",
		StateStopped:    "#FF5555",
		StateInactive:   "#6272A4",
		StateError:      "#FF5555",
	},
	{
		Name:            "Emerald",
		Focus:           "#34D399",
		Border:          "#4B5563",
		Selected:        "#10B981",
		TitleServices:   "#34D399",
		TitleWorkspace:  "#6EE7B7",
		TitleTypes:      "#A7F3D0",
		TitleFilters:    "#FDE68A",
		TitleThemes:     "#93C5FD",
		TitleAddService: "#FCA5A5",
		TitleRemote:     "#C4B5FD",
		StateRunning:    "#22C55E",
		StateDegraded:   "#EAB308",
		StateStopped:    "#F43F5E",
		StateInactive:   "#9CA3AF",
		StateError:      "#DC2626",
	},
}

func All() []Palette {
	cp := make([]Palette, len(palettes))
	copy(cp, palettes)
	return cp
}

func Default() Palette {
	return palettes[0]
}

func ByName(name string) (Palette, bool) {
	for _, p := range palettes {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Palette{}, false
}

type savedTheme struct {
	Name string `json:"name"`
}

func SaveSelected(name string) error {
	if _, ok := ByName(name); !ok {
		return fmt.Errorf("unknown theme: %s", name)
	}
	path, err := themeConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(savedTheme{Name: name})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadSelected() (Palette, error) {
	path, err := themeConfigPath()
	if err != nil {
		return Palette{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Palette{}, err
	}
	var s savedTheme
	if err := json.Unmarshal(data, &s); err != nil {
		return Palette{}, err
	}
	p, ok := ByName(s.Name)
	if !ok {
		return Palette{}, fmt.Errorf("theme not found: %s", s.Name)
	}
	return p, nil
}

func themeConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "sentinel", "theme.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sentinel", "theme.json"), nil
}
