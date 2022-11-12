package event

// Nop is an event Manager that does nothing.
var Nop Manager = &nopMgr{}

type nopMgr struct{}

func (n *nopMgr) Subscribe(eventType Event, priority int, fn HandlerFunc) (unsubscribe func()) {
	return func() {}
}
func (n *nopMgr) Wait(events ...Event)               {}
func (n *nopMgr) HasSubscriber(events ...Event) bool { return false }
func (n *nopMgr) UnsubscribeAll(events ...Event) int { return 0 }
func (n *nopMgr) Fire(Event)                         {}
func (n *nopMgr) FireParallel(Event, ...HandlerFunc) {}
