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
	"fmt"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueueReadyPushPop(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	Convey("Given a ready SubQueue", t, func() {
		sq := newReadySubQueue()

		Convey("You can push() equal priority items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {})

			Convey("And then pop() them out in FIFO order", func() {
				testPopsInInsertionOrder(ctx, sq, num, ips)
			})
		})

		testPopsReverseOrder := func() {
			popItemsFromSubQueue(sq, num, func(key string, i int) {
				So(key, ShouldEqual, ips[num-1-i].Key)
			})
		}

		Convey("You can push() different priority items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				item.priority = uint8(i)
			})

			Convey("And then pop() them out in priority order", func() {
				testPopsReverseOrder()
			})
		})

		Convey("You can push() different size items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				item.size = uint8(i)
			})

			Convey("And then pop() them out in size order", func() {
				testPopsReverseOrder()
			})
		})

		Convey("Priority has precedence over size which has precedence over age", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				switch i {
				case 3:
					item.priority = uint8(3)
					item.size = uint8(4)
				case 4:
					item.priority = uint8(3)
					item.size = uint8(5)
				}
			})

			popItemsFromSubQueue(sq, num, func(key string, i int) {
				switch i {
				case 0:
					So(key, ShouldEqual, ips[4].Key)
				case 1:
					So(key, ShouldEqual, ips[3].Key)
				case 2, 3, 4:
					So(key, ShouldEqual, ips[i-2].Key)
				default:
					So(key, ShouldEqual, ips[i].Key)
				}
			})
		})
	})
}

func TestQueueReadyUpdate(t *testing.T) {
	ctx := context.Background()
	num := 6
	ips := newSetOfItemParameters(num)

	for i := 0; i < num; i++ {
		ips[i].Priority = 5
		ips[i].Size = 5
	}

	Convey("Given a ready SubQueue with some items push()ed to it", t, func() {
		sq := newReadySubQueue()
		items := make([]*Item, num)
		pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
			items[i] = item
		})
		So(sq.len(), ShouldEqual, num)

		update2p := func() {
			items[2].SetPriority(1)
		}
		update2s := func() {
			items[2].SetSize(2)
		}

		Convey("You can update() an item's priority, which changes pop() order", func() {
			update2p()
			So(sq.len(), ShouldEqual, num)
			testPopsInAlteredOrder(ctx, sq, num, ips)
		})

		Convey("You can update() an item's size, which changes pop() order", func() {
			update2s()
			So(sq.len(), ShouldEqual, num)
			testPopsInAlteredOrder(ctx, sq, num, ips)
		})

		Convey("You can do simultaneous update()s", func() {
			testSimultaneousUpdates(update2p, update2s)
			testPopsInAlteredOrder(ctx, sq, num, ips)
		})
	})
}

func TestQueueReady(t *testing.T) {
	num := 2
	ips := newSetOfItemParameters(num)
	ipsSameRG := newSetOfItemParameters(num)
	backgroundCtx := context.Background()

	Convey("Given a readyQueues", t, func() {
		rqs := newReadyQueues()

		Convey("Items with different ReserveGroups can be pushed to it simultaneously", func() {
			items := make([]*Item, num)

			for i := 0; i < num; i++ {
				ips[i].ReserveGroup = fmt.Sprintf("%d", i+1)
				items[i] = ips[i].toItem()
			}

			canDoConcurrently(num, func(i int) error {
				rqs.push(items[i-1])

				return nil
			})
			So(len(rqs.queues), ShouldEqual, num)
			So(rqs.numItems(), ShouldEqual, num)

			Convey("Then they can be simultaneously pop()ed", func() {
				okCh := make(chan bool, num)
				canDoConcurrently(num, func(i int) error {
					item := rqs.pop(backgroundCtx, ips[i-1].ReserveGroup)
					okCh <- item != nil && item.Key() == ips[i-1].Key

					return nil
				})

				So(len(rqs.queues), ShouldEqual, 0)
				So(rqs.numItems(), ShouldEqual, 0)

				for i := 0; i < num; i++ {
					ok := <-okCh
					So(ok, ShouldBeTrue)
				}

				item := rqs.pop(backgroundCtx, "")
				So(item, ShouldBeNil)
				So(rqs.inUse, ShouldEqual, 0)
			})

			Convey("Then they can be simultaneously remove()ed", func() {
				canDoConcurrently(num, func(i int) error {
					rqs.remove(items[i-1])

					return nil
				})

				So(len(rqs.queues), ShouldEqual, 0)
				So(rqs.numItems(), ShouldEqual, 0)
				item := rqs.pop(backgroundCtx, "")
				So(item, ShouldBeNil)
				So(rqs.inUse, ShouldEqual, 0)
			})

			Convey("You can change the ReserveGroup of items", func() {
				So(rqs.queues[ips[0].ReserveGroup].len(), ShouldEqual, 1)
				So(rqs.queues[ips[1].ReserveGroup].len(), ShouldEqual, 1)

				rqs.changeItemReserveGroup(items[0], ips[1].ReserveGroup)
				So(len(rqs.queues), ShouldEqual, 1)
				So(rqs.queues[ips[1].ReserveGroup].len(), ShouldEqual, 2)

				rqs.changeItemReserveGroup(items[0], ips[1].ReserveGroup)
				So(len(rqs.queues), ShouldEqual, 1)
				So(rqs.queues[ips[1].ReserveGroup].len(), ShouldEqual, 2)
				So(rqs.numItems(), ShouldEqual, num)
			})

			Convey("ReserveGroups can be changed simultaneously", func() {
				newGroup := "foo"
				canDoConcurrently(num, func(i int) error {
					rqs.changeItemReserveGroup(items[i-1], newGroup)

					return nil
				})

				So(len(rqs.queues), ShouldEqual, 1)
				So(rqs.queues[newGroup].len(), ShouldEqual, 2)
				item := rqs.pop(backgroundCtx, newGroup)
				So(item, ShouldNotBeNil)
				So(item, ShouldEqual, items[0])

				var failed bool
				for j := 0; j < 100; j++ {
					j := j
					canDoConcurrently(num, func(i int) error {
						rqs.changeItemReserveGroup(items[1], fmt.Sprintf("%d.%d", j, i))

						return nil
					})

					if len(rqs.queues) != 1 {
						failed = true

						break
					}
				}
				So(failed, ShouldBeFalse)
			})
		})

		Convey("Immediately after emptying a queue you can add items to it.", func() {
			var failed bool

			for j := 1; j < 10; j++ {
				for i := 0; i < num; i++ {
					rqs.push(ipsSameRG[i].toItem())
				}

				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					for i := 0; i < num; i++ {
						rqs.pop(backgroundCtx, "")
					}
				}()

				go func() {
					defer wg.Done()
					for i := 0; i < num; i++ {
						rqs.push(ipsSameRG[i].toItem())
					}
				}()

				wg.Wait()
				for i := 0; i < num; i++ {
					item := rqs.pop(backgroundCtx, "")
					if item == nil || item.Key() != ipsSameRG[i].Key {
						failed = true
					}
				}

				if failed {
					break
				}
			}

			So(failed, ShouldBeFalse)
		})

		Convey("SubQueues are not dropped while we are in use", func() {
			rqs.queues["foo"] = newReadySubQueue()
			rqs.dropEmptyQueuesIfNotInUse()
			So(len(rqs.queues), ShouldEqual, 0)

			rqs.queues["foo"] = newReadySubQueue()
			rqs.inUse = 2
			rqs.dropEmptyQueuesIfNotInUse()
			So(len(rqs.queues), ShouldEqual, 1)
		})
	})
}
