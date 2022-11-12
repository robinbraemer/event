# event

Go event library to subscribe and fire events used to decouple the event
source and sink in complex or ad-don based systems.

```go
// Custom event type
type MyEvent struct {
    Message string
}

// Create a new event manager.
mgr := New()

// Subscribe to the event.
Subscribe(mgr, 0, func(e *MyEvent) {
    // Handle the event.
    fmt.Println("handler A received event:", e.Message)
})

// Subscribe to the event higher priority.
Subscribe(mgr, 1, func(e *MyEvent) {
    // Handle the event.
    fmt.Println("handler B received event:", e.Message)
    e.Message = "hello gophers"
})

unsubscribe := Subscribe(mgr, 0, func(e *MyEvent) {
    // Handle the event.
    fmt.Println("handler C received event:", e.Message)
})
// Unsubscribe from the event if you want to or ignore and never call it.
unsubscribe()

// Fire the event.
mgr.Fire(&MyEvent{Message: "hello world"})
```

## Used by

- [Gate](https://github.com/minekube/gate)

