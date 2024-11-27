package event

import (
	"reflect"
	"sort"
	"sync"

	"github.com/go-logr/logr"
)

// manager implements Manager interface.
type manager struct {
	activeSubscribers sync.WaitGroup // Wait for all active subscribers
	log               logr.Logger
	recoverPanic      bool

	mu          sync.RWMutex             // Protects following fields
	subscribers map[Type]*subscriberList // Event type to subscribers
}

type subscriberList struct {
	subs []*subscriber  // Subscribers sorted by priority
	wg   sync.WaitGroup // Wait for active subscribers in list
}

// subscriber is a subscriber to an event.
type subscriber struct {
	priority int         // The higher the priority, the earlier the subscriber is called.
	fn       HandlerFunc // The event handler func.
}

func (m *manager) Wait(events ...Event) {
	if len(events) == 0 {
		m.activeSubscribers.Wait()
		return
	}

	m.mu.RLock()
	subs := m.subscribers
	m.mu.RUnlock()

	for _, event := range events {
		eventType := typeOf(event)
		list, ok := subs[eventType]
		if ok {
			list.wg.Wait()
		}
	}
}

func (m *manager) HasSubscriber(events ...Event) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(events) == 0 {
		return len(m.subscribers) != 0
	}
	if m.subscribers[anyType] != nil {
		return true
	}
	for _, event := range events {
		if m.subscribers[typeOf(event)] != nil {
			return true
		}
	}
	return false
}

func (m *manager) UnsubscribeAll(events ...Event) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int
	if len(events) == 0 {
		for _, list := range m.subscribers {
			count += len(list.subs)
		}
		m.subscribers = make(map[Type]*subscriberList)
		return count
	}

	for _, event := range events {
		eventType := typeOf(event)
		list, ok := m.subscribers[eventType]
		if !ok {
			continue
		}
		count += len(list.subs)
		delete(m.subscribers, eventType)
	}
	return count
}

func (m *manager) Subscribe(eventType Event, priority int, fn HandlerFunc) (unsubscribe func()) {
	return m.subscribe(typeOf(eventType), priority, fn)
}
func (m *manager) subscribe(eventType Type, priority int, fn HandlerFunc) (unsubscribe func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub := &subscriber{
		priority: priority,
		fn:       fn,
	}

	// Get-add subscriber list for event type
	list, ok := m.subscribers[eventType]
	if ok {
		list.subs = append(list.subs, sub)
		// Sort subscribers by priority
		sort.Slice(list.subs, func(i, j int) bool {
			return list.subs[i].priority > list.subs[j].priority
		})
	} else {
		m.subscribers[eventType] = &subscriberList{subs: []*subscriber{sub}}
	}

	// Unsubscribe func
	var once sync.Once
	return func() { once.Do(func() { m.unsubscribe(eventType, sub) }) }
}

func (m *manager) unsubscribe(eventType Type, sub *subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list, ok := m.subscribers[eventType]
	if !ok {
		return
	}
	if len(list.subs) == 1 {
		delete(m.subscribers, eventType)
		return
	}
	for i, s := range list.subs {
		if s != sub { // Find by pointer
			continue
		}
		// Delete subscriber from list while maintaining the order.
		copy(list.subs[i:], list.subs[i+1:]) // Shift list[i+1:] left one index.
		list.subs[len(list.subs)-1] = nil    // Erase last element (write zero value).
		list.subs = list.subs[:len(list.subs)-1]
		return
	}
}

func (m *manager) FireParallel(event Event, after ...HandlerFunc) {
	m.activeSubscribers.Add(1)
	go func() {
		defer m.activeSubscribers.Done()
		m.fire(event)

		var i int
		if m.recoverPanic {
			defer func() {
				if r := recover(); r != nil {
					m.log.Error(nil,
						"recovered from panic by an 'after fire' func",
						"panic", r,
						"eventType", typeOf(event),
						"index", i)
				}
			}()
		}

		var fn HandlerFunc
		for i, fn = range after {
			fn(event)
		}
	}()
}

func (m *manager) Fire(event Event) {
	m.activeSubscribers.Add(1)
	defer m.activeSubscribers.Done()
	m.fire(event)
}

var anyType = typeOf(any(nil))

func (m *manager) fire(event Event) {
	eventType := typeOf(event)

	m.mu.RLock()
	list := m.subscribers[eventType]
	anyList := m.subscribers[anyType]
	m.mu.RUnlock()

	m.fireSubscribers(event, anyList)
	m.fireSubscribers(event, list)
}

func (m *manager) fireSubscribers(event Event, list *subscriberList) {
	if list == nil {
		return
	}
	list.wg.Add(1)
	defer list.wg.Done()

	for _, sub := range list.subs {
		m.callSubscriber(sub, event)
	}
}

func (m *manager) callSubscriber(sub *subscriber, event Event) {
	if m.recoverPanic {
		defer func() {
			if r := recover(); r != nil {
				m.log.Error(nil, "recovered from panic from an event subscriber",
					"panic", r,
					"eventType", typeOf(event),
					"subscriberPriority", sub.priority)
			}
		}()
	}
	sub.fn(event)
}

// typeOf returns the reflect.Type of e.
func typeOf(e Event) (t Type) {
	switch o := e.(type) {
	case reflect.Type:
		t = o
	case reflect.Value:
		t = o.Type()
	default:
		t = reflect.TypeOf(e)
	}
	return t
}
