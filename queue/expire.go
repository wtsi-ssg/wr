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

// expireSubQueue is a heapQueue that deals with items that are older than
// one of their time.Time properties by passing them to a callback.
type expireSubQueue struct {
	*heapQueue
	expireCB               expirationCB
	timeCB                 itemTimeCB
	expireTime             time.Time
	expireMutex            sync.RWMutex
	updateNextItemToExpire chan *Item
	closeCh                chan struct{}
}

func newExpireSubQueue(expireCB expirationCB, timeCB itemTimeCB, heapImplementation heap.Interface) SubQueue {
	esq := &expireSubQueue{
		expireCB:               expireCB,
		timeCB:                 timeCB,
		expireTime:             time.Now().Add(unsetItemExpiry),
		updateNextItemToExpire: make(chan *Item),
		closeCh:                make(chan struct{}),
	}

	hq := newHeapQueue(heapImplementation, esq.newNextItemCB)
	esq.heapQueue = hq

	go esq.processExpiringItems()

	return esq
}

// newNextItemCB is our newNextItemCB, called when an Item becomes the next that
// would be pop()ed.
func (esq *expireSubQueue) newNextItemCB(item *Item) {
	esq.expireMutex.RLock()
	fmt.Printf("checking %s vs %s\n", esq.expireTime, esq.timeCB(item))
	if esq.expireTime.After(esq.timeCB(item)) {
		fmt.Printf("expired!\n")
		esq.expireMutex.RUnlock()
		esq.updateNextItemToExpire <- item
	} else {
		esq.expireMutex.RUnlock()
	}
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
			fmt.Printf("sending %+v\n", item)
			itemExpires.Stop()
			esq.sendItemToExpireCB(item)
			item = nil
		case item = <-esq.updateNextItemToExpire:
			fmt.Printf("read\n")
			itemExpires.Stop()
		case <-esq.closeCh:
			itemExpires.Stop()

			return
		}
	}
}

// itemExpires returns a channel that is sent on when the given item is
// supposed to expire. If the item is nil, effectively never sends.
func (esq *expireSubQueue) itemExpires(item *Item) *time.Timer {
	esq.expireMutex.Lock()
	defer esq.expireMutex.Unlock()

	esq.expireTime = esq.timeCB(item)
	fmt.Printf("making timer that expires at %s\n", esq.expireTime)

	return time.NewTimer(time.Until(esq.expireTime))
}

// sendItemToExpireCB sends non-nil items to our expireCB.
func (esq *expireSubQueue) sendItemToExpireCB(item *Item) {
	if item != nil {
		esq.expireCB(item)
	}
}
