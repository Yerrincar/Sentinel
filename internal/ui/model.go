package tui

import (
	helpers "sentinel/internal/util"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	standardStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	servicesSideStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	workSpaceStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	typesSpaceStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	filtersSpaceStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	cardStyles        = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
)

type MainModel struct {
	height         int
	width          int
	servicesPerRow int
	serviceHeight  int
	serviceWidth   int
	cursor         int
	selected       map[int]struct{}
}

func InitialModel() *MainModel {
	return &MainModel{}
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

	standardStyle := standardStyle.Width(m.width).Height(m.height).MarginTop(1)
	servicesSideStyle := servicesSideStyle.Height(m.height - 3).Width(int(float64(m.width) * 0.63)).MarginLeft(1)
	workSpaceStyle := workSpaceStyle.Height(m.height / 12).Width(int(float64(m.width) * 0.33)).MarginLeft(1)
	typesSpaceStyle := typesSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	filtersSpaceStyle := filtersSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	servicesCardsStyle := cardStyles.Width(m.serviceWidth).Height(m.serviceHeight).MarginLeft(2).MarginTop(1)

	for _, i := range items {
		cards = append(cards, servicesCardsStyle.Render(i))
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
	sidePanels := lipgloss.JoinVertical(lipgloss.Left, workSpacePanel, lipgloss.JoinHorizontal(lipgloss.Left,
		typesSpacePanel,
		filtersSpacePanel))

	return standardStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, sidePanels,
		servicesPanel))
}
