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
	"container/heap"
	"context"
	"sync"

	"github.com/wtsi-ssg/wr/clog"
)

// HeapQueue holds Items in priority||size||age order. Use the private methods
// (which are thread safe), not the public ones that are there to implement
// heap.Interface.
type HeapQueue struct {
	items       []*Item
	pushChs     map[string]chan *Item
	waitingPops []string
	mutex       sync.RWMutex
}

// NewHeapQueue returns an initialised heap-based queue.
func NewHeapQueue() *HeapQueue {
	hq := &HeapQueue{
		pushChs: make(map[string]chan *Item),
	}

	heap.Init(hq)

	return hq
}

// push adds an item to the queue.
func (hq *HeapQueue) push(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()
	defer hq.popIfWaiting()

	item.setSubqueue(hq)
	heap.Push(hq, item)
}

// popIfWaiting pops the item we just pushed and sends it down the next pushCh,
// if any exist.
//
// You must hold the mutex lock before calling this.
func (hq *HeapQueue) popIfWaiting() {
	pushCh := hq.getNextPushCh()
	if pushCh == nil {
		return
	}
	pushCh <- heap.Pop(hq).(*Item)
}

// getNextPushCh finds the oldest waitingPops id that still has a pushChs
// entry, deletes the entry and returns the corresponding channel.
//
// You must hold the mutex lock before calling this.
func (hq *HeapQueue) getNextPushCh() chan *Item {
	for {
		if len(hq.waitingPops) == 0 {
			return nil
		}

		var id string
		id, hq.waitingPops = hq.waitingPops[0], hq.waitingPops[1:]

		if ch, exists := hq.pushChs[id]; exists {
			delete(hq.pushChs, id)

			return ch
		}
	}
}

// pop removes and returns the highest priority||size||oldest item in the queue.
//
// If there are currently no items in the queue, will wait for the context to be
// cancelled and return the next item push()ed to the queue before then, or if
// nothing gets pushed (or the context wasn't cancellable), nil.
func (hq *HeapQueue) pop(ctx context.Context) *Item {
	hq.mutex.Lock()

	done := ctx.Done()
	if hq.Len() == 0 {
		if done == nil {
			hq.mutex.Unlock()

			return nil
		}

		id, pushCh := hq.nextPushChannel()
		hq.mutex.Unlock()

		select {
		case item := <-pushCh:
			return item
		case <-done:
			return hq.readFromPushChannelIfSentOn(id, pushCh)
		}
	}

	defer hq.mutex.Unlock()

	return heap.Pop(hq).(*Item)
}

// nextPushChannel returns a channel that will be sent the next item push()ed.
//
// You must hold the mutex lock before calling this.
func (hq *HeapQueue) nextPushChannel() (string, chan *Item) {
	id := clog.UniqueID()
	hq.waitingPops = append(hq.waitingPops, id)

	ch := make(chan *Item, 1)
	hq.pushChs[id] = ch

	return id, ch
}

// readFromPushChannelIfSentOn checks if the given id no longer exists in
// pushChs, which by convention with getNextPushCh() means an item will be sent
// on the channel: in which case we read and return the item.
//
// Otherwise, we delete the id from pushChs, so that the next getNextPushCh()
// doesn't try and use this channel.
func (hq *HeapQueue) readFromPushChannelIfSentOn(id string, pushCh chan *Item) *Item {
	hq.mutex.Lock()
	if _, exists := hq.pushChs[id]; !exists {
		hq.mutex.Unlock()

		item := <-pushCh

		return item
	}

	delete(hq.pushChs, id)
	hq.mutex.Unlock()

	return nil
}

// remove removes a given item from the queue.
func (hq *HeapQueue) remove(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()

	if item.removed() || !item.belongsTo(hq) {
		return
	}

	heap.Remove(hq, item.index())
}

// update changes the item's position in the queue if its priority or size have
// changed. This implements the subQueue interface.
func (hq *HeapQueue) update(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()
	heap.Fix(hq, item.index())
}

// len returns the number of items in the queue.
func (hq *HeapQueue) len() int {
	hq.mutex.RLock()
	defer hq.mutex.RUnlock()

	return hq.Len()
}

// Len is to implement heap.Interface.
func (hq *HeapQueue) Len() int { return len(hq.items) }

// Less is to implement heap.Interface.
func (hq *HeapQueue) Less(i, j int) bool {
	ip := hq.items[i].Priority()
	jp := hq.items[j].Priority()

	if ip == jp {
		is := hq.items[i].Size()
		js := hq.items[j].Size()

		if is == js {
			return hq.items[i].Created().Before(hq.items[j].Created())
		}

		return is > js
	}

	return ip > jp
}

// Swap is to implement heap.Interface.
func (hq *HeapQueue) Swap(i, j int) {
	hq.items[i], hq.items[j] = hq.items[j], hq.items[i]
	hq.items[i].setIndex(i)
	hq.items[j].setIndex(j)
}

// Push is to implement heap.Interface.
func (hq *HeapQueue) Push(x interface{}) {
	n := len(hq.items)
	item := x.(*Item)
	item.setIndex(n)
	hq.items = append(hq.items, item)
}

// Pop is to implement heap.Interface.
func (hq *HeapQueue) Pop() interface{} {
	old := hq.items
	n := len(old)

	item := old[n-1]
	old[n-1] = nil
	hq.items = old[0 : n-1]

	item.remove()

	return item
}
