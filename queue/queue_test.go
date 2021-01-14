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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	dsync "github.com/sasha-s/go-deadlock"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/clog"
)

var errNoItem = errors.New("no item")

func TestQueueAddGet(t *testing.T) {
	ips := newSetOfItemParameters(20)

	Convey("Given a queue", t, func() {
		q := New()

		Convey("You can Add() items with just a key and data", func() {
			q.Add(ips[0])
			q.Add(ips[1])

			Convey("And then Get() the items back", func() {
				item := q.Get(ips[1].Key)
				So(item.Key(), ShouldEqual, ips[1].Key)
				So(item.Data().(int), ShouldEqual, ips[1].Data)

				item = q.Get(ips[0].Key)
				So(item.Key(), ShouldEqual, ips[0].Key)
				So(item.Data().(int), ShouldEqual, ips[0].Data)

				item = q.Get("foo")
				So(item, ShouldBeNil)
			})
		})

		Convey("You can Add() a meaningless item without a key and data", func() {
			q.Add(&ItemParameters{})
			item := q.Get("")
			So(item, ShouldNotBeNil)
		})

		Convey("You can Add() items concurrently", func() {
			n := 20
			canDoConcurrently(n, queueAdder(q))

			Convey("And then Get() them concurrently", func() {
				canDoConcurrently(n, queueGetter(q))
			})
		})

		Convey("You can Add() multiple items at once", func() {
			q.Add(ips[0], ips[1])

			item := q.Get(ips[0].Key)
			So(item, ShouldNotBeNil)
			item = q.Get(ips[1].Key)
			So(item, ShouldNotBeNil)

			Convey("And duplicates are detected", func() {
				added, dups := q.Add(ips[0], ips[1], &ItemParameters{
					Key:  "foo",
					Data: "bar",
				}, ips[0])
				So(added, ShouldEqual, 1)
				So(dups, ShouldEqual, 3)
				So(q.readyQueues.queues[""].len(), ShouldEqual, 3)

				item := q.Get(ips[0].Key)
				So(item, ShouldNotBeNil)
				item = q.Get(ips[1].Key)
				So(item, ShouldNotBeNil)
				item = q.Get("foo")
				So(item, ShouldNotBeNil)
			})
		})
	})
}

func TestQueueReserve(t *testing.T) {
	num := 10
	s, p := uint8(3), uint8(2)
	ctx := context.Background()

	Convey("Given a queue", t, func() {
		q := New()

		reserveMultiple := func(n int, f func(*Item, int)) {
			for i := 0; i < n; i++ {
				item := q.Reserve(ctx, "")
				f(item, i)
			}
		}

		reserveSPOrder := func(ips []*ItemParameters) {
			reserveMultiple(num, func(item *Item, i int) {
				So(item, ShouldNotBeNil)
				switch i {
				case 0:
					So(item.Key(), ShouldEqual, ips[4].Key)
				case 1:
					So(item.Key(), ShouldEqual, ips[3].Key)
				case 2, 3, 4:
					So(item.Key(), ShouldEqual, ips[i-2].Key)
				}
			})
			item := q.Reserve(ctx, "")
			So(item, ShouldBeNil)
		}

		Convey("With a set of equal priority and size items added", func() {
			ips := newSetOfItemParameters(num)
			q.Add(ips...)

			Convey("You can Reserve() the items in FIFO order", func() {
				reserveMultiple(num, func(item *Item, i int) {
					So(item, ShouldNotBeNil)
					So(item.Key(), ShouldEqual, ips[i].Key)
				})
				item := q.Reserve(ctx, "")
				So(item, ShouldBeNil)
			})

			Convey("2 threads can Reserve() at once", func() {
				half := num / 2

				var wg sync.WaitGroup
				wg.Add(2)
				reserveHalf := func(keys []string) {
					defer wg.Done()
					reserveMultiple(half, func(item *Item, i int) {
						if item != nil {
							keys[i] = item.Key()
						}
					})
				}

				keysA := make([]string, half)
				keysB := make([]string, half)
				go reserveHalf(keysA)
				go reserveHalf(keysB)
				wg.Wait()

				keysAll := append(keysA, keysB...)
				keys := make(map[string]bool)
				for _, key := range keysAll {
					keys[key] = true
				}
				So(len(keys), ShouldEqual, num)
			})

			Convey("You can Add() and Reserve() simultaneously", func() {
				ips = newSetOfItemParameters(num * 2)
				keys := make([]string, num)

				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					q.Add(ips...)
				}()

				go func() {
					defer wg.Done()
					reserveMultiple(num, func(item *Item, i int) {
						if item != nil {
							keys[i] = item.Key()
						}
					})
				}()

				wg.Wait()

				for i, key := range keys {
					So(key, ShouldEqual, ips[i].Key)
				}

				reserveMultiple(num, func(item *Item, i int) {
					So(item.Key(), ShouldEqual, ips[i+10].Key)
				})
			})

			Convey("You can Remove() an item, then Reserve() the others", func() {
				q.Remove(ips[5].Key)

				item := q.Get(ips[5].Key)
				So(item, ShouldBeNil)

				reserveMultiple(num-1, func(item *Item, i int) {
					So(item, ShouldNotBeNil)
					So(item.Key, ShouldNotEqual, ips[5].Key)
				})
				item = q.Reserve(ctx, "")
				So(item, ShouldBeNil)
				So(len(q.readyQueues.queues), ShouldEqual, 0)

				Convey("Remove()ing the only item in the readyQueue drops the readyQueue", func() {
					added, dups := q.Add(ips[5])
					So(added, ShouldEqual, 1)
					So(dups, ShouldEqual, 0)
					So(len(q.readyQueues.queues), ShouldEqual, 1)
					q.Remove(ips[5].Key)
					item := q.Get(ips[5].Key)
					So(item, ShouldBeNil)
					So(len(q.readyQueues.queues), ShouldEqual, 0)
				})
			})

			Convey("2 threads can Remove() at once", func() {
				var wg sync.WaitGroup
				wg.Add(2)
				remove := func(key string) {
					defer wg.Done()
					q.Remove(key)
				}

				go remove(ips[5].Key)
				go remove(ips[3].Key)
				wg.Wait()

				item := q.Get(ips[5].Key)
				So(item, ShouldBeNil)
				item = q.Get(ips[3].Key)
				So(item, ShouldBeNil)
				So(len(q.items), ShouldEqual, num-2)
				So(q.readyQueues.queues[""].len(), ShouldEqual, num-2)
			})

			three := q.Get(ips[3].Key)
			four := q.Get(ips[4].Key)

			Convey("You can set new item properties, then Reserve() them in the new order", func() {
				three.SetSize(s)
				four.SetPriority(p)

				reserveSPOrder(ips)
			})

			Convey("2 threads can set item properties at once", func() {
				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					four.SetPriority(p)
				}()

				go func() {
					defer wg.Done()
					three.SetSize(s)
				}()

				wg.Wait()
				reserveSPOrder(ips)
			})
		})

		Convey("With a set of different priority and size items added", func() {
			ips := newSetOfItemParameters(num)
			ips[3].Size = s
			ips[4].Priority = p
			q.Add(ips...)

			Convey("You can Reserve() the items in priority, then size, then FIFO order", func() {
				reserveSPOrder(ips)
			})
		})
	})
}

func TestQueueReserveWait(t *testing.T) {
	ips := newSetOfItemParameters(1)
	backgroundCtx := context.Background()
	wait := 5 * time.Millisecond

	Convey("Given a queue with no items", t, func() {
		q := New()

		Convey("Reserve() with an uncancellable context returns nothing immediately", func() {
			t := time.Now()
			item := q.Reserve(backgroundCtx, "")
			So(item, ShouldBeNil)
			So(time.Now(), ShouldHappenBefore, t.Add(wait))
		})

		reserveWithTimeout := func() (time.Time, *Item) {
			t := time.Now()
			timerCtx, cancel := context.WithTimeout(backgroundCtx, wait)
			defer cancel()
			item := q.Reserve(timerCtx, "")

			return t, item
		}

		Convey("Reserve() with a timeout returns nothing after waiting", func() {
			t, item := reserveWithTimeout()
			So(item, ShouldBeNil)
			So(time.Now(), ShouldHappenAfter, t.Add(wait))
		})

		Convey("After Add()ing an item, Reserve() with a timeout returns immediately", func() {
			q.Add(ips...)

			t, item := reserveWithTimeout()
			So(item, ShouldNotBeNil)
			So(time.Now(), ShouldHappenBefore, t.Add(wait))
		})

		Convey("Before Add()ing an item, Reserve() with a timeout returns immediately after Add()ing", func() {
			var t time.Time
			var item *Item
			var wg sync.WaitGroup

			wg.Add(1)
			started := make(chan bool)
			go func() {
				defer wg.Done()
				go func() {
					<-time.After(500 * time.Nanosecond)
					started <- true
				}()

				t, item = reserveWithTimeout()
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-started
				q.Add(ips...)
			}()

			wg.Wait()
			So(item, ShouldNotBeNil)
			So(time.Now(), ShouldHappenBefore, t.Add(wait))
		})

		Convey("Multiple Reserve()s can wait at once", func() {
			num := 100
			items := make([]*Item, num)
			var wg sync.WaitGroup

			for i := 0; i < num; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					_, item := reserveWithTimeout()
					items[i] = item
				}(i)
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				q.Add(newSetOfItemParameters(num)...)
			}()

			wg.Wait()

			var gotAnItem bool
			for i := 0; i < num; i++ {
				if items[i] != nil {
					gotAnItem = true

					break
				}
			}
			So(gotAnItem, ShouldBeTrue)
		})
	})
}

func TestQueueReserveGroups(t *testing.T) {
	num := 2
	ips := newSetOfItemParameters(num)
	rg1, rg2, rg3 := "1", "2", "3"
	ips[0].ReserveGroup = rg1
	ips[1].ReserveGroup = rg2

	backgroundCtx := context.Background()

	Convey("Given a queue with items of differing ReserveGroups added", t, func() {
		q := New()
		q.Add(ips...)

		Convey("Items can be reserved by specifying a group", func() {
			item := q.Reserve(backgroundCtx, "")
			So(item, ShouldBeNil)

			item = q.Reserve(backgroundCtx, rg2)
			So(item, ShouldNotBeNil)
			So(item.Key(), ShouldEqual, ips[1].Key)

			item = q.Reserve(backgroundCtx, rg1)
			So(item, ShouldNotBeNil)
			So(item.Key(), ShouldEqual, ips[0].Key)
		})

		Convey("You can change the group of an item and reserve it", func() {
			q.ChangeReserveGroup(ips[0].Key, rg3)

			item := q.Reserve(backgroundCtx, rg2)
			So(item, ShouldNotBeNil)
			So(item.Key(), ShouldEqual, ips[1].Key)

			item = q.Reserve(backgroundCtx, rg1)
			So(item, ShouldBeNil)

			item = q.Reserve(backgroundCtx, rg3)
			So(item, ShouldNotBeNil)
			So(item.Key(), ShouldEqual, ips[0].Key)
		})
	})
}

func TestQueueRun(t *testing.T) {
	dsync.Opts.DeadlockTimeout = 6 * time.Millisecond
	num := 2
	ips := newSetOfItemParameters(num)
	backgroundCtx := context.Background()
	ttr := 5 * time.Millisecond

	for _, ip := range ips {
		ip.TTR = ttr
	}

	Convey("Given a queue with items added", t, func() {
		q := New()
		q.Add(ips...)

		Convey("Reserving an item moves it to the run SubQueue and starts its TTR", func() {
			So(q.readyQueues.numItems(), ShouldEqual, 2)
			So(q.runQueue.len(), ShouldEqual, 0)
			item := q.Reserve(backgroundCtx, "")
			t := time.Now()
			So(item, ShouldNotBeNil)
			So(q.readyQueues.numItems(), ShouldEqual, 1)
			So(q.runQueue.len(), ShouldEqual, 1)
			So(item.releaseAt, ShouldHappenBetween, t, t.Add(ttr))

			Convey("You wouldn't be able to put it on the run SubQueue again", func() {
				buff := clog.ToBufferAtLevel("eror")
				defer clog.ToDefault()
				q.pushToRunQueue(backgroundCtx, item)
				errStr := buff.String()
				So(errStr, ShouldContainSubstring, "lvl=eror")
				So(errStr, ShouldContainSubstring, `err="item key0 cannot transition from run to run"`)
			})

			Convey("After TTR, the item is automatically switched to the delay SubQueue", func() {
				So(item.State(), ShouldEqual, ItemStateRun)
				So(q.delayQueue.len(), ShouldEqual, 0)
				<-time.After(ttr)
				<-time.After(ttr)
				So(item.State(), ShouldEqual, ItemStateDelay)
				// *** this sometimes fails, and we don't have 100% code
				// coverage, and we need lots more tests to see if this
				// implementation really works, eg. with multiple items with
				// identical releaseAt time, or sequential times.
				So(q.delayQueue.len(), ShouldEqual, 1)
				So(q.runQueue.len(), ShouldEqual, 0)
			})
		})
	})
}

func newSetOfItemParameters(n int) []*ItemParameters {
	ips := make([]*ItemParameters, n)

	for i := 0; i < n; i++ {
		ips[i] = newItemParameters(i)
	}

	return ips
}

func newItemParameters(i int) *ItemParameters {
	return &ItemParameters{
		Key:  fmt.Sprintf("key%d", i),
		Data: i,
	}
}

type errorReturningFunc func(int) error

func queueAdder(q *Queue) errorReturningFunc {
	return func(i int) error {
		q.Add(newItemParameters(i))

		return nil
	}
}

func queueGetter(q *Queue) errorReturningFunc {
	return func(i int) error {
		item := q.Get(fmt.Sprintf("key%d", i))
		if item == nil {
			return errNoItem
		}

		return nil
	}
}

// canDoConcurrently runs the given function n times concurrently, and tests
// that they all returning nil error.
func canDoConcurrently(n int, f errorReturningFunc) {
	errCh := doConcurrently(n, f)
	checkErrorsInChan(n, errCh, func(err error) {
		So(err, ShouldBeNil)
	})
}

func doConcurrently(n int, f errorReturningFunc) chan error {
	errCh := make(chan error, n)

	for i := 1; i <= n; i++ {
		go func(i int) {
			err := f(i)
			errCh <- err
		}(i)
	}

	return errCh
}

func checkErrorsInChan(n int, errCh chan error, f func(error)) {
	for i := 1; i <= n; i++ {
		f(<-errCh)
	}
}

// BenchmarkQueueAddGet showed that adding and getting to a manually mutex
// locked map was consistently faster than using sync.Map.
func BenchmarkQueueAddGet(b *testing.B) {
	num := 20

	for n := 0; n < b.N; n++ {
		q := New()
		errCh := doConcurrently(num, queueAdder(q))
		checkErrorsInChan(num, errCh, func(err error) {})

		for i := 0; i < 5; i++ {
			errCh = doConcurrently(num, queueGetter(q))
			checkErrorsInChan(num, errCh, func(err error) {})
		}
	}
}

func BenchmarkQueueAddMany(b *testing.B) {
	ips := newSetOfItemParameters(20)

	for n := 0; n < b.N; n++ {
		q := New()
		q.Add(ips...)
	}
}
