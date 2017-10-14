package voyeur

import (
	"context"
	"sync"
)

// End is like EOF or a channel close. Nothing to see here anymore.
var End = simpleEvent{"End"}

// Event describes an event.
type Event interface {
	// EventType returns a short descriptor for the event
	EventType() string

	// Context returns a context assiciated with this Event
	//Context() context.Context
}

// Observer consumes events
type Observer interface {
	OnEvent(context.Context, Event)
}

// Observable  emits events
type Observable interface {
	Register(context.Context, Observer)
}

// Emitter lets you send events
type Emitter interface {
	Emit(context.Context, Event)
	End(context.Context)
}

type observable struct {
	done      chan struct{}
	lock      sync.Mutex
	observers map[Observer]struct{}
}

type emitter observable

func (o *observable) Register(ctx context.Context, oer Observer) {
	func() {
		o.lock.Lock()
		defer o.lock.Unlock()
		o.observers[oer] = struct{}{}
	}()
	go func() {
		select {
		case <-o.done:
		case <-ctx.Done():
			o.lock.Lock()
			defer o.lock.Unlock()
			delete(o.observers, oer)
		}
	}()
}

func (em *emitter) Emit(ctx context.Context, e Event) {
	em.lock.Lock()
	defer em.lock.Unlock()

	for o := range em.observers {
		o.OnEvent(ctx, e)
	}

	if e == End {
		close(em.done)
	}
}

func (em *emitter) End(ctx context.Context) {
	em.Emit(ctx, End)
}

// Pair returns an Emitter and corresponding Observable. Events emitted on one can be observed on the other.
func Pair() (Emitter, Observable) {
	o := &observable{
		done:      make(chan struct{}),
		observers: make(map[Observer]struct{}),
	}

	em := (*emitter)(o)
	return em, o
}

type simpleEvent struct {
	typ string
}

func (e simpleEvent) EventType() string {
	return e.typ
}

func (e simpleEvent) String() string {
	return e.EventType()
}

func (e simpleEvent) Context() context.Context {
	return context.Background()
}
