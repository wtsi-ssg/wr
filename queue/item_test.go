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

type mockSubQueue struct {
	updates int
	sync.Mutex
}

func (sq *mockSubQueue) update(item *Item) {
	sq.Lock()
	defer sq.Unlock()
	sq.updates++
}

func (sq *mockSubQueue) push(*Item) {}

func (sq *mockSubQueue) pop(context.Context) *Item { return nil }

func (sq *mockSubQueue) remove(*Item) {}

func (sq *mockSubQueue) len() int { return 0 }

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
		So(item.ttr, ShouldEqual, 0)
		So(item.releaseAt, ShouldBeZeroValue)
		So(item.readyAt, ShouldBeZeroValue)
		So(item.ReleaseAt(), ShouldHappenAfter, before.Add(unsetItemExpiry))
		So(item.ReadyAt(), ShouldHappenAfter, before.Add(unsetItemExpiry))

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
		So(item.index(), ShouldEqual, indexOfRemovedItem)

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
			So(item.belongsTo(sq), ShouldBeTrue)

			item.setIndex(0)
			So(item.subQueueIndex, ShouldEqual, 0)

			Convey("When you set priority or size or Touch() or Reset(), the subQueue is updated", func() {
				new := uint8(10)
				item.SetPriority(new)
				So(item.Priority(), ShouldEqual, new)
				So(sq.updates, ShouldEqual, 1)

				item.SetSize(new)
				So(item.Size(), ShouldEqual, new)
				So(sq.updates, ShouldEqual, 2)

				item.Touch()
				t := time.Now()
				So(item.releaseAt, ShouldHappenBetween, t, t.Add(DefaultTTR))
				So(sq.updates, ShouldEqual, 3)

				item.restart()
				t = time.Now()
				So(item.readyAt, ShouldHappenBetween, t, t.Add(DefaultDelay))
				So(sq.updates, ShouldEqual, 4)
			})

			Convey("And then you can remove() it", func() {
				remove()
				So(item.subQueue, ShouldBeNil)
				So(item.belongsTo(sq), ShouldBeFalse)
				So(item.index(), ShouldEqual, indexOfRemovedItem)
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

		Convey("Touch() and restart() use TTR and Delay properties", func() {
			d := 1 * time.Second
			ip = &ItemParameters{
				TTR:   d,
				Delay: d,
			}
			item = ip.toItem()

			item.Touch()
			t := time.Now()
			So(item.releaseAt, ShouldHappenBetween, t, t.Add(d))

			item.restart()
			t = time.Now()
			So(item.readyAt, ShouldHappenBetween, t, t.Add(d))
		})

		Convey("You can set and get item properties simultaneously", func() {
			sq := &mockSubQueue{}
			newVal := uint8(10)

			canDoInPairsConcurrently(func() {
				item.setSubqueue(sq)
				item.setIndex(int(newVal))
				item.SetPriority(newVal)
				item.SetSize(newVal)
				item.SetData("foo")
			}, func() {
				item.belongsTo(sq)
				item.index()
				item.Priority()
				item.Size()
				item.Data()
				item.Key()
				item.Created()
			})

			canDoInPairsConcurrently(func() {
				item.SetReserveGroup("rg")
			}, func() {
				item.ReserveGroup()
			})

			canDoInPairsConcurrently(item.restart, func() {
				item.ReadyAt()
			})

			canDoInPairsConcurrently(item.Touch, func() {
				item.ReleaseAt()
			})

			So(item.Size(), ShouldEqual, newVal)
			So(item.Priority(), ShouldEqual, newVal)
			So(item.index(), ShouldEqual, int(newVal))
			So(item.Data(), ShouldNotEqual, data)
			So(item.Key(), ShouldEqual, key)
			So(item.ReserveGroup(), ShouldEqual, "rg")
			So(item.ReadyAt(), ShouldHappenAfter, time.Now())
			So(item.ReleaseAt(), ShouldHappenAfter, time.Now())

			canDoInPairsConcurrently(item.remove, func() {
				item.removed()
			})

			So(item.removed(), ShouldBeTrue)

			canDoInPairsConcurrently(func() {
				item.remove()
				item.setSubqueue(sq)
				item.setIndex(int(newVal))
			}, func() {
				item.removed()
			})

			So(item.removed(), ShouldBeFalse)
		})
	})

	Convey("ReleaseAt() and ReadyAt() work on nil items", t, func() {
		before := time.Now()

		var item *Item

		So(item.ReleaseAt(), ShouldHappenAfter, before.Add(unsetItemExpiry))
		So(item.ReadyAt(), ShouldHappenAfter, before.Add(unsetItemExpiry))
	})
}

func TestQueueItemTransitions(t *testing.T) {
	Convey("Given an item", t, func() {
		ip := &ItemParameters{
			Key:  "key",
			Data: "data",
		}
		item := ip.toItem()
		So(item.State(), ShouldEqual, ItemStateReady)
		So(item.readyAt, ShouldBeZeroValue)
		So(item.releaseAt, ShouldBeZeroValue)

		Convey("You can switch from ready to run", func() {
			canSwitchTo(item, ItemStateRun)
			So(item.releaseAt, ShouldNotBeZeroValue)
			So(item.readyAt, ShouldBeZeroValue)

			Convey("You can switch from run to delay", func() {
				canSwitchTo(item, ItemStateDelay)
				So(item.readyAt, ShouldNotBeZeroValue)

				canSwitchToRDR(item)

				Convey("You can't switch from delay to run or bury", func() {
					cannotSwitchTo(item, ItemStateDelay, ItemStateRun, ItemStateBury)
				})
			})

			Convey("You can switch from run to bury", func() {
				canSwitchTo(item, ItemStateBury)

				canSwitchToRDR(item)

				Convey("You can't switch from bury to run or delay", func() {
					cannotSwitchTo(item, ItemStateBury, ItemStateRun, ItemStateDelay)
				})
			})

			Convey("You can switch from run to dependent", func() {
				canSwitchTo(item, ItemStateDependent)
			})

			Convey("You can't switch from run to ready or removed", func() {
				cannotSwitchTo(item, ItemStateRun, ItemStateReady, ItemStateRemoved)
			})
		})

		Convey("You can switch from ready to dependent", func() {
			canSwitchTo(item, ItemStateDependent)

			Convey("You can switch from dependent to ready", func() {
				canSwitchTo(item, ItemStateReady)
			})

			Convey("You can switch from dependent to removed", func() {
				canSwitchTo(item, ItemStateRemoved)
			})

			Convey("You can't switch from dependent to run, delay or bury", func() {
				cannotSwitchTo(item, ItemStateDependent, ItemStateRun, ItemStateDelay, ItemStateBury)
			})
		})

		Convey("You can switch from ready to removed", func() {
			canSwitchTo(item, ItemStateRemoved)

			Convey("You can't switch from removed to anything else", func() {
				cannotSwitchTo(item, ItemStateRemoved,
					ItemStateReady,
					ItemStateRun,
					ItemStateDelay,
					ItemStateBury,
					ItemStateDependent,
				)
			})
		})

		Convey("You can't switch from ready to delay or bury", func() {
			cannotSwitchTo(item, ItemStateReady, ItemStateDelay, ItemStateBury)
		})

		Convey("You can attempt to switch states and read state simultaneously", func() {
			errCh := make(chan error, 10)
			canDoInPairsConcurrently(func() {
				errCh <- item.SwitchState(ItemStateRun)
			}, func() {
				item.State()
			})

			So(item.State(), ShouldEqual, ItemStateRun)
			errors := 0
			for i := 0; i < 10; i++ {
				err := <-errCh
				if err != nil {
					errors++
				}
			}
			So(errors, ShouldEqual, 9)
		})
	})
}

func canDoInPairsConcurrently(f1 func(), f2 func()) {
	var wg sync.WaitGroup

	for i := 1; i <= 10; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			f1()
		}()
		go func() {
			defer wg.Done()
			f2()
		}()
	}

	wg.Wait()
}

func canSwitchTo(item *Item, to ItemState) {
	err := item.SwitchState(to)
	So(err, ShouldBeNil)
	So(item.State(), ShouldEqual, to)
}

func cannotSwitchTo(item *Item, from ItemState, tos ...ItemState) {
	for _, to := range tos {
		err := item.SwitchState(to)
		So(err, ShouldNotBeNil)
		So(item.State(), ShouldEqual, from)
	}
}

func canSwitchToRDR(item *Item) {
	Convey("You can switch from delay to ready", func() {
		canSwitchTo(item, ItemStateReady)
	})

	Convey("You can switch from delay to dependent", func() {
		canSwitchTo(item, ItemStateDependent)
	})

	Convey("You can switch from delay to removed", func() {
		canSwitchTo(item, ItemStateRemoved)
	})
}
