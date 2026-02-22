package tui

import (
	helpers "sentinel/internal/util"

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
	items          []string
	height         int
	width          int
	servicesPerRow int
	serviceHeight  int
	serviceWidth   int
	innerWidth     int
	innerHeight    int
	cursor         int
	activeArea     focusArea
	contentFocus   bool
	viewport       viewport.Model
}

func InitialModel() *MainModel {

	return &MainModel{
		items:      make([]string, 0),
		activeArea: workSpaceFocus,
	}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	leng := len(m.items)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.activeArea == servicesFocus {
				m.contentFocus = !m.contentFocus
				if m.contentFocus && m.cursor < 0 {
					m.cursor = 0
				}
			}
		case "esc":
			m.contentFocus = false
		case "up", "k":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("up", leng)
			} else if !m.contentFocus {
				m.moveFocus("up")
			}
		case "down", "j":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("down", leng)
			} else if !m.contentFocus {
				m.moveFocus("down")
			}
		case "left", "h":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("left", leng)
			} else if !m.contentFocus {
				m.moveFocus("left")
			}
		case "right", "l":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("right", leng)
			} else if !m.contentFocus {
				m.moveFocus("right")
			}
		}
	}
	//Total width is 167 and side width is 105 for my screen
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.items = []string{"Service1", "Service2", "Service3", "Service4",
			"Service5", "Service6", "Service7"}
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		standarSideWidth := int(float64(m.width) * 0.63)

		m.servicesPerRow = (standarSideWidth / 35)
		if m.servicesPerRow < 1 {
			m.servicesPerRow = 1
		}
		m.serviceWidth = (standarSideWidth / 3) - 5
		if m.serviceWidth < 12 {
			m.serviceWidth = 12
		}
		m.serviceHeight = m.serviceWidth / 2

		m.innerWidth = standarSideWidth - 1
		m.innerHeight = m.height - 4
		if m.innerWidth < 1 {
			m.innerWidth = 1
		}
		if m.innerHeight < 1 {
			m.innerHeight = 1
		}

		if m.viewport.Width == 0 && m.viewport.Height == 0 {
			m.viewport = viewport.New(m.innerWidth, m.innerHeight)
		} else {
			m.viewport.Width = m.innerWidth
			m.viewport.Height = m.innerHeight
		}
	}
	return m, nil
}

func (m *MainModel) View() string {
	cards := make([]string, 0, len(m.items))
	if len(m.items) > 0 {
		if m.cursor < 0 {
			m.cursor = 0
		}
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
	}

	standardStyle := standardStyle.Width(m.width).Height(m.height).MarginTop(1)
	servicesSideStyle := servicesSideStyle.Height(m.height - 3).Width(int(float64(m.width) * 0.63)).MarginLeft(1)
	workSpaceStyle := workSpaceStyle.Height(m.height / 12).Width(int(float64(m.width) * 0.33)).MarginLeft(1)
	typesSpaceStyle := typesSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	filtersSpaceStyle := filtersSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	servicesCardsStyle := cardStyles.Width(m.serviceWidth).Height(m.serviceHeight).MarginLeft(2).MarginTop(1)

	for idx, i := range m.items {
		card := servicesCardsStyle.Render(i)
		if m.contentFocus && m.activeArea == servicesFocus && idx == m.cursor {
			card = helpers.ColorPanelBorder(card, focusColor)
		}
		cards = append(cards, card)
	}
	cardsPerRow := m.servicesPerRow
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}
	rows := make([]string, 0, (len(cards)+cardsPerRow-1)/cardsPerRow)
	for i := 0; i < len(cards); i += cardsPerRow {
		endIdx := i + cardsPerRow
		if endIdx > len(cards) {
			endIdx = len(cards)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards[i:endIdx]...))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	m.viewport.SetContent(content)
	if m.contentFocus && m.activeArea == servicesFocus && len(rows) > 0 {
		rowHeight := lipgloss.Height(rows[0])
		if rowHeight < 1 {
			rowHeight = 1
		}

		selectedRow := m.cursor / cardsPerRow
		top := selectedRow * rowHeight
		bottom := top + rowHeight

		if top < m.viewport.YOffset {
			m.viewport.YOffset = top
		}
		if bottom > m.viewport.YOffset+m.viewport.Height {
			m.viewport.YOffset = bottom - m.viewport.Height
		}
		if m.viewport.YOffset < 0 {
			m.viewport.YOffset = 0
		}
	}

	servicesPanel := helpers.BorderTitle(servicesSideStyle.Render(m.viewport.View()), "Services")
	workSpacePanel := helpers.BorderTitle(workSpaceStyle.String(), "Workspace")
	typesSpacePanel := helpers.BorderTitle(typesSpaceStyle.String(), "Types")
	filtersSpacePanel := helpers.BorderTitle(filtersSpaceStyle.String(), "Filters")

	if m.activeArea == servicesFocus {
		servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, focusColor)
	}
	if m.activeArea == workSpaceFocus {
		workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, focusColor)
	}
	if m.activeArea == typesFocus {
		typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, focusColor)
	}
	if m.activeArea == filtersFocus {
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
			m.activeArea = workSpaceFocus
		}
	case workSpaceFocus:
		switch dir {
		case "right":
			m.activeArea = servicesFocus
		case "down":
			m.activeArea = typesFocus
		}
	case typesFocus:
		switch dir {
		case "up":
			m.activeArea = workSpaceFocus
		case "right":
			m.activeArea = filtersFocus
		}
	case filtersFocus:
		switch dir {
		case "up":
			m.activeArea = workSpaceFocus
		case "left":
			m.activeArea = typesFocus
		case "right":
			m.activeArea = servicesFocus
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
