package event

import (
	"reflect"

	"github.com/go-logr/logr"
)

// Manager is an event manager to subscribe, fire events and
// decouple the event source and sink in a complex system.
type Manager interface {
	// Subscribe subscribes a handler to an event type with a priority
	// and returns a func that can be run to unsubscribe the handler.
	//
	// HandlerFunc should return as soon as possible and start long-running tasks in parallel.
	// The Type can be any type, pointer to type or reflect.Type and the handler is only run for
	// the exact type subscribed for.
	//
	// HandlerFunc always gets the fired event of the same subscribed eventType or the same type as
	// represented by reflect.Type.
	Subscribe(eventType Event, priority int, fn HandlerFunc) (unsubscribe func())

	// Fire fires an event in the calling goroutine and returns after all subscribers are complete handling it.
	// Any panic by a subscriber is caught so firing the event to the next subscriber can proceed.
	Fire(Event)
	// FireParallel fires an event in a new goroutine and returns immediately.
	// The subscribers are called in order of priority and the event value is passed to the next subscriber.
	//
	// It optionally runs handlers in the goroutine after all subscribers are done.
	// If an after handler panics no further handlers in the slice are run.
	FireParallel(event Event, after ...HandlerFunc)

	// Wait blocks until no event handlers are running for the specified events.
	// If no events are specified it waits for all events.
	Wait(events ...Event)

	// HasSubscriber determines whether all given events have at least one subscriber.
	// If no events are specified it returns true if there are any subscribers for any event.
	//
	// It is useful to check whether an event is subscribed for before firing it when
	// the event value is expensive to create.
	HasSubscriber(events ...Event) bool
	// UnsubscribeAll unsubscribes all subscribers of the given events
	// and returns the number of subscribers unsubscribed.
	UnsubscribeAll(events ...Event) int
}

// Subscribe subscribes a handler to an event type with a priority.
// The event type is inferred from the argument of the handler.
// See Manager.Subscribe for more details.
func Subscribe[T Event](mgr Manager, priority int, handler func(T)) (unsubscribe func()) {
	var typ T
	return mgr.Subscribe(typ, priority, func(e Event) { handler(e.(T)) })
}

// FireParallel fires an event in a new goroutine and returns immediately.
// The subscribers are called in order of priority and the event value is passed to the next subscriber.
//
// It optionally runs handlers in the goroutine after all subscribers are done.
// If an after handler panics no further handlers in the slice are run.
func FireParallel[T Event](mgr Manager, event T, after ...func(T)) {
	mgr.FireParallel(event, func(e Event) {
		ev := e.(T)
		for _, fn := range after {
			fn(ev)
		}
	})
}

// FireParallelChan fires an event in a new goroutine and returns a result channel immediately.
// The subscribers are called in order of priority and the event value is passed to the next subscriber.
func FireParallelChan[T Event](mgr Manager, event T) (resultChan <-chan T) {
	result := make(chan T, 1)
	FireParallel(mgr, event, func(e T) {
		result <- event
		close(result)
	})
	return result
}

// HandlerFunc is an event handler.
type HandlerFunc func(e Event)

// Event is the event interface.
type Event any

// Type is an event type.
type Type reflect.Type

// New returns a new event Manager.
func New(opts ...ManagerOption) Manager {
	m := &manager{
		subscribers:  make(map[Type]*subscriberList),
		recoverPanic: true,
		log:          logr.Discard(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ManagerOption is a Manager option for New.
type ManagerOption func(*manager)

// WithRecoverPanic returns a ManagerOption that enables/disables panic recovery.
// Default is true.
func WithRecoverPanic(enabled bool) ManagerOption {
	return func(m *manager) {
		m.recoverPanic = enabled
	}
}

// WithLogger returns a ManagerOption that sets the logger.
// Default is logr.Discard().
func WithLogger(log logr.Logger) ManagerOption {
	return func(m *manager) {
		m.log = log
	}
}
