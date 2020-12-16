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
	"fmt"
	"sync"
	"time"
)

// expirationCB is something that will be called when an item expires. It should
// return true if the item should be removed from the expire SubQueue, and false
// if it should instead have its expiry effectively ignored.
type expirationCB func(*Item) bool

// itemTimeCB is something that will be called to find out if an item has
// expired.
type itemTimeCB func(*Item) time.Time

// expireSubQueue is a heapQueue that deals with items that are older than
// one of their time.Time properties by passing them to a callback.
type expireSubQueue struct {
	*heapQueue
	expireCB               expirationCB
	timeCB                 itemTimeCB
	expireTime             time.Time
	expireMutex            sync.RWMutex
	updateNextItemToExpire chan *Item
	nextExpiringItem       *Item
	closeCh                chan struct{}
}

func newExpireSubQueue(expireCB expirationCB, timeCB itemTimeCB, heapImplementation heapWithNext) SubQueue {
	esq := &expireSubQueue{
		expireCB:               expireCB,
		timeCB:                 timeCB,
		expireTime:             time.Now().Add(unsetItemExpiry),
		updateNextItemToExpire: make(chan *Item),
		closeCh:                make(chan struct{}),
	}

	hq := newHeapQueue(heapImplementation)
	esq.heapQueue = hq

	go esq.processExpiringItems()

	return esq
}

// push adds an item to the queue.
func (esq *expireSubQueue) push(item *Item) {
	esq.expireMutex.Lock()
	esq.heapQueue.push(item)
	fmt.Printf("consider next due to push\n")
	// if many pushes during the time period we are removing an expired item,
	// we will end up sending the expired item to be processed after we removed
	// it
	esq.considerNextExpiringItem()
}

// considerNextExpiringItem will trigger processing of the next item that would
// be popped. You must hold the expireMutex lock before calling this.
func (esq *expireSubQueue) considerNextExpiringItem() {
	item := esq.heapQueue.nextItem()
	beingConsidered := esq.nextExpiringItem
	esq.expireMutex.Unlock()

	if item == nil {
		return
	}

	if beingConsidered != nil {
		if item.Key() == beingConsidered.Key() {
			return
		}
	}

	fmt.Printf("sending item %s with index %d down channel\n", item.key, item.subQueueIndex)
	esq.updateNextItemToExpire <- item
}

// pop removes and returns the next item in the queue, waiting like heapQueue.
func (esq *expireSubQueue) pop(ctx context.Context) *Item {
	esq.expireMutex.Lock()
	item := esq.heapQueue.pop(ctx)
	fmt.Printf("consider next due to pop\n")
	esq.considerNextExpiringItem()

	return item
}

// remove removes a given item from the queue.
func (esq *expireSubQueue) remove(item *Item) {
	esq.expireMutex.Lock()
	esq.heapQueue.remove(item)
	fmt.Printf("consider next due to remove\n")
	esq.considerNextExpiringItem()
}

// update changes the item's position in the queue if its order properties have
// changed.
func (esq *expireSubQueue) update(item *Item) {
	esq.expireMutex.Lock()
	esq.heapQueue.update(item)
	fmt.Printf("consider next due to update\n")
	esq.considerNextExpiringItem()
}

// processExpiringItems starts waiting for items to expire and calls our
// expireCB when they are.
func (esq *expireSubQueue) processExpiringItems() {
	fmt.Printf("processing\n")

	item := <-esq.updateNextItemToExpire

	for {
		itemExpires := esq.itemExpires(item)

		select {
		case <-itemExpires.C:
			if item != nil {
				fmt.Printf("item %s just expired\n", item.key)
			} else {
				fmt.Printf("item nil just expired\n")
			}

			itemExpires.Stop()
			item = esq.sendItemToExpireCB(item)
		case item = <-esq.updateNextItemToExpire:
			ik := "nil"
			if item != nil {
				ik = item.Key()
			}
			fmt.Printf("item %s was sent down updateNextItemToExpire ch\n", ik)
			itemExpires.Stop()
		case <-esq.closeCh:
			itemExpires.Stop()

			return
		}
	}
}

// itemExpires returns a timer for when the given item is supposed to expire. If
// the item is nil, timer effectively does not go off.
func (esq *expireSubQueue) itemExpires(item *Item) *time.Timer {
	esq.expireMutex.Lock()
	defer esq.expireMutex.Unlock()

	esq.expireTime = esq.timeCB(item)
	esq.nextExpiringItem = item

	if item != nil {
		fmt.Printf("expireTime set based on %s\n", item.key)
	}

	return time.NewTimer(time.Until(esq.expireTime))
}

// sendItemToExpireCB sends non-nil items to our expireCB and returns the next
// item to expire.
func (esq *expireSubQueue) sendItemToExpireCB(item *Item) *Item {
	if item != nil {
		fmt.Printf("sending item to expireCB\n")
		if remove := esq.expireCB(item); remove {
			fmt.Printf("item should be removed since expired\n")
			esq.expireMutex.Lock()
			esq.heapQueue.remove(item)
			next := esq.heapQueue.nextItem()
			esq.expireMutex.Unlock()
			nk := "nil"
			if next != nil {
				nk = next.Key()
			}
			fmt.Printf("item %s removed since expired, next to expire is %s\n", item.Key(), nk)
			return next
		}
	}

	return nil
}
