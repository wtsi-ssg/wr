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

// mockOrder implements heap.Interface, keeping items in creation order.
type mockOrder struct {
	items []*Item
}

// newMockSubQueue creates a SubQueue that is ordered by creation.
func newMockSubQueue() SubQueue {
	return newHeapQueue(&mockOrder{})
}

// Len is to implement heap.Interface.
func (mo *mockOrder) Len() int { return len(mo.items) }

// Less is to implement heap.Interface.
func (mo *mockOrder) Less(i, j int) bool {
	mo.items[i].mutex.RLock()
	defer mo.items[i].mutex.RUnlock()
	mo.items[j].mutex.RLock()
	defer mo.items[j].mutex.RUnlock()

	return mo.items[i].created.Before(mo.items[j].created)
}

// Swap is to implement heap.Interface.
func (mo *mockOrder) Swap(i, j int) {
	heapSwap(mo.items, i, j)
}

// Push is to implement heap.Interface.
func (mo *mockOrder) Push(x interface{}) {
	mo.items = heapPush(mo.items, x)
}

// Pop is to implement heap.Interface.
func (mo *mockOrder) Pop() interface{} {
	var item interface{}
	mo.items, item = heapPop(mo.items)

	return item
}

func TestQueueHeapPushPop(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	Convey("Given a SubQueue", t, func() {
		sq := newMockSubQueue()

		Convey("You can push() items in to it", func() {
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {})

			Convey("And then pop() them out in FIFO order", func() {
				testPopsInInsertionOrder(ctx, sq, num, ips)
			})
		})

		Convey("2 threads can push() at once while accessing len()", func() {
			half := num / 2

			var wg sync.WaitGroup
			wg.Add(2)
			allOK := true
			pushHalf := func(first int) {
				defer wg.Done()

				for _, ip := range ips[first : first+half] {
					item := ip.toItem()
					sq.push(item)
					if sq.len() <= 0 {
						allOK = false
					}
				}
			}

			go pushHalf(0)
			go pushHalf(half)
			wg.Wait()

			So(sq.len(), ShouldEqual, num)
			So(allOK, ShouldBeTrue)

			Convey("And then 2 threads can pop() at once", func() {
				var wg sync.WaitGroup
				wg.Add(2)
				popHalf := func() {
					defer wg.Done()
					popItemsFromSubQueue(sq, half, func(key string, i int) {})
				}

				go popHalf()
				go popHalf()
				wg.Wait()

				So(sq.len(), ShouldEqual, 0)
			})
		})
	})
}

func testPopsInInsertionOrder(ctx context.Context, sq SubQueue, num int, ips []*ItemParameters) {
	popItemsFromSubQueue(sq, num, func(key string, i int) {
		So(key, ShouldEqual, ips[i].Key)
		So(sq.len(), ShouldEqual, num-i-1)
	})

	item := sq.pop(ctx)
	So(item, ShouldBeNil)
	So(sq.len(), ShouldEqual, 0)
}

// popper is used to test that popping happens after context cancellation.
type popper struct {
	sq        SubQueue
	done      bool
	doneMutex sync.RWMutex
}

// newPopper makes a new *popper.
func newPopper(sq SubQueue) *popper {
	return &popper{
		sq: sq,
	}
}

// pop calls pop on the SubQueue in a goroutine and returns after the call has
// been made, but potentially before the pop completes. The result of the pop
// can be read from the returned channel, after p.isDone() returns true.
func (p *popper) pop(ctx context.Context) chan *Item {
	started := make(chan bool)
	itemCh := make(chan *Item, 1)

	go func() {
		go func() {
			<-time.After(1 * time.Millisecond)
			close(started)
		}()

		item := p.sq.pop(ctx)
		p.doneMutex.Lock()
		p.done = true
		p.doneMutex.Unlock()
		itemCh <- item
	}()
	<-started

	return itemCh
}

// isDone, if called after pop(), tells you when the pop from the SubQueue has
// finished running.
func (p *popper) isDone() bool {
	<-time.After(1 * time.Millisecond)
	p.doneMutex.RLock()
	defer p.doneMutex.RUnlock()

	return p.done
}

func TestQueueHeapPopWait(t *testing.T) {
	Convey("Given a SubQueue with no items", t, func() {
		sq := newMockSubQueue()
		backgroundCtx := context.Background()
		cancellableCtx, cancel := context.WithCancel(backgroundCtx)
		defer cancel()

		Convey("pop() with an uncancellable context returns nothing immediately", func() {
			item := sq.pop(backgroundCtx)
			So(item, ShouldBeNil)
		})

		Convey("pop() with a cancellable context returns nothing after waiting", func() {
			popper := newPopper(sq)
			ich := popper.pop(cancellableCtx)
			So(popper.isDone(), ShouldBeFalse)
			cancel()
			So(popper.isDone(), ShouldBeTrue)
			So(<-ich, ShouldBeNil)
		})

		Convey("After push()ing an item, pop() with a cancellable context  returns immediately", func() {
			pushItemsToSubQueue(sq, newSetOfItemParameters(1), func(item *Item, i int) {})
			item := sq.pop(cancellableCtx)
			So(item, ShouldNotBeNil)
		})

		Convey("Before push()ing an item, pop() with a cancellable context returns immediately after push()ing", func() {
			pushDuringPop(cancellableCtx, sq, newSetOfItemParameters(1))
		})

		Convey("Multiple pop()s can wait at once", func() {
			pushDuringPop(cancellableCtx, sq, newSetOfItemParameters(10))
		})

		Convey("You can cancel the context on waiting pop()s while pushing", func() {
			ips := newSetOfItemParameters(100)
			poppers, ichs := makePoppers(cancellableCtx, sq, ips)

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				pushItemsToSubQueue(sq, ips, func(item *Item, i int) {})
			}()
			go func() {
				defer wg.Done()
				cancel()
			}()

			wg.Wait()

			notPopped := testSomePopsReturnedItems(ips, poppers, ichs)

			for i := 0; i < notPopped; i++ {
				item := sq.pop(backgroundCtx)
				So(item, ShouldNotBeNil)
			}

			So(sq.len(), ShouldEqual, 0)
		})
	})
}

func pushDuringPop(ctx context.Context, sq SubQueue, ips []*ItemParameters) {
	poppers, ichs := makePoppers(ctx, sq, ips)

	pushItemsToSubQueue(sq, ips, func(item *Item, i int) {})

	testAllPopsReturnedItems(ips, poppers, ichs)
}

func makePoppers(ctx context.Context, sq SubQueue, ips []*ItemParameters) ([]*popper, []chan *Item) {
	num := len(ips)
	poppers := make([]*popper, num)
	ichs := make([]chan *Item, num)

	for i := 0; i < num; i++ {
		poppers[i] = newPopper(sq)
		ichs[i] = poppers[i].pop(ctx)
		So(poppers[i].isDone(), ShouldBeFalse)
	}

	return poppers, ichs
}

func testAllPopsReturnedItems(ips []*ItemParameters, poppers []*popper, ichs []chan *Item) {
	num := len(ips)
	for i := 0; i < num; i++ {
		item, done := getItemFromPopperIfDone(poppers[i], ichs[i])
		So(done, ShouldBeTrue)
		So(item, ShouldNotBeNil)
		So(item.Key(), ShouldEqual, ips[i].Key)
	}
}

func getItemFromPopperIfDone(popper *popper, ich chan *Item) (*Item, bool) {
	if popper.isDone() {
		return <-ich, true
	}

	return nil, false
}

func testSomePopsReturnedItems(ips []*ItemParameters, poppers []*popper, ichs []chan *Item) int {
	num := len(ips)
	notPopped := 0
	notReady := 0

	<-time.After(1 * time.Millisecond)

	for i := 0; i < num; i++ {
		item, done := getItemFromPopperIfDone(poppers[i], ichs[i])

		switch {
		case !done:
			notReady++
		case item == nil:
			notPopped++
		default:
			So(item.Key(), ShouldEqual, ips[i-notPopped].Key)
		}
	}

	So(notReady, ShouldEqual, 0)

	return notPopped
}

func TestQueueHeapRemoveUpdate(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	Convey("Given a SubQueue with some items push()ed to it", t, func() {
		sq := newMockSubQueue()
		items := make([]*Item, num)
		pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
			items[i] = item
		})
		So(sq.len(), ShouldEqual, num)

		Convey("You can remove() an item ", func() {
			So(items[2].subQueue, ShouldEqual, sq)
			sq.remove(items[2])
			So(sq.len(), ShouldEqual, num-1)
			So(items[2].removed(), ShouldBeTrue)
			So(items[2].subQueue, ShouldBeNil)

			Convey("And then pop() the remainder out", func() {
				popItemsFromSubQueue(sq, num-1, func(key string, i int) {
					switch i {
					case 0, 1:
						So(key, ShouldEqual, ips[i].Key)
					default:
						So(key, ShouldEqual, ips[i+1].Key)
					}
				})
				So(sq.len(), ShouldEqual, 0)
			})
		})

		Convey("You can remove() items simultaneously", func() {
			var wg sync.WaitGroup
			wg.Add(2)
			removeItem := func(item *Item) {
				defer wg.Done()
				sq.remove(item)
			}
			go removeItem(items[2])
			go removeItem(items[4])
			wg.Wait()
			So(sq.len(), ShouldEqual, num-2)

			Convey("And remove()ing an item twice is harmless", func() {
				sq.remove(items[2])
				So(sq.len(), ShouldEqual, num-2)
			})
		})

		update := func() {
			item := items[2]
			item.mutex.Lock()
			item.created = time.Now()
			item.mutex.Unlock()
			sq.update(item)
		}

		Convey("You can update() an item's created, which changes pop() order", func() {
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

func testPopsInAlteredOrder(ctx context.Context, sq SubQueue, num int, ips []*ItemParameters) {
	popItemsFromSubQueue(sq, num, func(key string, i int) {
		switch i {
		case 0, 1:
			So(key, ShouldEqual, ips[i].Key)
		case 5:
			So(key, ShouldEqual, ips[2].Key)
		default:
			So(key, ShouldEqual, ips[i+1].Key)
		}
	})
	So(sq.len(), ShouldEqual, 0)
	item := sq.pop(ctx)
	So(item, ShouldBeNil)
}

func testSimultaneousUpdates(update1, update2 func()) {
	var wg sync.WaitGroup

	updateSim := func(method func()) {
		defer wg.Done()
		method()
	}

	wg.Add(2)

	go updateSim(update1)
	go updateSim(update2)

	wg.Wait()
}

func TestQueueHeapSimultaneous(t *testing.T) {
	num := 1000
	ips := newSetOfItemParameters(num)

	Convey("You can do all SubQueue operations simultaneously", t, func() {
		sq := newMockSubQueue()
		var wg sync.WaitGroup

		addAtLeast := 500
		items := make([]*Item, addAtLeast)
		started := make(chan bool)
		wg.Add(1)
		go func() {
			defer wg.Done()
			pushItemsToSubQueue(sq, ips, func(item *Item, i int) {
				if i < addAtLeast {
					items[i] = item
				}
				if i == addAtLeast {
					close(started)
				}
			})
		}()

		updateAtLeast := 250
		updated := make(chan bool)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-started
			for i := 0; i < addAtLeast; i++ {
				items[i].SetPriority(uint8(i))

				if i == updateAtLeast {
					close(updated)
				}
			}
		}()

		removeAtLeast := 125
		removed := make(chan bool)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-updated
			for i := 0; i < addAtLeast; i++ {
				sq.remove(items[i])

				if i == removeAtLeast {
					close(removed)
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-removed
			popItemsFromSubQueue(sq, addAtLeast, func(key string, i int) {})
		}()

		wg.Wait()
		So(sq.len(), ShouldBeBetweenOrEqual, 0, num-addAtLeast)
	})
}

func pushItemsToSubQueue(sq SubQueue, ips []*ItemParameters, f func(*Item, int)) {
	for i, ip := range ips {
		item := ip.toItem()
		f(item, i)
		sq.push(item)
	}
}

func popItemsFromSubQueue(sq SubQueue, num int, f func(string, int)) {
	ctx := context.Background()

	for i := 0; i < num; i++ {
		item := sq.pop(ctx)
		if item != nil {
			f(item.Key(), i)
		} else {
			f("", i)
		}
	}
}
