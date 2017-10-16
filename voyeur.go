/*
   This file is part of voyeur.

   voyeur is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   voyeur is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with voyeur.  If not, see <http://www.gnu.org/licenses/>.
*/

package voyeur

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

func init() {
	Noop = Map(func(ctx context.Context, em Emitter, e Event) {})
	Fwd = Map(func(ctx context.Context, em Emitter, e Event) { em.Emit(ctx, e) })
}

var Noop, Fwd Filter

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

// ObserverFunc is an observer that calls the function for each event
type ObserverFunc func(context.Context, Event)

func (o ObserverFunc) OnEvent(ctx context.Context, e Event) {
	o(ctx, e)
}

// Observable  emits events
type Observable interface {
	Register(context.Context, Observer)
}

// FilterBuilder is a function taking an arbitrary number of inputs and returns a Filter
type FilterBuilder struct {
	v interface{}
}

func NewFilterBuilder(f interface{}) *FilterBuilder {
	return &FilterBuilder{f}
}

// Valid returns whether FilterBuilder is a valid FilterBuilder
func (fb *FilterBuilder) Valid() bool {
	t := reflect.TypeOf(fb.v)
	if t.Kind() != reflect.Func {
		return false
	}

	if t.NumOut() != 1 {
		return false
	}

	if t.Out(0) != reflect.TypeOf(Filter(nil)) {
		return false
	}

	return true
}

type paramMismatchError struct {
	t  reflect.Type
	vs []interface{}
}

func newParamMismatchError(t reflect.Type, vs interface{}) error {
	return paramMismatchError{
		t:  t,
		vs: vs.([]interface{}),
	}
}

func (err paramMismatchError) Error() string {
	gotTypeStrs := make([]string, len(err.vs))
	expTypeStrs := make([]string, err.t.NumIn())

	for i, v := range err.vs {
		t := reflect.TypeOf(v)
		gotTypeStrs[i] = fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	}

	for i := 0; i < err.t.NumIn(); i++ {
		t := err.t.In(i)
		gotTypeStrs[i] = fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	}

	return fmt.Sprintf("parameter mismatch error: got (%s), expected (%s)", strings.Join(gotTypeStrs, " "), strings.Join(expTypeStrs, " "))
}

func (fb *FilterBuilder) Build(vs ...interface{}) (Filter, error) {
	t := reflect.TypeOf(fb.v)

	if len(vs) != t.NumIn() {
		return nil, newParamMismatchError(t, vs)
	}

	for i := range vs {
		tv := reflect.TypeOf(vs[i])
		tp := t.In(i)
		if tv != tp {
			return nil, newParamMismatchError(t, vs)
		}
	}

	var rvs = make([]reflect.Value, len(vs))

	for i, v := range vs {
		rvs[i] = reflect.ValueOf(v)
	}

	// spread this over several lines, because each of these may, but shouldn't panic.
	// the resulting stack trace will be more informative/easier to read.
	out := reflect.ValueOf(fb.v).Call(rvs)
	filterValue := out[0]
	filterIface := filterValue.Interface()

	return filterIface.(Filter), nil
}

// Filter is bot observer and observant. A generalization of map, filter and reduce.
type Filter interface {
	Observable
	Observer
}

type filter struct {
	Observable
	Observer
}

// NewFilter constructs a Filter from an observable and an observer.
func NewFilter(o Observable, oer Observer) Filter {
	return filter{Observable: o, Observer: oer}
}

// Emitter lets you send events
type Emitter interface {
	Emit(context.Context, Event)
	End(context.Context)
}

type observable struct {
	done      chan struct{}
	lock      sync.Mutex
	observers map[*Observer]struct{}
}

type emitter observable

func (o *observable) Register(ctx context.Context, oer Observer) {
	func() {
		o.lock.Lock()
		defer o.lock.Unlock()
		o.observers[&oer] = struct{}{}
	}()
	go func() {
		select {
		case <-o.done:
		case <-ctx.Done():
			o.lock.Lock()
			defer o.lock.Unlock()
			delete(o.observers, &oer)
		}
	}()
}

func (em *emitter) Emit(ctx context.Context, e Event) {
	em.lock.Lock()
	defer em.lock.Unlock()

	for o := range em.observers {
		(*o).OnEvent(ctx, e)
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
		observers: make(map[*Observer]struct{}),
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

type mapFilter struct {
	o  Observable
	em Emitter

	f func(context.Context, Emitter, Event)
}

func Map(f func(context.Context, Emitter, Event)) Filter {
	em, o := Pair()
	return &mapFilter{
		o:  o,
		em: em,
		f:  f,
	}
}

func (m *mapFilter) OnEvent(ctx context.Context, e Event) {
	m.f(ctx, m.em, e)
}

func (m *mapFilter) Register(ctx context.Context, oer Observer) {
	m.o.Register(ctx, oer)
}
