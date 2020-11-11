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

func TestQueueReady(t *testing.T) {
	num := 2
	ips := newSetOfItemParameters(num)
	backgroundCtx := context.Background()

	Convey("Given a readyQueues", t, func() {
		rqs := newReadyQueues()
		items := make([]*Item, num)

		for i := 0; i < num; i++ {
			ips[i].ReserveGroup = fmt.Sprintf("%d", i+1)
			items[i] = ips[i].toItem()
		}

		Convey("Items with different ReserveGroups can be pushed to it simultaneously", func() {
			canDoConcurrently(num, func(i int) error {
				rqs.push(items[i-1])

				return nil
			})
			So(len(rqs.queues), ShouldEqual, num)

			Convey("Then they can be simultaneously pop()ed", func() {
				okCh := make(chan bool, num)
				canDoConcurrently(num, func(i int) error {
					item := rqs.pop(backgroundCtx, ips[i-1].ReserveGroup)
					okCh <- item != nil && item.Key() == ips[i-1].Key

					return nil
				})

				So(len(rqs.queues), ShouldEqual, 0)

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
				item := rqs.pop(backgroundCtx, "")
				So(item, ShouldBeNil)
				So(rqs.inUse, ShouldEqual, 0)
			})

			Convey("You can change the ReserveGroup of items", func() {

			})
		})

		Convey("Immediately after emptying a queue you can add items to it.", func() {
			var failed bool

			for j := 1; j < 10; j++ {
				ipsSameRG := newSetOfItemParameters(num)
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
	})
}
