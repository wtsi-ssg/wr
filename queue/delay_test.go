/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

package queue

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueueDelayPushPop(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	Convey("Given a delay SubQueue", t, func() {
		sq := newDelaySubQueue(func(*Item) (bool, chan struct{}) {
			return true, make(chan struct{})
		})

		Convey("You can push() increasing readyAt items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				item.restart()
			})

			Convey("And then pop() them out in readyAt order", func() {
				testPopsInInsertionOrder(ctx, sq, num, ips)
			})
		})

		SkipConvey("You can push() reversed readyAt items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				item.readyAt = time.Now().Add(time.Duration(num-i) * time.Millisecond)
			})

			Convey("And then pop() them out in readyAt order", func() {
				popItemsFromSubQueue(sq, num, func(key string, i int) {
					So(key, ShouldEqual, ips[num-1-i].Key)
				})
			})
		})
	})
}

func TestQueueDelayUpdate(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	SkipConvey("Given a delay SubQueue with some items push()ed to it", t, func() {
		sq := newDelaySubQueue(func(*Item) (bool, chan struct{}) {
			return true, make(chan struct{})
		})
		items := make([]*Item, num)
		pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
			item.restart()
			items[i] = item
		})
		So(sq.len(), ShouldEqual, num)

		update := func() {
			items[2].restart()
		}

		Convey("You can restart() an item, which changes pop() order", func() {
			update()
			So(sq.len(), ShouldEqual, num)
			testPopsInAlteredOrder(ctx, sq, num, ips)
		})

		Convey("You can do simultaneous update()s", func() {
			testSimultaneousUpdates(update, update)
			testPopsInAlteredOrder(ctx, sq, num, ips)
		})
	})
}
