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
	TitleKubeconfig string
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
		TitleKubeconfig: "#FBCFE8",
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
		TitleKubeconfig: "#B48EAD",
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
		TitleKubeconfig: "#B16286",
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
		TitleKubeconfig: "#BD93F9",
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
		TitleKubeconfig: "#FDBA74",
		StateRunning:    "#22C55E",
		StateDegraded:   "#EAB308",
		StateStopped:    "#F43F5E",
		StateInactive:   "#9CA3AF",
		StateError:      "#DC2626",
	},
	{
		Name:            "Tokyo Night",
		Focus:           "#7AA2F7",
		Border:          "#565F89",
		Selected:        "#BB9AF7",
		TitleServices:   "#7DCFFF",
		TitleWorkspace:  "#9ECE6A",
		TitleTypes:      "#7AA2F7",
		TitleFilters:    "#E0AF68",
		TitleThemes:     "#BB9AF7",
		TitleAddService: "#F7768E",
		TitleRemote:     "#73DACA",
		TitleKubeconfig: "#C0CAF5",
		StateRunning:    "#9ECE6A",
		StateDegraded:   "#E0AF68",
		StateStopped:    "#F7768E",
		StateInactive:   "#565F89",
		StateError:      "#DB4B4B",
	},
	{
		Name:            "Catppuccin Mocha",
		Focus:           "#89B4FA",
		Border:          "#6C7086",
		Selected:        "#CBA6F7",
		TitleServices:   "#89B4FA",
		TitleWorkspace:  "#A6E3A1",
		TitleTypes:      "#74C7EC",
		TitleFilters:    "#F9E2AF",
		TitleThemes:     "#F5C2E7",
		TitleAddService: "#FAB387",
		TitleRemote:     "#F38BA8",
		TitleKubeconfig: "#B4BEFE",
		StateRunning:    "#A6E3A1",
		StateDegraded:   "#F9E2AF",
		StateStopped:    "#F38BA8",
		StateInactive:   "#6C7086",
		StateError:      "#F38BA8",
	},
	{
		Name:            "Everforest",
		Focus:           "#A7C080",
		Border:          "#7A8478",
		Selected:        "#83C092",
		TitleServices:   "#A7C080",
		TitleWorkspace:  "#D3C6AA",
		TitleTypes:      "#7FBBB3",
		TitleFilters:    "#E69875",
		TitleThemes:     "#DBBC7F",
		TitleAddService: "#E67E80",
		TitleRemote:     "#D699B6",
		TitleKubeconfig: "#7FBBB3",
		StateRunning:    "#A7C080",
		StateDegraded:   "#DBBC7F",
		StateStopped:    "#E67E80",
		StateInactive:   "#7A8478",
		StateError:      "#E67E80",
	},
	{
		Name:            "Kanagawa",
		Focus:           "#7E9CD8",
		Border:          "#727169",
		Selected:        "#957FB8",
		TitleServices:   "#7FB4CA",
		TitleWorkspace:  "#98BB6C",
		TitleTypes:      "#7E9CD8",
		TitleFilters:    "#C0A36E",
		TitleThemes:     "#957FB8",
		TitleAddService: "#E46876",
		TitleRemote:     "#7AA89F",
		TitleKubeconfig: "#DCD7BA",
		StateRunning:    "#98BB6C",
		StateDegraded:   "#C0A36E",
		StateStopped:    "#E46876",
		StateInactive:   "#727169",
		StateError:      "#E82424",
	},
	{
		Name:            "Rose Pine",
		Focus:           "#9CCFD8",
		Border:          "#6E6A86",
		Selected:        "#C4A7E7",
		TitleServices:   "#9CCFD8",
		TitleWorkspace:  "#31748F",
		TitleTypes:      "#C4A7E7",
		TitleFilters:    "#F6C177",
		TitleThemes:     "#EBBCBA",
		TitleAddService: "#EB6F92",
		TitleRemote:     "#908CAA",
		TitleKubeconfig: "#E0DEF4",
		StateRunning:    "#9CCFD8",
		StateDegraded:   "#F6C177",
		StateStopped:    "#EB6F92",
		StateInactive:   "#6E6A86",
		StateError:      "#EB6F92",
	},
	{
		Name:            "One Dark",
		Focus:           "#61AFEF",
		Border:          "#5C6370",
		Selected:        "#C678DD",
		TitleServices:   "#61AFEF",
		TitleWorkspace:  "#98C379",
		TitleTypes:      "#56B6C2",
		TitleFilters:    "#E5C07B",
		TitleThemes:     "#C678DD",
		TitleAddService: "#D19A66",
		TitleRemote:     "#E06C75",
		TitleKubeconfig: "#ABB2BF",
		StateRunning:    "#98C379",
		StateDegraded:   "#E5C07B",
		StateStopped:    "#E06C75",
		StateInactive:   "#5C6370",
		StateError:      "#E06C75",
	},
	{
		Name:            "Solarized Dark",
		Focus:           "#268BD2",
		Border:          "#586E75",
		Selected:        "#2AA198",
		TitleServices:   "#268BD2",
		TitleWorkspace:  "#859900",
		TitleTypes:      "#2AA198",
		TitleFilters:    "#B58900",
		TitleThemes:     "#6C71C4",
		TitleAddService: "#CB4B16",
		TitleRemote:     "#DC322F",
		TitleKubeconfig: "#93A1A1",
		StateRunning:    "#859900",
		StateDegraded:   "#B58900",
		StateStopped:    "#DC322F",
		StateInactive:   "#586E75",
		StateError:      "#DC322F",
	},
	{
		Name:            "Monokai",
		Focus:           "#66D9EF",
		Border:          "#75715E",
		Selected:        "#AE81FF",
		TitleServices:   "#66D9EF",
		TitleWorkspace:  "#A6E22E",
		TitleTypes:      "#FD971F",
		TitleFilters:    "#E6DB74",
		TitleThemes:     "#AE81FF",
		TitleAddService: "#F92672",
		TitleRemote:     "#A1EFE4",
		TitleKubeconfig: "#F8F8F2",
		StateRunning:    "#A6E22E",
		StateDegraded:   "#E6DB74",
		StateStopped:    "#F92672",
		StateInactive:   "#75715E",
		StateError:      "#F92672",
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
