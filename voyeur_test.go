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
	"time"
)

// printObserver simply prints all received events to stdout
type printObserver struct{}

func (o printObserver) OnEvent(ctx context.Context, e Event) {
	fmt.Println(e)
}

// stringEvent is a very simple event type
type stringEvent string

func (e stringEvent) EventType() string {
	return "string"
}

func (e stringEvent) String() string {
	return string(e)
}

func Example() {
	// build a context we can cancel
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	em, o := Pair()

	// create and register a very simple observer. It just prints events to stdout.
	printer := printObserver{}
	o.Register(ctx, printer)

	// emit some events.
	em.Emit(ctx, stringEvent("test"))
	em.Emit(ctx, stringEvent("foo"))

	// cancel the observation. This only affects observers that use the context returned by context.WithCancel(ctx)
	cancel()
	time.Sleep(time.Millisecond) // kick scheduler, otherwise the observer will not be removed before we emit the next event

	// this event is not seen by the observer anymore, because we cancelled the observation before.
	em.Emit(ctx, stringEvent("bar"))

	// let's reregister, this time without being able to cancel
	ctx = context.Background()
	o.Register(ctx, printer)

	em.End(ctx)

	// Output:
	// test
	// foo
	// End
}

func ExampleFilter() {
	var strEv stringEvent
	ctx := context.Background()

	m := Map(func(ctx context.Context, em Emitter, e Event) {
		strEv += e.(stringEvent)
		em.Emit(ctx, strEv)
	})

	em, o := Pair()

	o.Register(ctx, m)
	m.Register(ctx, printObserver{})

	em.Emit(ctx, stringEvent("a"))
	em.Emit(ctx, stringEvent("b"))
	em.Emit(ctx, stringEvent("c"))

	// Output:
	// a
	// ab
	// abc
}

func ExampleFilterBuilder() {
	fb := NewFilterBuilder(
		func(l int) Filter {
			return Map(func(ctx context.Context, em Emitter, e Event) {
				if se, ok := e.(stringEvent); ok && len(se) == l {
					em.Emit(ctx, e)
				}
			})
		})

	// only forward events of length 4
	m, err := fb.Build(4)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()

	em, o := Pair()

	o.Register(ctx, m)
	m.Register(ctx, printObserver{})

	em.Emit(ctx, stringEvent("1"))
	em.Emit(ctx, stringEvent("12"))
	em.Emit(ctx, stringEvent("123"))
	em.Emit(ctx, stringEvent("1234"))
	em.Emit(ctx, stringEvent("12345"))
	em.Emit(ctx, stringEvent("123456"))

	em.Emit(ctx, stringEvent("a"))
	em.Emit(ctx, stringEvent("ab"))
	em.Emit(ctx, stringEvent("abc"))
	em.Emit(ctx, stringEvent("abcd"))
	em.Emit(ctx, stringEvent("abcde"))
	em.Emit(ctx, stringEvent("abcdef"))

	// Output:
	// 1234
	// abcd
}
