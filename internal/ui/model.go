package tui

import (
	"fmt"
	"os"
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
	logsSpaceStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	remoteConectionStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	kubeconfigStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	cardStyles           = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
)

type focusArea int
type TickMsg time.Time

const (
	servicesFocus focusArea = iota
	workSpaceFocus
	typesFocus
	filtersFocus
	logsFocus
	remoteConectionFocus
	kubeconfigFocus
)

type MainModel struct {
	items               []string
	services            []config.ServiceDef
	typeOptions         []string
	filterOptions       []string
	typeCursor          int
	filterCursor        int
	selectedType        string
	selectedState       string
	runtimeByID         map[string]model.ServiceRuntime
	height              int
	width               int
	servicesPerRow      int
	serviceHeight       int
	serviceWidth        int
	innerWidth          int
	innerHeight         int
	cursor              int
	activeArea          focusArea
	contentFocus        bool
	viewport            viewport.Model
	workspace           textinput.Model
	workspaceActivated  bool
	kubeconfig          textinput.Model
	kubeconfigActive    bool
	addServiceMode      bool
	deleteServiceMode   bool
	deleteServiceIdx    int
	deleteConfirmCursor int
	deleteError         string
	addTypeOptions      []string
	addTypeCursor       int
	addFieldCursor      int
	addInputs           []textinput.Model
	addError            string
	logsViewport        viewport.Model
	logsContent         string
	themes              []theme.Palette
	themeCursor         int
	selectedTheme       string
	palette             theme.Palette
	configHandler       *config.YamlConfig
	configServiceDef    *config.ServiceDef
	samplerStruct       *systemd.Sampler
	lastTick            time.Time
	interval            time.Duration
}

func InitialModel(y *config.YamlConfig, d *config.ServiceDef, s *systemd.Sampler, services []config.ServiceDef) *MainModel {
	t := textinput.New()
	t.Placeholder = ""
	t.SetValue(y.Settings.Workspace.Name)
	t.Blur()
	t.Width = 40

	k := textinput.New()
	k.Placeholder = "Set/Update your KUBECONFIG path"
	k.Prompt = "> "
	k.SetValue(os.Getenv("KUBECONFIG"))
	k.EchoMode = textinput.EchoPassword
	k.EchoCharacter = '*'
	k.Blur()
	k.Width = 40

	p := theme.Default()
	if loaded, err := theme.LoadSelected(); err == nil {
		p = loaded
	}
	allThemes := theme.All()

	return &MainModel{
		items:               make([]string, 0),
		typeOptions:         []string{"All", "Docker", "Systemd", "K8s"},
		filterOptions:       []string{"All", "Running", "Degraded", "Stopped", "Inactive"},
		addTypeOptions:      []string{"Docker", "Systemd", "K8s"},
		activeArea:          workSpaceFocus,
		configHandler:       y,
		configServiceDef:    d,
		samplerStruct:       s,
		services:            services,
		runtimeByID:         map[string]model.ServiceRuntime{},
		interval:            y.Interval(),
		workspace:           t,
		workspaceActivated:  false,
		kubeconfig:          k,
		kubeconfigActive:    false,
		deleteServiceMode:   false,
		deleteServiceIdx:    -1,
		deleteConfirmCursor: 1,
		logsContent:         "Logs preview will appear here.\n\nUse Enter to focus this panel and j/k to scroll.\nUse your logs keybinding to open full logs in a new pane.",
		themes:              allThemes,
		selectedTheme:       p.Name,
		palette:             p,
		selectedType:        "",
		selectedState:       "",
	}
}

func (m *MainModel) Init() tea.Cmd {
	for _, t := range m.services {
		switch t.TypeOfService {
		case "docker":
			dockerStats := docker.GetMetricsFromContainer(t.Docker.ContainerName)
			m.runtimeByID[t.Id] = dockerStats
			m.items = append(m.items, m.buildServiceItem(t, dockerStats))
		case "systemd":
			systemdStats := m.samplerStruct.GetSystemdMetrics(t.Id, t.Systemd.Unit)
			m.runtimeByID[t.Id] = systemdStats
			m.items = append(m.items, m.buildServiceItem(t, systemdStats))
		case "k8s":
			k8sStats := kubernetes.GetMetricsFromDeployment(t.K8s.Deployment, t.K8s.Namespace)
			m.runtimeByID[t.Id] = k8sStats
			m.items = append(m.items, m.buildServiceItem(t, k8sStats))
		}
	}
	m.refreshLogsPreview()
	return m.tickCmd()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if tick, ok := msg.(TickMsg); ok {
		if m.interval > 0 {
			m.lastTick = time.Time(tick)
			m.refreshCard()
			m.refreshLogsPreview()
			return m, m.tickCmd()
		}
	}

	leng := len(m.items)
	if m.selectedType != "" || m.selectedState != "" {
		leng = 0
		for i, s := range m.services {
			if m.matchFilters(i, s) {
				leng++
			}
		}
	}

	if m.addServiceMode {
		return m.handleAddServiceUpdate(msg)
	}
	if m.deleteServiceMode {
		return m.handleDeleteServiceUpdate(msg)
	}

	//text input mode
	if m.workspaceActivated || m.kubeconfigActive {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				if m.workspaceActivated {
					m.workspaceActivated = false
					m.workspace.Blur()
				}
				if m.kubeconfigActive {
					m.kubeconfigActive = false
					m.kubeconfig.Blur()
				}
				return m, nil
			case "enter":
				if m.workspaceActivated {
					workspaceText := strings.TrimSpace(m.workspace.Value())
					if workspaceText != "" {
						m.configHandler.WriteYamlConfigFile(workspaceText)
					}
					m.workspaceActivated = false
					m.workspace.Blur()
					return m, nil
				}
				if m.kubeconfigActive {
					kubeconfigText := strings.TrimSpace(m.kubeconfig.Value())
					_ = kubernetes.UpdateEnvKubeconfig(kubeconfigText, "KUBECONFIG")
					m.kubeconfigActive = false
					m.kubeconfig.Blur()
					return m, nil
				}
				return m, nil
			}
		}
		if m.workspaceActivated {
			var cmdWorkspace tea.Cmd
			m.workspace, cmdWorkspace = m.workspace.Update(msg)
			return m, cmdWorkspace
		}
		var cmdKubeconfig tea.Cmd
		m.kubeconfig, cmdKubeconfig = m.kubeconfig.Update(msg)
		return m, cmdKubeconfig
	}
	//normal Update func
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			m.startAddService()
			return m, nil
		case "d":
			if m.activeArea == servicesFocus && m.contentFocus {
				serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
				if serviceIdx >= 0 && serviceIdx < len(m.services) {
					m.startDeleteService(serviceIdx)
				}
			}
			return m, nil
		case "t":
			m.cycleTheme(1)
			return m, nil
		case "T", "shift+t":
			m.cycleTheme(-1)
			return m, nil
		case "c":
			m.selectedType = ""
			m.selectedState = ""
			m.typeCursor = 0
			m.filterCursor = 0
			m.refreshLogsPreview()
		case " ":
			if m.activeArea == servicesFocus && m.contentFocus {
				serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
				if serviceIdx < 0 || serviceIdx >= len(m.services) {
					break
				}
				service := m.services[serviceIdx]
				switch service.TypeOfService {
				case "docker":
					dockerContainer := service.Docker.ContainerName
					err := docker.DockerStart(dockerContainer)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "systemd":
					systemdUnit := service.Systemd.Unit
					result, err := systemd.SystemdStart(systemdUnit)
					if err != nil && result == 0 {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "k8s":
					err := kubernetes.K8sStart(service.K8s.Namespace, service.K8s.Deployment)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				}
			}
		case "s":
			if m.activeArea == servicesFocus && m.contentFocus {
				serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
				if serviceIdx < 0 || serviceIdx >= len(m.services) {
					break
				}
				service := m.services[serviceIdx]
				switch service.TypeOfService {
				case "docker":
					dockerContainer := service.Docker.ContainerName
					err := docker.DockerStop(dockerContainer)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "systemd":
					systemdUnit := service.Systemd.Unit
					result, err := systemd.SystemdStop(systemdUnit)
					if err != nil && result == 0 {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "k8s":
					err := kubernetes.K8sStop(service.K8s.Namespace, service.K8s.Deployment)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				}
			}
		case "r":
			if m.activeArea == servicesFocus && m.contentFocus {
				serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
				if serviceIdx < 0 || serviceIdx >= len(m.services) {
					break
				}
				service := m.services[serviceIdx]
				switch service.TypeOfService {
				case "docker":
					dockerContainer := service.Docker.ContainerName
					err := docker.DockerRestart(dockerContainer)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "systemd":
					systemdUnit := service.Systemd.Unit
					result, err := systemd.SystemdRestart(systemdUnit)
					if err != nil && result == 0 {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				case "k8s":
					err := kubernetes.K8sRestart(service.K8s.Namespace, service.K8s.Deployment)
					if err != nil {
						rt := m.runtimeByID[service.Id]
						rt.ErrorMsg = err.Error()
						m.runtimeByID[service.Id] = rt
						m.syncServiceItem(serviceIdx)
						break
					}
					m.refreshCard()
				}

			}
		case "enter":
			if m.activeArea == typesFocus && m.contentFocus {
				m.selectedType = m.typeValueByOption()
				m.refreshLogsPreview()
			}
			if m.activeArea == filtersFocus && m.contentFocus {
				m.selectedState = m.filterValueByOption()
				m.refreshLogsPreview()
			}
			if m.activeArea == servicesFocus || m.activeArea == typesFocus || m.activeArea == filtersFocus || m.activeArea == logsFocus {
				m.contentFocus = !m.contentFocus
				if m.contentFocus && m.cursor < 0 {
					m.cursor = 0
				}
				if m.contentFocus && m.activeArea == filtersFocus && m.filterCursor < 0 {
					m.filterCursor = 0
				}
			}
			if m.activeArea == workSpaceFocus {
				m.workspaceActivated = true
				m.workspace.Focus()
				return m, nil
			}
			if m.activeArea == kubeconfigFocus {
				m.kubeconfigActive = true
				m.kubeconfig.Focus()
				return m, nil
			}
		case "esc":
			m.contentFocus = false
		case "up", "k":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("up", leng)
				m.refreshLogsPreview()
			} else if m.contentFocus && m.activeArea == typesFocus {
				m.moveListCursor("up", &m.typeCursor, len(m.typeOptions))
			} else if m.contentFocus && m.activeArea == filtersFocus {
				m.moveListCursor("up", &m.filterCursor, len(m.filterOptions))
			} else if m.contentFocus && m.activeArea == logsFocus {
				m.logsViewport.LineUp(1)
			} else if !m.contentFocus {
				m.moveFocus("up")
			}
		case "down", "j":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("down", leng)
				m.refreshLogsPreview()
			} else if m.contentFocus && m.activeArea == typesFocus {
				m.moveListCursor("down", &m.typeCursor, len(m.typeOptions))
			} else if m.contentFocus && m.activeArea == filtersFocus {
				m.moveListCursor("down", &m.filterCursor, len(m.filterOptions))
			} else if m.contentFocus && m.activeArea == logsFocus {
				m.logsViewport.LineDown(1)
			} else if !m.contentFocus {
				m.moveFocus("down")
			}
		case "left", "h":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("left", leng)
				m.refreshLogsPreview()
			} else if !m.contentFocus {
				m.moveFocus("left")
			}
		case "right", "l":
			if m.contentFocus && m.activeArea == servicesFocus {
				m.moveServicesCursor("right", leng)
				m.refreshLogsPreview()
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
	kubeconfigStyle := kubeconfigStyle.Height(m.height/12).Width(int(float64(m.width)*0.33)).MarginLeft(1).Align(lipgloss.Left, lipgloss.Center)
	logsSpaceStyle := logsSpaceStyle.Width(int(float64(m.width) * 0.33)).Height(m.height/3 + 6).MarginLeft(1)
	servicesCardsStyle := cardStyles.Width(m.serviceWidth).Height(m.serviceHeight).MarginLeft(2).MarginTop(1)

	workspaceContent := m.workspace.View()
	if m.workspaceActivated == true {
		workspaceContent = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(m.workspace.View())
	}
	kubeconfigContent := lipgloss.NewStyle().
		PaddingLeft(1).
		Render("> " + strings.Repeat("*", len([]rune(m.kubeconfig.Value()))))
	if m.kubeconfigActive {
		kubeconfigContent = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(m.kubeconfig.View())
	}
	servicesContent := m.renderServicesGrid(servicesCardsStyle, m.selectedType, m.selectedState)
	servicesPanel := helpers.BorderTitle(servicesSideStyle.Render(servicesContent), "Services")
	workSpacePanel := helpers.BorderTitle(workSpaceStyle.Render(workspaceContent), "Workspace")
	typesSpacePanel := helpers.BorderTitle(m.renderListPanel(typesSpaceStyle, m.typeOptions, m.typeCursor, m.contentFocus && m.activeArea == typesFocus, m.selectedType, m.typeValueByLabel), "Types")
	filtersSpacePanel := helpers.BorderTitle(m.renderListPanel(filtersSpaceStyle, m.filterOptions, m.filterCursor, m.contentFocus && m.activeArea == filtersFocus, m.selectedState, m.filterValueByLabel), "Filters")
	// remoteSpacePanel disabled for MVP; keep widget code for future remote feature.
	kubeconfigPanel := helpers.BorderTitle(kubeconfigStyle.Render(kubeconfigContent), "Kubeconfig")
	logsSidePanel := helpers.BorderTitle(m.renderLogsPanel(logsSpaceStyle), "Logs")

	servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, border)
	workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, border)
	typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, border)
	filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, border)
	kubeconfigPanel = helpers.ColorPanelBorder(kubeconfigPanel, border)
	logsSidePanel = helpers.ColorPanelBorder(logsSidePanel, border)

	servicesPanel = m.colorPanelTitle(servicesPanel, "Services", lipgloss.Color(m.palette.TitleServices))
	workSpacePanel = m.colorPanelTitle(workSpacePanel, "Workspace", lipgloss.Color(m.palette.TitleWorkspace))
	typesSpacePanel = m.colorPanelTitle(typesSpacePanel, "Types", lipgloss.Color(m.palette.TitleTypes))
	filtersSpacePanel = m.colorPanelTitle(filtersSpacePanel, "Filters", lipgloss.Color(m.palette.TitleFilters))
	kubeconfigPanel = m.colorPanelTitle(kubeconfigPanel, "Kubeconfig", lipgloss.Color(m.palette.TitleKubeconfig))
	logsSidePanel = m.colorPanelTitle(logsSidePanel, "Logs", lipgloss.Color(m.palette.TitleThemes))

	switch m.activeArea {
	case servicesFocus:
		servicesPanel = helpers.ColorOuterPanelBorder(servicesPanel, focus)
	case workSpaceFocus:
		workSpacePanel = helpers.ColorPanelBorder(workSpacePanel, focus)
	case typesFocus:
		typesSpacePanel = helpers.ColorPanelBorder(typesSpacePanel, focus)
	case filtersFocus:
		filtersSpacePanel = helpers.ColorPanelBorder(filtersSpacePanel, focus)
	case logsFocus:
		logsSidePanel = helpers.ColorPanelBorder(logsSidePanel, focus)
	case kubeconfigFocus:
		kubeconfigPanel = helpers.ColorPanelBorder(kubeconfigPanel, focus)
	}

	sidePanels := lipgloss.JoinVertical(lipgloss.Left, workSpacePanel, lipgloss.JoinHorizontal(lipgloss.Left,
		typesSpacePanel,
		filtersSpacePanel),
		kubeconfigPanel,
		logsSidePanel)

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.palette.Selected)).
		PaddingLeft(1).
		Render("Move: ↑↓→← / hjkl | Select: enter | Theme: t / Shift+T | Clear filters: c | Add service: a | Delete: d | Quit: ctrl+c/q | Start/Stop/Restart Services: space/s/r")

	base := standardStyle.Render(lipgloss.JoinHorizontal(lipgloss.Bottom,
		lipgloss.JoinHorizontal(lipgloss.Top, sidePanels,
			servicesPanel)), footer)
	if m.addServiceMode {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderAddServiceModal())
	}
	if m.deleteServiceMode {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderDeleteServiceModal())
	}
	return base
}

func (m *MainModel) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *MainModel) startAddService() {
	m.addServiceMode = true
	m.addError = ""
	m.addTypeCursor = 0
	m.addFieldCursor = 1
	m.resetAddInputs()
}

func (m *MainModel) startDeleteService(serviceIdx int) {
	m.deleteServiceMode = true
	m.deleteServiceIdx = serviceIdx
	m.deleteConfirmCursor = 1
	m.deleteError = ""
}

func (m *MainModel) handleDeleteServiceUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.deleteServiceMode = false
			m.deleteError = ""
			return m, nil
		case "left", "h":
			m.deleteConfirmCursor = 0
			return m, nil
		case "right", "l":
			m.deleteConfirmCursor = 1
			return m, nil
		case "enter":
			if m.deleteConfirmCursor == 1 {
				m.deleteServiceMode = false
				m.deleteError = ""
				return m, nil
			}
			if m.deleteServiceIdx < 0 || m.deleteServiceIdx >= len(m.services) {
				m.deleteError = "invalid selected service"
				return m, nil
			}

			svc := m.services[m.deleteServiceIdx]
			if err := m.configHandler.DeleteService(svc.Id); err != nil {
				m.deleteError = err.Error()
				return m, nil
			}

			delete(m.runtimeByID, svc.Id)
			m.services = append(m.services[:m.deleteServiceIdx], m.services[m.deleteServiceIdx+1:]...)
			m.deleteServiceMode = false
			m.deleteServiceIdx = -1
			m.deleteError = ""
			m.refreshCard()
			m.refreshLogsPreview()
			return m, nil
		}
	}
	return m, nil
}

func (m *MainModel) addFieldLabels() []string {
	base := []string{"Id", "Name", "Url"}
	switch strings.ToLower(m.addTypeOptions[m.addTypeCursor]) {
	case "docker":
		return append(base, "Container")
	case "systemd":
		return append(base, "Unit")
	case "k8s":
		return append(base, "Context", "Namespace", "Deployment")
	default:
		return base
	}
}

func (m *MainModel) resetAddInputs() {
	labels := m.addFieldLabels()
	m.addInputs = make([]textinput.Model, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Prompt = label + ": "
		ti.Width = 36
		ti.Blur()
		m.addInputs[i] = ti
	}
	m.focusAddInput()
}

func (m *MainModel) focusAddInput() {
	for i := range m.addInputs {
		m.addInputs[i].Blur()
	}
	if m.addFieldCursor > 0 && m.addFieldCursor <= len(m.addInputs) {
		m.addInputs[m.addFieldCursor-1].Focus()
	}
}

func (m *MainModel) moveAddFieldCursor(dir string) {
	max := len(m.addInputs)
	switch dir {
	case "up":
		if m.addFieldCursor > 0 {
			m.addFieldCursor--
		}
	case "down":
		if m.addFieldCursor < max {
			m.addFieldCursor++
		}
	}
	m.focusAddInput()
}

func (m *MainModel) handleAddServiceUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.addServiceMode = false
			m.addError = ""
			return m, nil
		case "up":
			m.moveAddFieldCursor("up")
			return m, nil
		case "down":
			m.moveAddFieldCursor("down")
			return m, nil
		case "left", "shift+tab":
			m.addTypeCursor = (m.addTypeCursor - 1 + len(m.addTypeOptions)) % len(m.addTypeOptions)
			m.resetAddInputs()
			m.addError = ""
			return m, nil
		case "right", "tab":
			m.addTypeCursor = (m.addTypeCursor + 1) % len(m.addTypeOptions)
			m.resetAddInputs()
			m.addError = ""
			return m, nil
		case "enter":
			if m.addFieldCursor == 0 {
				m.moveAddFieldCursor("down")
				return m, nil
			}
			if m.addFieldCursor < len(m.addInputs) {
				m.moveAddFieldCursor("down")
				return m, nil
			}
			if err := m.submitAddService(); err != nil {
				m.addError = err.Error()
				return m, nil
			}
			m.addServiceMode = false
			m.addError = ""
			m.refreshCard()
			return m, nil
		}
	}

	if m.addFieldCursor > 0 && m.addFieldCursor <= len(m.addInputs) {
		idx := m.addFieldCursor - 1
		var cmd tea.Cmd
		m.addInputs[idx], cmd = m.addInputs[idx].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *MainModel) submitAddService() error {
	labels := m.addFieldLabels()
	values := make(map[string]string, len(labels))
	for i, label := range labels {
		values[label] = strings.TrimSpace(m.addInputs[i].Value())
	}

	if values["Id"] == "" || values["Name"] == "" {
		return fmt.Errorf("id and name are required")
	}
	for _, existing := range m.services {
		if existing.Id == values["Id"] {
			return fmt.Errorf("service id already exists: %s", values["Id"])
		}
	}

	svcType := strings.ToLower(m.addTypeOptions[m.addTypeCursor])
	svc := config.ServiceDef{
		Id:            values["Id"],
		Name:          values["Name"],
		TypeOfService: svcType,
		Url:           values["Url"],
	}
	switch svcType {
	case "docker":
		if values["Container"] == "" {
			return fmt.Errorf("container is required")
		}
		svc.Docker.ContainerName = values["Container"]
	case "systemd":
		if values["Unit"] == "" {
			return fmt.Errorf("unit is required")
		}
		svc.Systemd.Unit = values["Unit"]
	case "k8s":
		if values["Namespace"] == "" || values["Deployment"] == "" {
			return fmt.Errorf("namespace and deployment are required")
		}
		svc.K8s.Context = values["Context"]
		svc.K8s.Namespace = values["Namespace"]
		svc.K8s.Deployment = values["Deployment"]
	default:
		return fmt.Errorf("unsupported type: %s", svcType)
	}

	if err := m.configHandler.AddService(svc); err != nil {
		return err
	}
	m.services = append(m.services, svc)
	return nil
}

func (m *MainModel) renderAddServiceModal() string {
	focus := m.focusThemeColor()
	border := lipgloss.Color(m.palette.Border)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(focus).Render("Add Service"),
		"Use ←/→/tab/shift+tab to change type, ↑/↓ to move, Enter to continue, Esc to cancel",
	}

	typeLine := "  Type: " + m.addTypeOptions[m.addTypeCursor]
	if m.addFieldCursor == 0 {
		typeLine = lipgloss.NewStyle().Foreground(focus).Render("> Type: " + m.addTypeOptions[m.addTypeCursor])
	}
	lines = append(lines, typeLine)

	for i := range m.addInputs {
		line := "  " + m.addInputs[i].View()
		if m.addFieldCursor == i+1 {
			line = lipgloss.NewStyle().Foreground(focus).Render("> " + m.addInputs[i].View())
		}
		lines = append(lines, line)
	}

	if m.addError != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.palette.StateError)).Render("Error: "+m.addError))
	}

	w := 64
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}
	if w < 40 {
		w = 40
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2).
		Width(w).
		Render(strings.Join(lines, "\n"))
}

func (m *MainModel) renderDeleteServiceModal() string {
	focus := m.focusThemeColor()
	border := lipgloss.Color(m.palette.Border)
	serviceLabel := "selected"
	if m.deleteServiceIdx >= 0 && m.deleteServiceIdx < len(m.services) {
		serviceLabel = m.deleteTargetLabel(m.services[m.deleteServiceIdx])
	}

	yes := "Yes"
	no := "No"
	if m.deleteConfirmCursor == 0 {
		yes = lipgloss.NewStyle().Foreground(focus).Bold(true).Render("> Yes")
		no = "  No"
	} else {
		yes = "  Yes"
		no = lipgloss.NewStyle().Foreground(focus).Bold(true).Render("> No")
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(focus).Render("Delete Service"),
		fmt.Sprintf("Are you sure you want to delete %s service?", serviceLabel),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, yes, "    ", no),
	}
	if m.deleteError != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color(m.palette.StateError)).Render("Error: "+m.deleteError))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2).
		Width(64).
		Render(strings.Join(lines, "\n"))
}

func (m *MainModel) deleteTargetLabel(s config.ServiceDef) string {
	switch s.TypeOfService {
	case "docker":
		if s.Docker.ContainerName != "" {
			return s.Docker.ContainerName
		}
	case "systemd":
		if s.Systemd.Unit != "" {
			return s.Systemd.Unit
		}
	case "k8s":
		if s.K8s.Deployment != "" {
			return s.K8s.Deployment
		}
	}
	if s.Name != "" {
		return s.Name
	}
	return s.Id
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

func (m *MainModel) renderLogsPanel(panelStyle lipgloss.Style) string {
	w := panelStyle.GetWidth() - 2
	h := panelStyle.GetHeight() - 2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if m.logsViewport.Width == 0 && m.logsViewport.Height == 0 {
		m.logsViewport = viewport.New(w, h)
	} else {
		m.logsViewport.Width = w
		m.logsViewport.Height = h
	}

	content := m.logsContent
	serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
	if serviceIdx >= 0 && serviceIdx < len(m.services) {
		target := m.deleteTargetLabel(m.services[serviceIdx])
		content = "Selected: " + target + "\n\n" + m.logsContent
	}

	m.logsViewport.SetContent(wrapToWidth(content, w))
	return panelStyle.Render(m.logsViewport.View())
}

func (m *MainModel) cycleTheme(step int) {
	if len(m.themes) == 0 || step == 0 {
		return
	}

	idx := 0
	for i, p := range m.themes {
		if p.Name == m.selectedTheme {
			idx = i
			break
		}
	}
	idx = (idx + step + len(m.themes)) % len(m.themes)
	m.themeCursor = idx
	m.palette = m.themes[idx]
	m.selectedTheme = m.palette.Name
	_ = theme.SaveSelected(m.selectedTheme)
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

func (m *MainModel) syncServiceItem(serviceIdx int) {
	if serviceIdx < 0 || serviceIdx >= len(m.services) {
		return
	}
	s := m.services[serviceIdx]
	rt := m.runtimeByID[s.Id]
	item := m.buildServiceItem(s, rt)

	if serviceIdx < len(m.items) {
		m.items[serviceIdx] = item
	}
}

func (m *MainModel) refreshCard() {
	newItems := make([]string, 0, len(m.services))
	for _, s := range m.services {
		switch s.TypeOfService {
		case "docker":
			rt := docker.GetMetricsFromContainer(s.Docker.ContainerName)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, m.buildServiceItem(s, rt))
		case "systemd":
			rt := m.samplerStruct.GetSystemdMetrics(s.Id, s.Systemd.Unit)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, m.buildServiceItem(s, rt))
		case "k8s":
			rt := kubernetes.GetMetricsFromDeployment(s.K8s.Deployment, s.K8s.Namespace)
			m.runtimeByID[s.Id] = rt
			newItems = append(newItems, m.buildServiceItem(s, rt))
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

func (m *MainModel) buildServiceItem(s config.ServiceDef, rt model.ServiceRuntime) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	statusStyle := lipgloss.NewStyle().Foreground(m.serviceStateColor(rt)).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.palette.StateError)).Bold(true)

	name := s.Name
	if name == "" {
		name = s.Id
	}

	target := m.deleteTargetLabel(s)
	status := rt.Status
	if status == "" {
		status = "Unknown"
	}
	mem := strings.TrimSpace(rt.Mem)
	if mem == "" {
		mem = "0 B"
	}
	memLimit := strings.TrimSpace(rt.MemLimit)
	if memLimit == "" {
		memLimit = "No limit assigned"
	}
	uptime := strings.TrimSpace(rt.Uptime)
	if uptime == "" {
		uptime = "-"
	}

	errValue := "-"
	if strings.TrimSpace(rt.ErrorMsg) != "" {
		errValue = strings.TrimSpace(rt.ErrorMsg)
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(name),
		labelStyle.Render("Target: ") + valueStyle.Render(target),
		labelStyle.Render("Status: ") + statusStyle.Render(status),
		"",
		labelStyle.Render("CPU: ") + valueStyle.Render(strconv.FormatFloat(rt.Cpu, 'f', 2, 64)+"%"),
		labelStyle.Render("Memory: ") + valueStyle.Render(mem+" / "+memLimit),
		labelStyle.Render("Uptime: ") + valueStyle.Render(uptime),
	}

	if errValue == "-" {
		lines = append(lines, labelStyle.Render("Error: ")+valueStyle.Render(errValue))
	} else {
		lines = append(lines, labelStyle.Render("Error: ")+errorStyle.Render(errValue))
	}

	return strings.Join(lines, "\n")
}

func (m *MainModel) refreshLogsPreview() {
	serviceIdx := m.filteredServiceIndex(m.cursor, m.selectedType, m.selectedState)
	if serviceIdx < 0 || serviceIdx >= len(m.services) {
		m.logsContent = "No service selected."
		return
	}

	s := m.services[serviceIdx]
	var (
		logs string
		err  error
	)

	switch s.TypeOfService {
	case "docker":
		logs, err = docker.GetLogsFromContainer(s.Docker.ContainerName)
	case "systemd":
		logs, err = systemd.GetUnitLogs(s.Systemd.Unit)
	case "k8s":
		logs, err = kubernetes.GetLogsFromDeployment(s.K8s.Namespace, s.K8s.Deployment, 200)
	default:
		m.logsContent = "Logs not supported for this service."
		return
	}

	if err != nil {
		m.logsContent = "Error getting logs: " + err.Error()
		return
	}

	logs = strings.TrimSpace(lastNLines(logs, 120))
	if logs == "" {
		m.logsContent = "No logs available."
		return
	}
	m.logsContent = logs
}

func lastNLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func wrapToWidth(s string, width int) string {
	if width <= 1 {
		return s
	}

	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		r := []rune(line)
		if len(r) == 0 {
			out = append(out, "")
			continue
		}
		for len(r) > width {
			out = append(out, string(r[:width]))
			r = r[width:]
		}
		out = append(out, string(r))
	}

	return strings.Join(out, "\n")
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
			m.activeArea = kubeconfigFocus
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
			m.activeArea = kubeconfigFocus
		}
	case remoteConectionFocus:
		switch dir {
		case "up", "k":
			m.activeArea = filtersFocus
		case "right", "l":
			m.activeArea = servicesFocus
		case "down", "j":
			m.activeArea = kubeconfigFocus
		}
	case kubeconfigFocus:
		switch dir {
		case "up", "k":
			m.activeArea = filtersFocus
		case "right", "l":
			m.activeArea = servicesFocus
		case "down", "j":
			m.activeArea = logsFocus
		}
	case logsFocus:
		switch dir {
		case "up", "k":
			m.activeArea = kubeconfigFocus
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
		row := m.cursor / cols
		col := m.cursor % cols
		if row > 0 {
			prevStart := (row - 1) * cols
			prevEnd := prevStart + cols - 1
			if prevEnd >= total {
				prevEnd = total - 1
			}
			target := prevStart + col
			if target > prevEnd {
				target = prevEnd
			}
			m.cursor = target
		}
	case "down", "j":
		row := m.cursor / cols
		col := m.cursor % cols
		nextStart := (row + 1) * cols
		if nextStart < total {
			nextEnd := nextStart + cols - 1
			if nextEnd >= total {
				nextEnd = total - 1
			}
			target := nextStart + col
			if target > nextEnd {
				target = nextEnd
			}
			m.cursor = target
		}
	}
}
