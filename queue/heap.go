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

	sync "github.com/sasha-s/go-deadlock"

	"github.com/wtsi-ssg/wr/clog"
)

// heapWithNext is the heap.Interface interface with an additional Next()
// method.
type heapWithNext interface {
	heap.Interface

	// Next returns the next item that would be Pop()ed without actually
	// removing it.
	Next() interface{}
}

// heapQueue holds Items in a heap. It implements the SubQueue interface.
type heapQueue struct {
	pushChs            map[string]chan *Item
	waitingPops        []string
	heapImplementation heapWithNext
	mutex              sync.RWMutex
}

// newHeapQueue returns an initialised heap-based queue.
func newHeapQueue(heapImplementation heapWithNext) *heapQueue {
	hq := &heapQueue{
		pushChs:            make(map[string]chan *Item),
		heapImplementation: heapImplementation,
	}

	heap.Init(hq.heapImplementation)

	return hq
}

// push adds an item to the queue.
func (hq *heapQueue) push(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()
	defer hq.popIfWaiting()

	item.setSubqueue(hq)
	heap.Push(hq.heapImplementation, item)
}

// popIfWaiting pops the item we just pushed and sends it down the next pushCh,
// if any exist.
//
// You must hold the mutex lock before calling this.
func (hq *heapQueue) popIfWaiting() {
	pushCh := hq.getNextPushCh()
	if pushCh == nil {
		return
	}
	pushCh <- heap.Pop(hq.heapImplementation).(*Item)
}

// getNextPushCh finds the oldest waitingPops id that still has a pushChs
// entry, deletes the entry and returns the corresponding channel.
//
// You must hold the mutex lock before calling this.
func (hq *heapQueue) getNextPushCh() chan *Item {
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

// pop removes and returns the next item in the queue.
//
// If there are currently no items in the queue, will wait for the context to be
// cancelled and return the next item push()ed to the queue before then, or if
// nothing gets pushed (or the context wasn't cancellable), nil.
func (hq *heapQueue) pop(ctx context.Context) *Item {
	hq.mutex.Lock()

	done := ctx.Done()
	if hq.heapImplementation.Len() == 0 {
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

	return heap.Pop(hq.heapImplementation).(*Item)
}

// nextPushChannel returns a channel that will be sent the next item push()ed.
//
// You must hold the mutex lock before calling this.
func (hq *heapQueue) nextPushChannel() (string, chan *Item) {
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
func (hq *heapQueue) readFromPushChannelIfSentOn(id string, pushCh chan *Item) *Item {
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
func (hq *heapQueue) remove(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()

	if item.removed() || !item.belongsTo(hq) {
		return
	}

	heap.Remove(hq.heapImplementation, item.index())
}

// update changes the item's position in the queue if its order properties have
// changed.
func (hq *heapQueue) update(item *Item) {
	hq.mutex.Lock()
	defer hq.mutex.Unlock()
	heap.Fix(hq.heapImplementation, item.index())
}

// len returns the number of items in the queue.
func (hq *heapQueue) len() int {
	hq.mutex.RLock()
	defer hq.mutex.RUnlock()

	return hq.heapImplementation.Len()
}

// nextItem returns the next item in the queue that would be pop()ed.
func (hq *heapQueue) nextItem() *Item {
	hq.mutex.RLock()
	defer hq.mutex.RUnlock()

	next := hq.heapImplementation.Next()
	if next == nil {
		return nil
	}

	return next.(*Item)
}

// basicHeapWithNext implements most of the methods of heapWithNext interface.
// Just embed and add a Less method to complete.
type basicHeapWithNext struct {
	items []*Item
}

// Len is to implement heap.Interface.
func (h *basicHeapWithNext) Len() int { return len(h.items) }

// Swap is to implement heap.Interface.
func (h *basicHeapWithNext) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].setIndex(i)
	h.items[j].setIndex(j)
}

// Push is to implement heap.Interface.
func (h *basicHeapWithNext) Push(x interface{}) {
	n := len(h.items)
	item, ok := x.(*Item)

	if !ok {
		panic("basicHeapWithNext.Push got an item that wasn't an Item")
	}

	item.setIndex(n)

	h.items = append(h.items, item)
}

// Pop is to implement heap.Interface.
func (h *basicHeapWithNext) Pop() interface{} {
	n := len(h.items)

	item := h.items[n-1]
	h.items[n-1] = nil
	h.items = h.items[0 : n-1]

	item.remove()

	return item
}

// Next is to implement heapWithNext.
func (h *basicHeapWithNext) Next() interface{} {
	if len(h.items) == 0 {
		return nil
	}

	return h.items[0]
}
