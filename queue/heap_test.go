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

func TestQueueHeapPushPop(t *testing.T) {
	num := 6
	ips := newSetOfItemParameters(num)
	ctx := context.Background()

	Convey("Given a HeapQueue", t, func() {
		hq := NewHeapQueue()

		Convey("You can push() equal priority items in to it", func() {
			pushItemsToHeap(hq, ips, func(item *Item, i int) {})

			Convey("And then pop() them out in FIFO order", func() {
				popItemsFromHeap(hq, num, func(key string, i int) {
					So(key, ShouldEqual, ips[i].Key)
					So(hq.len(), ShouldEqual, num-i-1)
				})
				item := hq.pop(ctx)
				So(item, ShouldBeNil)
				So(hq.len(), ShouldEqual, 0)
			})
		})

		testPopsReverseOrder := func() {
			popItemsFromHeap(hq, num, func(key string, i int) {
				So(key, ShouldEqual, ips[num-1-i].Key)
			})
		}

		Convey("You can push() different priority items in to it", func() {
			pushItemsToHeap(hq, ips, func(item *Item, i int) {
				item.priority = uint8(i)
			})

			Convey("And then pop() them out in priority order", func() {
				testPopsReverseOrder()
			})
		})

		Convey("You can push() different size items in to it", func() {
			pushItemsToHeap(hq, ips, func(item *Item, i int) {
				item.size = uint8(i)
			})

			Convey("And then pop() them out in size order", func() {
				testPopsReverseOrder()
			})
		})

		Convey("Priority has precedence over size which has precedence over age", func() {
			pushItemsToHeap(hq, ips, func(item *Item, i int) {
				switch i {
				case 3:
					item.priority = uint8(3)
					item.size = uint8(4)
				case 4:
					item.priority = uint8(3)
					item.size = uint8(5)
				}
			})

			popItemsFromHeap(hq, num, func(key string, i int) {
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

		Convey("2 threads can push() at once while accessing len()", func() {
			half := num / 2

			var wg sync.WaitGroup
			wg.Add(2)
			allOK := true
			pushHalf := func(first int) {
				defer wg.Done()

				for _, ip := range ips[first : first+half] {
					item := ip.toItem()
					hq.push(item)
					if hq.len() <= 0 {
						allOK = false
					}
				}
			}

			go pushHalf(0)
			go pushHalf(half)
			wg.Wait()

			So(hq.len(), ShouldEqual, num)
			So(allOK, ShouldBeTrue)

			Convey("And then 2 threads can pop() at once", func() {
				var wg sync.WaitGroup
				wg.Add(2)
				popHalf := func() {
					defer wg.Done()
					popItemsFromHeap(hq, half, func(key string, i int) {})
				}

				go popHalf()
				go popHalf()
				wg.Wait()

				So(hq.len(), ShouldEqual, 0)
			})
		})
	})
}

// popper is used to test that popping happens after context cancellation.
type popper struct {
	hq        *HeapQueue
	done      bool
	doneMutex sync.RWMutex
}

// newPopper makes a new *popper.
func newPopper(hq *HeapQueue) *popper {
	return &popper{
		hq: hq,
	}
}

// pop calls pop on the HeapQueue in a goroutine and returns after the call has
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

		item := p.hq.pop(ctx)
		p.doneMutex.Lock()
		p.done = true
		p.doneMutex.Unlock()
		itemCh <- item
	}()
	<-started

	return itemCh
}

// isDone, if called after pop(), tells you when the pop from the HeapQueue has
// finished running.
func (p *popper) isDone() bool {
	<-time.After(1 * time.Millisecond)
	p.doneMutex.RLock()
	defer p.doneMutex.RUnlock()

	return p.done
}

func TestQueueHeapPopWait(t *testing.T) {
	Convey("Given a HeapQueue with no items", t, func() {
		hq := NewHeapQueue()
		backgroundCtx := context.Background()
		cancellableCtx, cancel := context.WithCancel(backgroundCtx)
		defer cancel()

		Convey("pop() with an uncancellable context returns nothing immediately", func() {
			item := hq.pop(backgroundCtx)
			So(item, ShouldBeNil)
		})

		Convey("pop() with a cancellable context returns nothing after waiting", func() {
			popper := newPopper(hq)
			ich := popper.pop(cancellableCtx)
			So(popper.isDone(), ShouldBeFalse)
			cancel()
			So(popper.isDone(), ShouldBeTrue)
			So(<-ich, ShouldBeNil)
		})

		Convey("After push()ing an item, pop() with a cancellable context  returns immediately", func() {
			pushItemsToHeap(hq, newSetOfItemParameters(1), func(item *Item, i int) {})
			item := hq.pop(cancellableCtx)
			So(item, ShouldNotBeNil)
		})

		Convey("Before push()ing an item, pop() with a cancellable context returns immediately after push()ing", func() {
			pushDuringPop(cancellableCtx, hq, newSetOfItemParameters(1))
		})

		Convey("Multiple pop()s can wait at once", func() {
			pushDuringPop(cancellableCtx, hq, newSetOfItemParameters(10))
		})

		Convey("You can cancel the context on waiting pop()s while pushing", func() {
			ips := newSetOfItemParameters(100)
			poppers, ichs := makePoppers(cancellableCtx, hq, ips)

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				pushItemsToHeap(hq, ips, func(item *Item, i int) {})
			}()
			go func() {
				defer wg.Done()
				cancel()
			}()

			wg.Wait()

			notPopped := testSomePopsReturnedItems(ips, poppers, ichs)

			for i := 0; i < notPopped; i++ {
				item := hq.pop(backgroundCtx)
				So(item, ShouldNotBeNil)
			}

			So(hq.len(), ShouldEqual, 0)
		})
	})
}

func pushDuringPop(ctx context.Context, hq *HeapQueue, ips []*ItemParameters) {
	poppers, ichs := makePoppers(ctx, hq, ips)

	pushItemsToHeap(hq, ips, func(item *Item, i int) {})

	testAllPopsReturnedItems(ips, poppers, ichs)
}

func makePoppers(ctx context.Context, hq *HeapQueue, ips []*ItemParameters) ([]*popper, []chan *Item) {
	num := len(ips)
	poppers := make([]*popper, num)
	ichs := make([]chan *Item, num)

	for i := 0; i < num; i++ {
		poppers[i] = newPopper(hq)
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

	Convey("Given a HeapQueue with some items push()ed to it", t, func() {
		hq := NewHeapQueue()
		items := make([]*Item, num)
		pushItemsToHeap(hq, ips, func(item *Item, i int) {
			items[i] = item
		})
		So(hq.len(), ShouldEqual, num)

		Convey("You can remove() an item ", func() {
			So(items[2].subQueue, ShouldEqual, hq)
			hq.remove(items[2])
			So(hq.len(), ShouldEqual, num-1)
			So(items[2].removed(), ShouldBeTrue)
			So(items[2].subQueue, ShouldBeNil)

			Convey("And then pop() the remainder out", func() {
				popItemsFromHeap(hq, num-1, func(key string, i int) {
					switch i {
					case 0, 1:
						So(key, ShouldEqual, ips[i].Key)
					default:
						So(key, ShouldEqual, ips[i+1].Key)
					}
				})
				So(hq.len(), ShouldEqual, 0)
			})
		})

		Convey("You can remove() items simultaneously", func() {
			var wg sync.WaitGroup
			wg.Add(2)
			removeItem := func(item *Item) {
				defer wg.Done()
				hq.remove(item)
			}
			go removeItem(items[2])
			go removeItem(items[4])
			wg.Wait()
			So(hq.len(), ShouldEqual, num-2)

			Convey("And remove()ing an item twice is harmless", func() {
				hq.remove(items[2])
				So(hq.len(), ShouldEqual, num-2)
			})
		})

		popAlteredOrder := func() {
			popItemsFromHeap(hq, num, func(key string, i int) {
				switch i {
				case 0:
					So(key, ShouldEqual, ips[2].Key)
				case 1, 2:
					So(key, ShouldEqual, ips[i-1].Key)
				default:
					So(key, ShouldEqual, ips[i].Key)
				}
			})
			So(hq.len(), ShouldEqual, 0)
			item := hq.pop(ctx)
			So(item, ShouldBeNil)
		}

		update2p := func() {
			items[2].SetPriority(1)
		}
		update2s := func() {
			items[2].SetSize(2)
		}

		Convey("You can update() an item's priority, which changes pop() order", func() {
			update2p()
			So(hq.len(), ShouldEqual, num)
			popAlteredOrder()
		})

		Convey("You can update() an item's size, which changes pop() order", func() {
			update2s()
			So(hq.len(), ShouldEqual, num)
			popAlteredOrder()
		})

		Convey("You can do simultaneous update()s", func() {
			var wg sync.WaitGroup
			wg.Add(2)
			update := func(method func()) {
				defer wg.Done()
				method()
			}

			go update(update2p)
			go update(update2s)
			wg.Wait()
			popAlteredOrder()
		})
	})
}

func TestQueueHeapSimultaneous(t *testing.T) {
	num := 1000
	ips := newSetOfItemParameters(num)

	Convey("You can do all HeapQueue operations simultaneously", t, func() {
		hq := NewHeapQueue()
		var wg sync.WaitGroup

		addAtLeast := 500
		items := make([]*Item, addAtLeast)
		started := make(chan bool)
		wg.Add(1)
		go func() {
			defer wg.Done()
			pushItemsToHeap(hq, ips, func(item *Item, i int) {
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
				hq.remove(items[i])

				if i == removeAtLeast {
					close(removed)
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-removed
			popItemsFromHeap(hq, addAtLeast, func(key string, i int) {})
		}()

		wg.Wait()
		So(hq.len(), ShouldBeBetweenOrEqual, 0, num-addAtLeast)
	})
}

func pushItemsToHeap(hq *HeapQueue, ips []*ItemParameters, f func(*Item, int)) {
	for i, ip := range ips {
		item := ip.toItem()
		f(item, i)
		hq.push(item)
	}
}

func popItemsFromHeap(hq *HeapQueue, num int, f func(string, int)) {
	ctx := context.Background()

	for i := 0; i < num; i++ {
		item := hq.pop(ctx)
		if item != nil {
			f(item.Key(), i)
		} else {
			f("", i)
		}
	}
}
