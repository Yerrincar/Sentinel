package tui

import (
	"sentinel/internal/backend/docker"
	kubernetes "sentinel/internal/backend/k8s"
	"sentinel/internal/backend/systemd"
	"sentinel/internal/config"
	"sentinel/internal/model"
	theme "sentinel/internal/ui/themes"
	helpers "sentinel/internal/util"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
	themesSpaceStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
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
	themesFocus
	addServiceFocus
	remoteConectionFocus
)

type MainModel struct {
	items              []string
	services           []config.ServiceDef
	typeOptions        []string
	filterOptions      []string
	typeCursor         int
	filterCursor       int
	selectedType       string
	selectedState      string
	runtimeByID        map[string]model.ServiceRuntime
	height             int
	width              int
	servicesPerRow     int
	serviceHeight      int
	serviceWidth       int
	innerWidth         int
	innerHeight        int
	cursor             int
	activeArea         focusArea
	contentFocus       bool
	viewport           viewport.Model
	workspace          textinput.Model
	workspaceActivated bool
	themesViewport     viewport.Model
	themes             []theme.Palette
	themeCursor        int
	selectedTheme      string
	palette            theme.Palette
	configHandler      *config.YamlConfig
	configServiceDef   *config.ServiceDef
	samplerStruct      *systemd.Sampler
	lastTick           time.Time
	interval           time.Duration
}

func InitialModel(y *config.YamlConfig, d *config.ServiceDef, s *systemd.Sampler, services []config.ServiceDef) *MainModel {
	t := textinput.New()
	t.Placeholder = ""
	t.SetValue(y.Settings.Workspace.Name)
	t.Blur()
	t.Width = 40
	p := theme.Default()
	if loaded, err := theme.LoadSelected(); err == nil {
		p = loaded
	}
	allThemes := theme.All()

	return &MainModel{
		items:              make([]string, 0),
		typeOptions:        []string{"All", "Docker", "Systemd", "K8s"},
		filterOptions:      []string{"All", "Running", "Degraded", "Stopped", "Inactive"},
		activeArea:         workSpaceFocus,
		configHandler:      y,
		configServiceDef:   d,
		samplerStruct:      s,
		services:           services,
		runtimeByID:        map[string]model.ServiceRuntime{},
		interval:           y.Interval(),
		workspace:          t,
		workspaceActivated: false,
		themes:             allThemes,
		selectedTheme:      p.Name,
		palette:            p,
		selectedType:       "",
		selectedState:      "",
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
		case "k8s":
			k8sStats := kubernetes.GetMetricsFromPod(t.K8s.Pod, t.K8s.Namespace)
			m.runtimeByID[t.Id] = k8sStats
			serviceInfo = append(serviceInfo, t.Id+"\n"+t.Name+"\n"+t.K8s.Pod+"\n"+
				strconv.FormatFloat(k8sStats.Cpu, 'f', 2, 64)+" %"+"\n"+
				k8sStats.Mem+" / "+k8sStats.MemLimit+"\n"+
				k8sStats.Status+"\n"+
				k8sStats.Uptime+"\n"+
				k8sStats.ErrorMsg)
			m.items = append(m.items, serviceInfo...)
		}
	}
	return m.tickCmd()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	leng := len(m.items)
	if m.selectedType != "" || m.selectedState != "" {
		leng = 0
		for i, s := range m.services {
			if m.matchFilters(i, s) {
				leng++
			}
		}
	}

	//workspace input text
	if m.workspaceActivated == true {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.workspaceActivated = false
				m.workspace.Blur()
				return m, nil
			case "enter":
				workspaceText := strings.TrimSpace(m.workspace.Value())
				if workspaceText != "" {
					m.configHandler.WriteYamlConfigFile(workspaceText)
				}
				m.workspaceActivated = false
				m.workspace.Blur()
				return m, nil
			}
		}
		var cmdWorkspace tea.Cmd
		m.workspace, cmdWorkspace = m.workspace.Update(msg)
		return m, cmdWorkspace
	}
	//normal Update func
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
		case "c":
			m.selectedType = ""
			m.selectedState = ""
			m.typeCursor = 0
			m.filterCursor = 0
		case "enter":
			if m.activeArea == typesFocus && m.contentFocus {
				m.selectedType = m.typeValueByOption()
			}
			if m.activeArea == filtersFocus && m.contentFocus {
				m.selectedState = m.filterValueByOption()
			}
			if m.activeArea == themesFocus && m.contentFocus {
				if m.themeCursor >= 0 && m.themeCursor < len(m.themes) {
					m.palette = m.themes[m.themeCursor]
					m.selectedTheme = m.palette.Name
					_ = theme.SaveSelected(m.selectedTheme)
				}
			}
			if m.activeArea == servicesFocus || m.activeArea == typesFocus || m.activeArea == filtersFocus || m.activeArea == themesFocus {
				m.contentFocus = !m.contentFocus
				if m.contentFocus && m.cursor < 0 {
					m.cursor = 0
				}
				if m.contentFocus && m.activeArea == filtersFocus && m.filterCursor < 0 {
					m.filterCursor = 0
				}
				if m.contentFocus && m.activeArea == themesFocus && m.themeCursor < 0 {
					m.themeCursor = 0
				}
			}
			if m.activeArea == workSpaceFocus {
				m.workspaceActivated = true
				m.workspace.Focus()
				return m, nil
			}
		case "esc":
			m.contentFocus = false
		case "up", "k":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("up", leng)
			} else if m.contentFocus && m.activeArea == typesFocus {
				m.moveListCursor("up", &m.typeCursor, len(m.typeOptions))
			} else if m.contentFocus && m.activeArea == filtersFocus {
				m.moveListCursor("up", &m.filterCursor, len(m.filterOptions))
			} else if m.contentFocus && m.activeArea == themesFocus {
				m.moveListCursor("up", &m.themeCursor, len(m.themes))
			} else if !m.contentFocus {
				m.moveFocus("up")
			}
		case "down", "j":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("down", leng)
			} else if m.contentFocus && m.activeArea == typesFocus {
				m.moveListCursor("down", &m.typeCursor, len(m.typeOptions))
			} else if m.contentFocus && m.activeArea == filtersFocus {
				m.moveListCursor("down", &m.filterCursor, len(m.filterOptions))
			} else if m.contentFocus && m.activeArea == themesFocus {
				m.moveListCursor("down", &m.themeCursor, len(m.themes))
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
	focus := m.focusThemeColor()
	border := lipgloss.Color(m.palette.Border)

	standardStyle := standardStyle.Width(m.width).Height(m.height).MarginTop(1)
	servicesSideStyle := servicesSideStyle.Height(m.height - 3).Width(int(float64(m.width) * 0.63)).MarginLeft(1)
	workSpaceStyle := workSpaceStyle.Height(m.height/12).Width(int(float64(m.width)*0.33)).MarginLeft(1).Align(lipgloss.Left, lipgloss.Center)
	typesSpaceStyle := typesSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 6).MarginLeft(1)
	filtersSpaceStyle := filtersSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 6).MarginLeft(1)
	remoteConectionStyle := remoteConectionStyle.Width(int(float64(m.width) * 0.33)).Height(m.height / 12).MarginLeft(1)
	addServiceStyle := addServiceStyle.Width(int(float64(m.width) * 0.33)).Height(m.height / 12).MarginLeft(1)
	themesSpaceStyle := themesSpaceStyle.Width(int(float64(m.width)*0.33)/2 - 1).Height(m.height / 3).MarginLeft(1)
	servicesCardsStyle := cardStyles.Width(m.serviceWidth).Height(m.serviceHeight).MarginLeft(2).MarginTop(1)

	workspaceContent := m.workspace.View()
	if m.workspaceActivated == true {
		workspaceContent = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(m.workspace.View())
	}
	servicesContent := m.renderServicesGrid(servicesCardsStyle, m.selectedType, m.selectedState)
	servicesPanel := helpers.BorderTitle(servicesSideStyle.Render(servicesContent), "Services")
	workSpacePanel := helpers.BorderTitle(workSpaceStyle.Render(workspaceContent), "Workspace")
	typesSpacePanel := helpers.BorderTitle(m.renderListPanel(typesSpaceStyle, m.typeOptions, m.typeCursor, m.contentFocus && m.activeArea == typesFocus, m.selectedType, m.typeValueByLabel), "Types")
	filtersSpacePanel := helpers.BorderTitle(m.renderListPanel(filtersSpaceStyle, m.filterOptions, m.filterCursor, m.contentFocus && m.activeArea == filtersFocus, m.selectedState, m.filterValueByLabel), "Filters")
	addServicePanel := helpers.BorderTitle(addServiceStyle.String(), "Add Services")
	remoteSpacePanel := helpers.BorderTitle(remoteConectionStyle.String(), "Remote Conection")
	themesSpacePanel := helpers.BorderTitle(m.renderThemesPanel(themesSpaceStyle), "Themes")

	servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, border)
	workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, border)
	typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, border)
	filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, border)
	addServicePanel = helpers.ColorPanelBorder(addServicePanel, border)
	remoteSpacePanel = helpers.ColorPanelBorder(remoteSpacePanel, border)
	themesSpacePanel = helpers.ColorPanelBorder(themesSpacePanel, border)

	servicesPanel = m.colorPanelTitle(servicesPanel, "Services", lipgloss.Color(m.palette.TitleServices))
	workSpacePanel = m.colorPanelTitle(workSpacePanel, "Workspace", lipgloss.Color(m.palette.TitleWorkspace))
	typesSpacePanel = m.colorPanelTitle(typesSpacePanel, "Types", lipgloss.Color(m.palette.TitleTypes))
	filtersSpacePanel = m.colorPanelTitle(filtersSpacePanel, "Filters", lipgloss.Color(m.palette.TitleFilters))
	addServicePanel = m.colorPanelTitle(addServicePanel, "Add Services", lipgloss.Color(m.palette.TitleAddService))
	remoteSpacePanel = m.colorPanelTitle(remoteSpacePanel, "Remote Conection", lipgloss.Color(m.palette.TitleRemote))
	themesSpacePanel = m.colorPanelTitle(themesSpacePanel, "Themes", lipgloss.Color(m.palette.TitleThemes))

	switch m.activeArea {
	case servicesFocus:
		servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, focus)
	case workSpaceFocus:
		workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, focus)
	case typesFocus:
		typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, focus)
	case filtersFocus:
		filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, focus)
	case themesFocus:
		themesSpacePanel = helpers.ColorPanelBorder(themesSpacePanel, focus)
	case remoteConectionFocus:
		remoteSpacePanel = helpers.ColorPanelBorder(remoteSpacePanel, focus)
	case addServiceFocus:
		addServicePanel = helpers.ColorPanelBorder(addServicePanel, focus)
	}

	sidePanels := lipgloss.JoinVertical(lipgloss.Left, workSpacePanel, lipgloss.JoinHorizontal(lipgloss.Left,
		typesSpacePanel,
		filtersSpacePanel),
		addServicePanel,
		remoteSpacePanel,
		themesSpacePanel)

	return standardStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, sidePanels,
		servicesPanel))
}

func (m *MainModel) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *MainModel) typeValueByOption() string {
	switch m.typeOptions[m.typeCursor] {
	case "All":
		return ""
	case "Docker":
		return "docker"
	case "Systemd":
		return "systemd"
	case "K8s":
		return "k8s"
	default:
		return ""
	}
}

func (m *MainModel) filterValueByOption() string {
	switch m.filterOptions[m.filterCursor] {
	case "All":
		return ""
	case "Running":
		return "running"
	case "Degraded":
		return "degraded"
	case "Stopped":
		return "stopped"
	case "Inactive":
		return "inactive"
	default:
		return ""
	}
}

func (m *MainModel) typeValueByLabel(label string) string {
	switch label {
	case "All":
		return ""
	case "Docker":
		return "docker"
	case "Systemd":
		return "systemd"
	case "K8s":
		return "k8s"
	default:
		return ""
	}
}

func (m *MainModel) filterValueByLabel(label string) string {
	switch label {
	case "All":
		return ""
	case "Running":
		return "running"
	case "Degraded":
		return "degraded"
	case "Stopped":
		return "stopped"
	case "Inactive":
		return "inactive"
	default:
		return ""
	}
}

func (m *MainModel) matchFilters(i int, s config.ServiceDef) bool {
	if m.selectedType != "" && s.TypeOfService != m.selectedType {
		return false
	}
	if m.selectedState == "" {
		return true
	}
	if i >= len(m.services) {
		return false
	}
	rt := m.runtimeByID[m.services[i].Id]
	return rt.State == m.selectedState
}

func (m *MainModel) renderServicesGrid(cardStyle lipgloss.Style, selectedType, selectedState string) string {
	filtered := make([]string, 0, len(m.items))
	for i, item := range m.items {
		if i < len(m.services) {
			svc := m.services[i]
			if (selectedType == "" || svc.TypeOfService == selectedType) &&
				(selectedState == "" || m.runtimeByID[svc.Id].State == selectedState) {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) > 0 {
		if m.cursor < 0 {
			m.cursor = 0
		}
		if m.cursor >= len(filtered) {
			m.cursor = len(filtered) - 1
		}
	} else {
		m.cursor = 0
	}

	cards := make([]string, 0, len(filtered))
	for idx, item := range filtered {
		card := cardStyle.Render(item)
		if idx < len(filtered) && idx < len(m.services) {
			svcIdx := m.filteredServiceIndex(idx, selectedType, selectedState)
			if svcIdx >= 0 && svcIdx < len(m.services) {
				rt := m.runtimeByID[m.services[svcIdx].Id]
				card = helpers.ColorPanelBorder(card, m.serviceStateColor(rt))
			}
		}
		if m.contentFocus && m.activeArea == servicesFocus && idx == m.cursor {
			card = helpers.ColorPanelBorder(card, m.focusThemeColor())
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

	return m.viewport.View()
}

func (m *MainModel) renderListPanel(panelStyle lipgloss.Style, options []string, cursor int, focused bool, selectedValue string, mapValue func(string) string) string {
	itemWidth := panelStyle.GetWidth() - 2
	if itemWidth < 0 {
		itemWidth = 0
	}

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Width(itemWidth)
	activeStyle := inactiveStyle.Foreground(m.focusThemeColor())
	selectedStyle := inactiveStyle.
		Foreground(lipgloss.Color(m.palette.Selected)).
		Underline(true)

	rendered := make([]string, len(options))
	for i, option := range options {
		isSelected := selectedValue != "" && mapValue(option) == selectedValue
		prefix := "- "
		if isSelected {
			prefix = "* "
		}
		if focused && i == cursor {
			prefix = "> "
			rendered[i] = activeStyle.Render(prefix + option)
			continue
		}
		if isSelected {
			rendered[i] = selectedStyle.Render(prefix + option)
			continue
		}
		rendered[i] = inactiveStyle.Render(prefix + option)
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rendered...))
}

func (m *MainModel) renderThemesPanel(panelStyle lipgloss.Style) string {
	w := panelStyle.GetWidth() - 2
	h := panelStyle.GetHeight() - 2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if m.themesViewport.Width == 0 && m.themesViewport.Height == 0 {
		m.themesViewport = viewport.New(w, h)
	} else {
		m.themesViewport.Width = w
		m.themesViewport.Height = h
	}

	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Width(w)
	activeStyle := inactiveStyle.Foreground(m.focusThemeColor())
	selectedStyle := inactiveStyle.Foreground(lipgloss.Color(m.palette.Selected))

	lines := make([]string, 0, len(m.themes))
	for i, p := range m.themes {
		line := "- " + p.Name
		if m.contentFocus && m.activeArea == themesFocus && i == m.themeCursor {
			line = activeStyle.Render("> " + p.Name)
		} else if p.Name == m.selectedTheme {
			line = selectedStyle.Render(line)
		} else {
			line = inactiveStyle.Render(line)
		}
		lines = append(lines, line)
	}

	m.themesViewport.SetContent(strings.Join(lines, "\n"))
	if m.contentFocus && m.activeArea == themesFocus {
		if m.themeCursor < m.themesViewport.YOffset {
			m.themesViewport.YOffset = m.themeCursor
		}
		if m.themeCursor >= m.themesViewport.YOffset+m.themesViewport.Height {
			m.themesViewport.YOffset = m.themeCursor - m.themesViewport.Height + 1
		}
		if m.themesViewport.YOffset < 0 {
			m.themesViewport.YOffset = 0
		}
	}
	return panelStyle.Render(m.themesViewport.View())
}

func (m *MainModel) filteredServiceIndex(filteredIdx int, selectedType, selectedState string) int {
	count := -1
	for i, svc := range m.services {
		if (selectedType == "" || svc.TypeOfService == selectedType) &&
			(selectedState == "" || m.runtimeByID[svc.Id].State == selectedState) {
			count++
			if count == filteredIdx {
				return i
			}
		}
	}
	return -1
}

func (m *MainModel) focusThemeColor() lipgloss.Color {
	if m.palette.Focus != "" {
		return lipgloss.Color(m.palette.Focus)
	}
	return focusColor
}

func (m *MainModel) serviceStateColor(rt model.ServiceRuntime) lipgloss.Color {
	if rt.ErrorMsg != "" {
		return lipgloss.Color(m.palette.StateError)
	}
	switch rt.State {
	case "running":
		return lipgloss.Color(m.palette.StateRunning)
	case "degraded":
		return lipgloss.Color(m.palette.StateDegraded)
	case "stopped":
		return lipgloss.Color(m.palette.StateStopped)
	case "inactive":
		return lipgloss.Color(m.palette.StateInactive)
	default:
		return lipgloss.Color(m.palette.StateDegraded)
	}
}

func (m *MainModel) colorPanelTitle(panel, title string, color lipgloss.Color) string {
	needle := " " + title + " "
	colored := lipgloss.NewStyle().Foreground(color).Bold(true).Render(needle)
	return strings.Replace(panel, needle, colored, 1)
}

func (m *MainModel) moveListCursor(dir string, cursor *int, total int) {
	if total < 1 {
		*cursor = 0
		return
	}
	if *cursor < 0 {
		*cursor = 0
	}
	if *cursor >= total {
		*cursor = total - 1
	}

	switch dir {
	case "up":
		if *cursor > 0 {
			*cursor--
		}
	case "down":
		if *cursor < total-1 {
			*cursor++
		}
	}
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
		case "k8s":
			rt := kubernetes.GetMetricsFromPod(s.K8s.Pod, s.K8s.Namespace)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, s.Id+"\n"+s.Name+"\n"+s.K8s.Pod+"\n"+
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
		case "down", "j":
			m.activeArea = themesFocus
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
	case themesFocus:
		switch dir {
		case "up", "k":
			m.activeArea = remoteConectionFocus
		case "right", "l":
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
