package empire

import (
	"fmt"
	"log"
	"strings"
)

// RunEvent is triggered when a user starts a one off process.
type RunEvent struct {
	User     string
	App      string
	Command  string
	Attached bool
}

func (e RunEvent) Event() string {
	return "run"
}

func (e RunEvent) String() string {
	attachment := "detached"
	if e.Attached {
		attachment = "attached"
	}
	return fmt.Sprintf("%s ran `%s` (%s) on %s", e.User, e.Command, attachment, e.App)
}

// RestartEvent is triggered when a user restarts an application.
type RestartEvent struct {
	User string
	App  string
	PID  string
}

func (e RestartEvent) Event() string {
	return "restart"
}

func (e RestartEvent) String() string {
	if e.PID == "" {
		return fmt.Sprintf("%s restarted %s", e.User, e.App)
	}

	return fmt.Sprintf("%s restarted `%s` on %s", e.User, e.PID, e.App)
}

// ScaleEvent is triggered when a manual scaling event happens.
type ScaleEvent struct {
	User     string
	App      string
	Process  string
	Quantity int
}

func (e ScaleEvent) Event() string {
	return "scale"
}

func (e ScaleEvent) String() string {
	return fmt.Sprintf("%s scaled `%s` on %s to %d", e.User, e.Process, e.App, e.Quantity)
}

// DeployEvent is triggered when a user deploys a new image to an app.
type DeployEvent struct {
	User  string
	App   string
	Image string
}

func (e DeployEvent) Event() string {
	return "deploy"
}

func (e DeployEvent) String() string {
	if e.App == "" {
		return fmt.Sprintf("%s deployed %s", e.User, e.Image)
	}

	return fmt.Sprintf("%s deployed %s to %s", e.User, e.Image, e.App)
}

// SetEvent is triggered when environment variables are changed on an
// application.
type SetEvent struct {
	User    string
	App     string
	Changed []string
}

func (e SetEvent) Event() string {
	return "set"
}

func (e SetEvent) String() string {
	return fmt.Sprintf("%s changed environment variables on %s (%s)", e.User, e.App, strings.Join(e.Changed, ", "))
}

// RollbackEvent is triggered when a user rolls back to an old version.
type RollbackEvent struct {
	User    string
	App     string
	Version int
}

func (e RollbackEvent) Event() string {
	return "rollback"
}

func (e RollbackEvent) String() string {
	return fmt.Sprintf("%s rolled back %s to v%d", e.User, e.App, e.Version)
}

// CreateEvent is triggered when a user creates a new application.
type CreateEvent struct {
	User string
	Name string
}

func (e CreateEvent) Event() string {
	return "create"
}

func (e CreateEvent) String() string {
	return fmt.Sprintf("%s created %s", e.User, e.Name)
}

// Event represents an event triggered within Empire.
type Event interface {
	// Returns the name of the event.
	Event() string

	// Returns a human readable string about the event.
	String() string
}

// EventStream is an interface for publishing events that happen within
// Empire.
type EventStream interface {
	PublishEvent(Event) error
}

// EventStreamFunc is a function that implements the Events interface.
type EventStreamFunc func(Event) error

func (fn EventStreamFunc) PublishEvent(event Event) error {
	return fn(event)
}

// NullEventStream an events service that does nothing.
var NullEventStream = EventStreamFunc(func(event Event) error {
	return nil
})

// AsyncEventStream wraps an EventStream to publish events asynchronously in a
// goroutine.
type AsyncEventStream struct {
	EventStream
	events chan Event
}

// AsyncEvents returns a new AsyncEventStream that will buffer upto 100 events
// before applying backpressure.
func AsyncEvents(e EventStream) *AsyncEventStream {
	s := &AsyncEventStream{
		EventStream: e,
		events:      make(chan Event, 100),
	}
	go s.start()
	return s
}

func (e *AsyncEventStream) PublishEvent(event Event) error {
	e.events <- event
	return nil
}

func (e *AsyncEventStream) start() {
	for event := range e.events {
		err := e.publishEvent(event)
		if err != nil {
			log.Printf("event stream error: %v\n", err)
		}
	}
}

func (e *AsyncEventStream) publishEvent(event Event) (err error) {
	defer func() {
		if v := recover(); v != nil {
			var ok bool
			if err, ok = v.(error); ok {
				return
			}

			err = fmt.Errorf("panic: %v", v)
		}
	}()

	err = e.EventStream.PublishEvent(event)
	return
}
