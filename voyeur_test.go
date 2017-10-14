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
type printObserver struct {}

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

