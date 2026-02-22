package tui

import (
	helpers "sentinel/internal/util"
	"strconv"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusColor = lipgloss.Color("#3333FF")

	standardStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	servicesSideStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	workSpaceStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	typesSpaceStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	filtersSpaceStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	cardStyles        = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
)

type focusArea int

const (
	servicesFocus focusArea = iota
	workSpaceFocus
	typesFocus
	filtersFocus
)

type MainModel struct {
	height         int
	width          int
	servicesPerRow int
	serviceHeight  int
	serviceWidth   int
	cursor         int
	activeArea     int
	contentFocus   bool
	viewport       viewport.Model
}

func InitialModel() *MainModel {
	return &MainModel{
		activeArea: int(workSpaceFocus),
	}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.activeArea == int(servicesFocus) {
				m.contentFocus = !m.contentFocus
				if m.contentFocus && m.cursor < 0 {
					m.cursor = 0
				}
			}
		case "esc":
			m.contentFocus = false
		case "up", "k":
			if m.contentFocus && m.activeArea == int(servicesFocus) {
				m.moveServicesCursor("up", 6)
			} else if !m.contentFocus {
				m.moveFocus("up")
			}
		case "down", "j":
			if m.contentFocus && m.activeArea == int(servicesFocus) {
				m.moveServicesCursor("down", 6)
			} else if !m.contentFocus {
				m.moveFocus("down")
			}
		case "left", "h":
			if m.contentFocus && m.activeArea == int(servicesFocus) {
				m.moveServicesCursor("left", 6)
			} else if !m.contentFocus {
				m.moveFocus("left")
			}
		case "right", "l":
			if m.contentFocus && m.activeArea == int(servicesFocus) {
				m.moveServicesCursor("right", 6)
			} else if !m.contentFocus {
				m.moveFocus("right")
			}
		}
	}

	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		m.servicesPerRow = ((int(float64(m.width) * 0.63)) / 35)
		if m.servicesPerRow < 1 {
			m.servicesPerRow = 1
		}
		m.serviceWidth = ((int(float64(m.width) * 0.63)) / 3) - 5
		if m.serviceWidth < 12 {
			m.serviceWidth = 12
		}
		m.serviceWidth = ((int(float64(m.width) * 0.63)) / 3) - 5
		m.serviceHeight = m.serviceWidth / 2
	}
	return m, nil
}

func (m *MainModel) View() string {
	items := []string{strconv.Itoa(m.width), strconv.Itoa(int(float64(m.width) * 0.63)), "Service3", "Service4", "Service5", "Service6"}
	cards := make([]string, 0)
	if len(items) > 0 {
		if m.cursor < 0 {
			m.cursor = 0
		}
		if m.cursor >= len(items) {
			m.cursor = len(items) - 1
		}
	}

	standardStyle := standardStyle.Width(m.width).Height(m.height).MarginTop(1)
	servicesSideStyle := servicesSideStyle.Height(m.height - 3).Width(int(float64(m.width) * 0.63)).MarginLeft(1)
	workSpaceStyle := workSpaceStyle.Height(m.height / 12).Width(int(float64(m.width) * 0.33)).MarginLeft(1)
	typesSpaceStyle := typesSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	filtersSpaceStyle := filtersSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	servicesCardsStyle := cardStyles.Width(m.serviceWidth).Height(m.serviceHeight).MarginLeft(2).MarginTop(1)

	for idx, i := range items {
		card := servicesCardsStyle.Render(i)
		if m.contentFocus && m.activeArea == int(servicesFocus) && idx == m.cursor {
			card = helpers.ColorPanelBorder(card, focusColor)
		}
		cards = append(cards, card)
	}
	cardsPerRow := m.servicesPerRow
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}
	var rows []string
	for i := 0; i < len(cards); i += cardsPerRow {
		endIdx := i + cardsPerRow
		if endIdx > len(cards) {
			endIdx = len(cards)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards[i:endIdx]...))
	}

	content := lipgloss.JoinVertical(lipgloss.Right, rows...)
	servicesPanel := helpers.BorderTitle(servicesSideStyle.Render(content), "Services")
	workSpacePanel := helpers.BorderTitle(workSpaceStyle.String(), "Workspace")
	typesSpacePanel := helpers.BorderTitle(typesSpaceStyle.String(), "Types")
	filtersSpacePanel := helpers.BorderTitle(filtersSpaceStyle.String(), "Filters")

	if m.activeArea == int(servicesFocus) {
		servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, focusColor)
	}
	if m.activeArea == int(workSpaceFocus) {
		workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, focusColor)
	}
	if m.activeArea == int(typesFocus) {
		typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, focusColor)
	}
	if m.activeArea == int(filtersFocus) {
		filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, focusColor)
	}

	sidePanels := lipgloss.JoinVertical(lipgloss.Left, workSpacePanel, lipgloss.JoinHorizontal(lipgloss.Left,
		typesSpacePanel,
		filtersSpacePanel))

	return standardStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, sidePanels,
		servicesPanel))
}

func (m *MainModel) moveFocus(dir string) {
	switch focusArea(m.activeArea) {
	case servicesFocus:
		if dir == "left" {
			m.activeArea = int(workSpaceFocus)
		}
	case workSpaceFocus:
		switch dir {
		case "right":
			m.activeArea = int(servicesFocus)
		case "down":
			m.activeArea = int(typesFocus)
		}
	case typesFocus:
		switch dir {
		case "up":
			m.activeArea = int(workSpaceFocus)
		case "right":
			m.activeArea = int(filtersFocus)
		}
	case filtersFocus:
		switch dir {
		case "up":
			m.activeArea = int(workSpaceFocus)
		case "left":
			m.activeArea = int(typesFocus)
		case "right":
			m.activeArea = int(servicesFocus)
		}
	}
}

func (m *MainModel) moveServicesCursor(dir string, total int) {
	if total < 1 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}

	cols := m.servicesPerRow
	if cols < 1 {
		cols = 1
	}

	switch dir {
	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
	case "right":
		if m.cursor < total-1 {
			m.cursor++
		}
	case "up":
		if m.cursor-cols >= 0 {
			m.cursor -= cols
		}
	case "down":
		if m.cursor+cols < total {
			m.cursor += cols
		}
	}
}
