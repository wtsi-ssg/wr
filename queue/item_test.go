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
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type mockSubQueue struct {
	updates int
}

func (sq *mockSubQueue) update(item *Item) {
	sq.updates++
}

func TestQueueItem(t *testing.T) {
	Convey("You can make items from ItemParameters", t, func() {
		before := time.Now()

		key, data := "key", "data"
		ip := &ItemParameters{
			Key:  key,
			Data: data,
		}

		item := ip.toItem()

		So(item.Key(), ShouldEqual, key)
		So(item.ReserveGroup(), ShouldEqual, "")
		So(item.Data(), ShouldEqual, data)
		So(item.Created(), ShouldHappenAfter, before)
		So(item.Priority(), ShouldEqual, 0)
		So(item.Size(), ShouldEqual, 0)

		p, s := uint8(5), uint8(3)
		ip = &ItemParameters{
			Key:          key,
			ReserveGroup: "rg",
			Data:         data,
			Priority:     p,
			Size:         s,
		}

		item = ip.toItem()

		So(item.ReserveGroup(), ShouldEqual, "rg")
		So(item.Priority(), ShouldEqual, p)
		So(item.Size(), ShouldEqual, s)
		So(item.index(), ShouldEqual, 0)

		remove := func() {
			item.remove()
			So(item.removed(), ShouldBeTrue)
			So(item.index(), ShouldEqual, indexOfRemovedItem)
			So(item.subQueue, ShouldBeNil)
		}

		Convey("You can add the item to a subQueue", func() {
			sq := &mockSubQueue{}
			item.setSubqueue(sq)
			So(item.subQueue, ShouldEqual, sq)

			Convey("When you set priority or size, the subQueue is updated", func() {
				new := uint8(10)
				item.SetPriority(new)
				So(item.Priority(), ShouldEqual, new)
				So(sq.updates, ShouldEqual, 1)

				item.SetSize(new)
				So(item.Size(), ShouldEqual, new)
				So(sq.updates, ShouldEqual, 2)
			})

			Convey("And then you can remove() it", func() {
				remove()
			})
		})

		Convey("You can remove() the item even when not in a subQueue", func() {
			remove()

			Convey("As well as set its properties", func() {
				new := uint8(10)
				item.SetPriority(new)
				So(item.Priority(), ShouldEqual, new)
			})
		})

		Convey("You can set and get item properties simultaneously", func() {
			sq := &mockSubQueue{}

			var wg sync.WaitGroup
			wg.Add(5)

			change := func(p, s uint8, index int, data interface{}) {
				defer wg.Done()
				item.SetReserveGroup("rg")
				item.setSubqueue(sq)
				item.SetPriority(p)
				item.SetSize(s)
				item.remove()
				item.setIndex(index)
				item.SetData(data)
			}

			read := func() {
				defer wg.Done()
				item.Key()
				item.ReserveGroup()
				item.Created()
				item.Priority()
				item.Size()
				item.removed()
				item.index()
				item.Data()
			}

			go change(10, 10, 10, "foo")
			go read()
			go change(11, 11, 11, "bar")
			go func() {
				defer wg.Done()
				item.setIndex(12)
			}()
			go read()
			wg.Wait()

			So(item.Size(), ShouldBeBetweenOrEqual, 10, 11)
			So(item.Priority(), ShouldBeBetweenOrEqual, 10, 11)
			So(item.index(), ShouldBeBetweenOrEqual, 10, 12)
			So(item.Data(), ShouldNotEqual, data)
			So(item.Key(), ShouldEqual, key)
			So(item.ReserveGroup(), ShouldEqual, "rg")
			So(item.removed(), ShouldBeFalse)
		})
	})
}
