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
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueueExpire(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	delay := 20 * time.Millisecond
	ctx := context.Background()

	for i := 0; i < num; i++ {
		ips[i].Delay = delay + time.Duration(i)*time.Nanosecond
	}

	Convey("Given a ready-based expire SubQueue with some items push()ed to it", t, func() {
		items := make([]*Item, num*2)
		expiredItems := 0
		var eiMutex sync.RWMutex
		itemCh := make(chan *Item, num*2)
		ecb := func(item *Item) bool {
			eiMutex.Lock()
			expiredItems++
			eiMutex.Unlock()
			itemCh <- item

			return true
		}

		numExpired := func() int {
			eiMutex.RLock()
			ei := expiredItems
			eiMutex.RUnlock()

			return ei
		}

		sq := newExpireSubQueue(ecb, getItemReady, newReadyOrder())
		pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
			item.restart()
			items[i] = item
		})
		So(sq.len(), ShouldEqual, num)
		So(numExpired(), ShouldEqual, 0)

		firstExpire := items[0].ReadyAt()
		beforeFirstExpire := firstExpire.Add(-1 * time.Millisecond)
		afterFirstExpire := firstExpire.Add(50 * time.Millisecond)

		Convey("After delay, the items get sent to our callback", func() {
			<-time.After(time.Until(beforeFirstExpire))
			So(numExpired(), ShouldEqual, 0)

			<-time.After(time.Until(afterFirstExpire))
			So(numExpired(), ShouldBeBetweenOrEqual, 1, num)

			for i := 0; i < num; i++ {
				item := <-itemCh
				So(item.Key(), ShouldEqual, items[i].Key())
			}

			So(numExpired(), ShouldEqual, num)
		})

		Convey("You can push new items while others expire", func() {
			ipsNew := newSetOfItemParameters(num)

			for i := 0; i < num; i++ {
				ipsNew[i].Key = ips[i].Key + ".new"
				ipsNew[i].Delay = delay
			}

			item := <-itemCh
			if item == nil {
				So(false, ShouldBeTrue)
			}

			pushItemsToSubQueue(sq, ipsNew, func(item *Item, i int) {
				item.restart()
				items[i+num] = item
			})
			So(sq.len(), ShouldBeLessThanOrEqualTo, num*2)

			for i := 0; i < num*2-1; i++ {
				item := <-itemCh
				So(item.Key(), ShouldEqual, items[i+1].Key())
			}

			So(numExpired(), ShouldEqual, num*2)
		})

		Convey("You can pop items while others expire", func() {
			item := <-itemCh
			if item == nil {
				So(false, ShouldBeTrue)
			}

			popped := 0
			for i := 0; i < num; i++ {
				item := sq.pop(ctx)
				if item == nil {
					break
				}
				popped++
			}

			for i := 0; i < num-popped-1; i++ {
				<-itemCh
			}

			So(popped, ShouldBeBetweenOrEqual, 0, num)
			So(numExpired(), ShouldEqual, num-popped)
		})
	})
}
