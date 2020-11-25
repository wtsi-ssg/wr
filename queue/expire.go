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
	"fmt"
	"sync"
	"time"
)

// expirationCB is something that will be called when an item expires.
type expirationCB func(*Item)

// itemTimeCB is something that will be called to find out if an item has
// expired.
type itemTimeCB func(*Item) time.Time

type heapWithPeek interface {
	heap.Interface

	// Peek returns the nth item them would be Pop()ed, or nil if there are not
	// that many items.
	Peek(n int) *Item
}

// expireSubQueue is a heapQueue that deals with items that are older than
// one of their time.Time properties by passing them to a callback.
type expireSubQueue struct {
	*heapQueue
	peeker                 heapWithPeek
	expireCB               expirationCB
	timeCB                 itemTimeCB
	expireTime             time.Time
	expireMutex            sync.RWMutex
	updateNextItemToExpire chan *Item
	closeCh                chan struct{}
}

func newExpireSubQueue(expireCB expirationCB, timeCB itemTimeCB, peeker heapWithPeek) SubQueue {
	esq := &expireSubQueue{
		peeker:                 peeker,
		expireCB:               expireCB,
		timeCB:                 timeCB,
		expireTime:             time.Now().Add(unsetItemExpiry),
		updateNextItemToExpire: make(chan *Item),
		closeCh:                make(chan struct{}),
	}

	hq := newHeapQueue(peeker)
	esq.heapQueue = hq

	go esq.processExpiringItems()

	return esq
}

// push is like heapQueue's push(), but we also consider the item's expiry.
func (esq *expireSubQueue) push(item *Item) {
	esq.heapQueue.push(item)
	esq.checkItemExpiry(item)
}

// checkItemExpiry checks if the given item expires sooner than we are currently
// waiting for, and triggers an update of the wait time if so.
func (esq *expireSubQueue) checkItemExpiry(item *Item) {
	esq.expireMutex.Lock()
	t := esq.timeCB(item)
	if item != nil {
		fmt.Printf("checkItemExpiry for %s\n", item.key)
	}
	if esq.expireTime.After(t) {
		if item != nil {
			fmt.Printf("checkItemExpiry item %s expires sooner\n", item.key)
		}
		esq.expireTime = t
		esq.expireMutex.Unlock()
		esq.updateNextItemToExpire <- item
	} else {
		esq.expireMutex.Unlock()
	}
}

func (esq *expireSubQueue) peek(n int) *Item {
	esq.heapQueue.mutex.RLock()
	defer esq.heapQueue.mutex.RUnlock()

	return esq.peeker.Peek(n)
}

// update is like heapQueue's update(), but we also consider the item's expiry.
// func (esq *expireSubQueue) update(item *Item) {
// 	esq.heapQueue.update(item)
// 	esq.checkItemExpiry(item)
// }

// processExpiringItems starts waiting for items to expire and calls our
// expireCB when they are.
func (esq *expireSubQueue) processExpiringItems() {
	fmt.Printf("processing\n")
	item := <-esq.updateNextItemToExpire

	peek := 0
	for {
		itemExpires := esq.itemExpires(item)

		select {
		case <-itemExpires.C:
			if item != nil {
				fmt.Printf("sending %s\n", item.key)
			} else {
				fmt.Printf("sending nil\n")
			}

			itemExpires.Stop()
			esq.sendItemToExpireCB(item)
			peek++
			item = esq.peek(peek)
		case item = <-esq.updateNextItemToExpire:
			fmt.Printf("read\n")
			itemExpires.Stop()
			peek = 0
		case <-esq.closeCh:
			itemExpires.Stop()

			return
		}
	}
}

// itemExpires returns a timer for when the given item is supposed to expire. If
// the item is nil, effectively never sends.
func (esq *expireSubQueue) itemExpires(item *Item) *time.Timer {
	esq.expireMutex.Lock()
	defer esq.expireMutex.Unlock()

	esq.expireTime = esq.timeCB(item)
	if item != nil {
		fmt.Printf("expireTime set based on %s\n", item.key)
	}

	return time.NewTimer(time.Until(esq.expireTime))
}

// sendItemToExpireCB sends non-nil items to our expireCB.
func (esq *expireSubQueue) sendItemToExpireCB(item *Item) {
	if item != nil {
		esq.expireCB(item)
	}
}

func heapPeek(items []*Item, n int) *Item {
	if len(items) <= n {
		return nil
	}

	return items[n]
}
