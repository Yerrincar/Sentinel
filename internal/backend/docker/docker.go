package docker

type state int

const (
	running state = iota
	stopped
	serviceError
	unknown
)

type ServiceRuntime struct {
	cpu      int
	mem      int
	status   state
	uptime   int
	errorMsg string
}
