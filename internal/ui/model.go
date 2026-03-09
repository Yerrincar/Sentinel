package tui

import (
	"sentinel/internal/backend/docker"
	"sentinel/internal/backend/systemd"
	"sentinel/internal/config"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusColor = lipgloss.Color("#3333FF")

	standardStyle        = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	servicesSideStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	workSpaceStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	typesSpaceStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	filtersSpaceStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	remoteConectionStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	addServiceStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	cardStyles           = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
)

type focusArea int
type TickMsg time.Time

const (
	servicesFocus focusArea = iota
	workSpaceFocus
	typesFocus
	filtersFocus
	addServiceFocus
	remoteConectionFocus
)

type MainModel struct {
	items            []string
	services         []config.ServiceDef
	runtimeByID      map[string]model.ServiceRuntime
	height           int
	width            int
	servicesPerRow   int
	serviceHeight    int
	serviceWidth     int
	innerWidth       int
	innerHeight      int
	cursor           int
	activeArea       focusArea
	contentFocus     bool
	viewport         viewport.Model
	configHandler    *config.YamlConfig
	configServiceDef *config.ServiceDef
	samplerStruct    *systemd.Sampler
	lastTick         time.Time
	interval         time.Duration
}

func InitialModel(y *config.YamlConfig, d *config.ServiceDef, s *systemd.Sampler, services []config.ServiceDef) *MainModel {
	return &MainModel{
		items:            make([]string, 0),
		activeArea:       workSpaceFocus,
		configHandler:    y,
		configServiceDef: d,
		samplerStruct:    s,
		services:         services,
		runtimeByID:      map[string]model.ServiceRuntime{},
		interval:         y.Interval(),
	}
}

func (m *MainModel) Init() tea.Cmd {
	for _, t := range m.services {
		serviceInfo := make([]string, 0)
		switch t.TypeOfService {
		case "docker":
			dockerStats := docker.GetMetricsFromContainer(t.Docker.ContainerName)
			m.runtimeByID[t.Id] = dockerStats
			dockerMetrics := m.runtimeByID[t.Id]
			serviceInfo = append(serviceInfo, t.Id+"\n"+t.Name+"\n"+t.Docker.ContainerName+"\n"+t.Url+
				"\n"+
				strconv.FormatFloat(dockerMetrics.Cpu, 'f', 2, 64)+" %"+"\n"+
				dockerMetrics.Mem+" / "+dockerMetrics.MemLimit+"\n"+
				dockerMetrics.Status+"\n"+
				dockerMetrics.Uptime+"\n"+
				"\n"+dockerMetrics.ErrorMsg)
			m.items = append(m.items, serviceInfo...)
		case "systemd":
			systemdStats := m.samplerStruct.GetSystemdMetrics(t.Id, t.Systemd.Unit)
			m.runtimeByID[t.Id] = systemdStats
			serviceInfo = append(serviceInfo, t.Id+"\n"+t.Name+"\n"+t.Systemd.Unit+"\n"+"\n"+
				strconv.FormatFloat(systemdStats.Cpu, 'f', 2, 64)+" %"+"\n"+
				systemdStats.Mem+" / "+systemdStats.MemLimit+"\n"+
				systemdStats.Status+"\n"+
				systemdStats.Uptime+"\n"+
				systemdStats.ErrorMsg)
			m.items = append(m.items, serviceInfo...)
		}
	}
	return m.tickCmd()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	leng := len(m.items)
	switch msg := msg.(type) {
	case TickMsg:
		if m.interval > 0 {
			m.lastTick = time.Time(msg)
			m.refreshCard()
			return m, m.tickCmd()
		}
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
	remoteConectionStyle := remoteConectionStyle.Width(int(float64(m.width) * 0.33)).Height(m.height / 12).MarginLeft(1)
	addServiceStyle := addServiceStyle.Width(int(float64(m.width) * 0.33)).Height(m.height / 12).MarginLeft(1)
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
	addServicePanel := helpers.BorderTitle(addServiceStyle.String(), "Add Services")
	remoteSpacePanel := helpers.BorderTitle(remoteConectionStyle.String(), "Remote Conection")

	switch m.activeArea {
	case servicesFocus:
		servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, focusColor)
	case workSpaceFocus:
		workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, focusColor)
	case typesFocus:
		typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, focusColor)
	case filtersFocus:
		filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, focusColor)
	case remoteConectionFocus:
		remoteSpacePanel = helpers.ColorPanelBorder(remoteSpacePanel, focusColor)
	case addServiceFocus:
		addServicePanel = helpers.ColorPanelBorder(addServicePanel, focusColor)
	}

	sidePanels := lipgloss.JoinVertical(lipgloss.Left, workSpacePanel, lipgloss.JoinHorizontal(lipgloss.Left,
		typesSpacePanel,
		filtersSpacePanel),
		addServicePanel,
		remoteSpacePanel)

	return standardStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, sidePanels,
		servicesPanel))
}

func (m *MainModel) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *MainModel) refreshCard() {
	newItems := make([]string, 0, len(m.services))
	for _, s := range m.services {
		switch s.TypeOfService {
		case "docker":
			rt := docker.GetMetricsFromContainer(s.Docker.ContainerName)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, s.Id+"\n"+s.Name+"\n"+s.Docker.ContainerName+"\n"+s.Url+"\n"+
				strconv.FormatFloat(rt.Cpu, 'f', 2, 64)+" %\n"+rt.Mem+" / "+
				rt.MemLimit+"\n"+
				rt.Status+"\n"+
				rt.Uptime+"\n"+
				rt.ErrorMsg)
		case "systemd":
			rt := m.samplerStruct.GetSystemdMetrics(s.Id, s.Systemd.Unit)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, s.Id+"\n"+s.Name+"\n"+s.Systemd.Unit+"\n"+"\n"+
				strconv.FormatFloat(rt.Cpu, 'f', 2, 64)+" %\n"+
				rt.Mem+" / "+rt.MemLimit+"\n"+
				rt.Status+"\n"+
				rt.Uptime+"\n"+
				rt.ErrorMsg)
		}

	}
	m.items = newItems
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *MainModel) moveFocus(dir string) {
	switch focusArea(m.activeArea) {
	case servicesFocus:
		if dir == "left" || dir == "h" {
			m.activeArea = workSpaceFocus
		}
	case workSpaceFocus:
		switch dir {
		case "right", "l":
			m.activeArea = servicesFocus
		case "down", "j":
			m.activeArea = typesFocus
		}
	case typesFocus:
		switch dir {
		case "up", "k":
			m.activeArea = workSpaceFocus
		case "right", "l":
			m.activeArea = filtersFocus
		case "down", "j":
			m.activeArea = addServiceFocus
		}
	case filtersFocus:
		switch dir {
		case "up", "k":
			m.activeArea = workSpaceFocus
		case "left", "h":
			m.activeArea = typesFocus
		case "right", "l":
			m.activeArea = servicesFocus
		case "down", "j":
			m.activeArea = addServiceFocus
		}
	case remoteConectionFocus:
		switch dir {
		case "up", "k":
			m.activeArea = addServiceFocus
		case "right", "l":
			m.activeArea = servicesFocus
		}
	case addServiceFocus:
		switch dir {
		case "up", "k":
			m.activeArea = typesFocus
		case "right", "l":
			m.activeArea = servicesFocus
		case "down", "j":
			m.activeArea = remoteConectionFocus
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
	case "left", "h":
		if m.cursor > 0 {
			m.cursor--
		}
	case "right", "l":
		if m.cursor < total-1 {
			m.cursor++
		}
	case "up", "k":
		if m.cursor-cols >= 0 {
			m.cursor -= cols
		}
	case "down", "j":
		if m.cursor+cols < total {
			m.cursor += cols
		}
	}
}
