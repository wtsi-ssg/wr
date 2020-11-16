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
	"time"
)

// releaseOrder implements heap.Interface, keeping items in releaseAt order.
type releaseOrder struct {
	items []*Item
}

// Len is to implement heap.Interface.
func (ro *releaseOrder) Len() int { return len(ro.items) }

// Less is to implement heap.Interface.
func (ro *releaseOrder) Less(i, j int) bool {
	return ro.items[i].ReleaseAt().Before(ro.items[j].ReleaseAt())
}

// Swap is to implement heap.Interface.
func (ro *releaseOrder) Swap(i, j int) {
	heapSwap(ro.items, i, j)
}

// Push is to implement heap.Interface.
func (ro *releaseOrder) Push(x interface{}) {
	ro.items = heapPush(ro.items, x)
}

// Pop is to implement heap.Interface.
func (ro *releaseOrder) Pop() interface{} {
	var item interface{}
	ro.items, item = heapPop(ro.items)

	return item
}

// expirationCB is something that will be called when an item expires.
type expirationCB func(*Item)

// expireSubQueue is a heapQueue that deals with items that are older than
// one of their time.Time properties by passing them to a callback.
type expireSubQueue struct {
	*heapQueue
	expireCB               expirationCB
	expireTime             time.Time
	expireMutex            sync.RWMutex
	updateNextItemToExpire chan *Item
	closeCh                chan struct{}
}

// newRunSubQueue creates a SubQueue that is ordered by releaseAt and passes
// expired releaseAt items to the given callback.
func newRunSubQueue(expireCB expirationCB) SubQueue {
	esq := &expireSubQueue{
		expireCB:               expireCB,
		updateNextItemToExpire: make(chan *Item),
		closeCh:                make(chan struct{}),
	}

	hq := newHeapQueue(&releaseOrder{}, esq.newNextItemCB)
	esq.heapQueue = hq

	go esq.processExpiringItems()

	return esq
}

// newNextItemCB is our newNextItemCB, called when an Item becomes the next that
// would be pop()ed.
func (esq *expireSubQueue) newNextItemCB(item *Item) {
	esq.expireMutex.RLock()
	if esq.expireTime.After(item.ReleaseAt()) {
		esq.expireMutex.RUnlock()
		esq.updateNextItemToExpire <- item
	} else {
		esq.expireMutex.RUnlock()
	}
}

// processExpiringItems starts waiting for items to be Releasable() and calls
// our expireCB when they are.
func (esq *expireSubQueue) processExpiringItems() {
	var item *Item

	for {
		esq.expireMutex.Lock()

		var timeUntilNextExpire time.Duration
		if item != nil {
			timeUntilNextExpire = time.Until(item.ReleaseAt())
		} else {
			timeUntilNextExpire = 1 * time.Hour
		}

		esq.expireTime = time.Now().Add(timeUntilNextExpire)
		nextExpire := time.After(time.Until(esq.expireTime))
		esq.expireMutex.Unlock()

		select {
		case <-nextExpire:
			if item != nil && item.Releasable() {
				esq.expireCB(item)
			}

			item = nil
		case item = <-esq.updateNextItemToExpire:
			continue
		case <-esq.closeCh:
			return
		}
	}
}
