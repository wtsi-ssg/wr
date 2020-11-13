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
)

// prioritySizeAgeOrder implements heap.Interface, keeping items in
// priority||size||age order.
type prioritySizeAgeOrder struct {
	items []*Item
}

// newReadySubQueue creates a *heapQueue that is ordered by priority||size||age.
func newReadySubQueue() SubQueue {
	return newHeapQueue(&prioritySizeAgeOrder{})
}

// Len is to implement heap.Interface.
func (po *prioritySizeAgeOrder) Len() int { return len(po.items) }

// Less is to implement heap.Interface.
func (po *prioritySizeAgeOrder) Less(i, j int) bool {
	ip := po.items[i].Priority()
	jp := po.items[j].Priority()

	if ip == jp {
		is := po.items[i].Size()
		js := po.items[j].Size()

		if is == js {
			return po.items[i].Created().Before(po.items[j].Created())
		}

		return is > js
	}

	return ip > jp
}

// Swap is to implement heap.Interface.
func (po *prioritySizeAgeOrder) Swap(i, j int) {
	heapSwap(po.items, i, j)
}

// Push is to implement heap.Interface.
func (po *prioritySizeAgeOrder) Push(x interface{}) {
	po.items = heapPush(po.items, x)
}

// Pop is to implement heap.Interface.
func (po *prioritySizeAgeOrder) Pop() interface{} {
	var item interface{}
	po.items, item = heapPop(po.items)

	return item
}

// readyQueues is a slice of ready SubQueue that will newly create or reuse a
// SubQueue for each item.reserveGroup push()ed to it.
type readyQueues struct {
	queues map[string]SubQueue
	inUse  int
	mutex  sync.Mutex
}

// newReadyQueues creates a readyQueues.
func newReadyQueues() *readyQueues {
	return &readyQueues{
		queues: make(map[string]SubQueue),
	}
}

// push will newly create or reuse a SubQueue for the item's reserveGroup and
// push the item to that SubQueue.
func (rq *readyQueues) push(item *Item) {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	sq := rq.reuseOrCreateReadyQueueForReserveGroup(item.ReserveGroup())
	sq.push(item)
}

// reuseOrCreateReadyQueueForReserveGroup creates a new ready SubQueue for the
// given reserveGroup, or returns any existing one.
//
// You must hold the mutex lock before calling this.
func (rq *readyQueues) reuseOrCreateReadyQueueForReserveGroup(reserveGroup string) SubQueue {
	var (
		sq    SubQueue
		found bool
	)

	if sq, found = rq.queues[reserveGroup]; !found {
		sq = newReadySubQueue()
		rq.queues[reserveGroup] = sq
	}

	return sq
}

// pop will pop the next Item from the SubQueue corresponding to the given
// group.
func (rq *readyQueues) pop(ctx context.Context, reserveGroup string) *Item {
	sq := rq.getHeapQueueForUse(reserveGroup)
	defer rq.dropEmptyQueuesIfNotInUse()

	return sq.pop(ctx)
}

// getHeapQueueForUse calls reuseOrCreateReadyQueueForReserveGroup() and
// increments inUse. To be used in a paired call with
// dropEmptyQueuesIfNotInUse().
func (rq *readyQueues) getHeapQueueForUse(reserveGroup string) SubQueue {
	rq.mutex.Lock()
	sq := rq.reuseOrCreateReadyQueueForReserveGroup(reserveGroup)
	rq.inUse++
	rq.mutex.Unlock()

	return sq
}

// dropEmptyQueuesIfNotInUse removes empty SubQueues from our map if we are not
// in the middle of using any SubQueues.
func (rq *readyQueues) dropEmptyQueuesIfNotInUse() {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	rq.inUse--
	if rq.inUse > 0 {
		return
	}

	for reserveGroup, sq := range rq.queues {
		if sq.len() == 0 {
			delete(rq.queues, reserveGroup)
		}
	}
}

// remove will remove the given Item from the SubQueue corresponding to the
// item's reserveGroup.
func (rq *readyQueues) remove(item *Item) {
	sq := rq.getHeapQueueForUse(item.ReserveGroup())
	defer rq.dropEmptyQueuesIfNotInUse()
	sq.remove(item)
}

// changeItemReserveGroup atomically removes the item from one SubQueue and
// pushes it to another.
func (rq *readyQueues) changeItemReserveGroup(item *Item, newGroup string) {
	defer rq.dropEmptyQueuesIfNotInUse()
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	oldGroup := item.ReserveGroup()
	if oldGroup == newGroup {
		return
	}

	sqOld := rq.reuseOrCreateReadyQueueForReserveGroup(oldGroup)
	sqOld.remove(item)

	sqNew := rq.reuseOrCreateReadyQueueForReserveGroup(newGroup)
	item.SetReserveGroup(newGroup)
	sqNew.push(item)
	rq.inUse++
}
