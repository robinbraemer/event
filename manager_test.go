package event

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type myEvent struct{ s string }

func TestTypeOf(t *testing.T) {
	assert.Equal(t, typeOf(&myEvent{}), reflect.TypeOf(&myEvent{}))
	assert.Equal(t, typeOf(reflect.TypeOf(&myEvent{})), reflect.TypeOf(&myEvent{}))
	assert.NotEqual(t, typeOf(reflect.TypeOf(myEvent{})), reflect.TypeOf(&myEvent{}))
}

func TestPriorityAndCorrectType(t *testing.T) {
	m := New()
	require.False(t, m.HasSubscriber(&myEvent{}))

	var calledAny int
	m.Subscribe(any(nil), 10, func(e Event) {
		calledAny++
	})

	require.True(t, m.HasSubscriber(&myEvent{}))

	m.Subscribe(typeOf(&myEvent{}), -1, func(e Event) {
		ev := e.(*myEvent)
		ev.s += "c"
	})
	m.Subscribe(&myEvent{}, 1, func(e Event) {
		ev := e.(*myEvent)
		ev.s += "a"
	})
	m.Subscribe(typeOf(myEvent{}), 0, func(e Event) {
		ev := e.(myEvent)
		ev.s += "d"
	})
	Subscribe(m, 0, func(ev *myEvent) {
		ev.s += "b"
	})

	var noPtr bool
	m.Subscribe(myEvent{}, 2, func(e Event) {
		_ = e.(myEvent)
		noPtr = true
	})

	e := &myEvent{s: "_"}
	m.Fire(e)
	require.False(t, noPtr)
	require.Equal(t, "_abc", e.s)
	require.True(t, m.HasSubscriber(&myEvent{}))
	require.Equal(t, calledAny, 1)
}
