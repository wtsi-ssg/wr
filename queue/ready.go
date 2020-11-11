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

// readyQueues is a slice of *HeapQueue that will newly create or reuse a
// HeapQueue for each item.reserveGroup push()ed to it.
type readyQueues struct {
	queues map[string]*HeapQueue
	inUse  int
	mutex  sync.Mutex
}

// newReadyQueues creates a readyQueues.
func newReadyQueues() *readyQueues {
	return &readyQueues{
		queues: make(map[string]*HeapQueue),
	}
}

// push will newly create or reuse a *HeapQueue for the item's reserveGroup and
// push the item to that HeapQueue.
func (rq *readyQueues) push(item *Item) {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	hq := rq.reuseOrCreateHeapQueueForReserveGroup(item.ReserveGroup())
	hq.push(item)
}

// reuseOrCreateHeapQueueForReserveGroup creates a new HeapQueue for the given
// reserveGroup, or returns any existing one.
//
// You must hold the mutex lock before calling this.
func (rq *readyQueues) reuseOrCreateHeapQueueForReserveGroup(reserveGroup string) *HeapQueue {
	var (
		hq    *HeapQueue
		found bool
	)

	if hq, found = rq.queues[reserveGroup]; !found {
		hq = NewHeapQueue()
		rq.queues[reserveGroup] = hq
	}

	return hq
}

// pop will pop the next Item from the HeapQueue corresponding to the given
// group.
func (rq *readyQueues) pop(ctx context.Context, reserveGroup string) *Item {
	hq := rq.getHeapQueueForUse(reserveGroup)
	defer rq.dropEmptyQueuesIfNotInUse()

	return hq.pop(ctx)
}

// getHeapQueueForUse calls reuseOrCreateHeapQueueForReserveGroup() and
// increments inUse. To be used in a paired call with
// dropEmptyQueuesIfNotInUse().
func (rq *readyQueues) getHeapQueueForUse(reserveGroup string) *HeapQueue {
	rq.mutex.Lock()
	hq := rq.reuseOrCreateHeapQueueForReserveGroup(reserveGroup)
	rq.inUse++
	rq.mutex.Unlock()

	return hq
}

// dropEmptyQueuesIfNotInUse removes empty HeapQueues from our map if they
// are empty and we are not in the middle of using any queues.
func (rq *readyQueues) dropEmptyQueuesIfNotInUse() {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	rq.inUse--
	if rq.inUse > 0 {
		return
	}

	for reserveGroup, hq := range rq.queues {
		if hq.len() == 0 {
			delete(rq.queues, reserveGroup)
		}
	}
}

// remove will remove the given Item from the HeapQueue corresponding to the
// item's reserveGroup.
func (rq *readyQueues) remove(item *Item) {
	hq := rq.getHeapQueueForUse(item.ReserveGroup())
	defer rq.dropEmptyQueuesIfNotInUse()
	hq.remove(item)
}
