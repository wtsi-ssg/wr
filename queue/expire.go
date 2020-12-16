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
	closeCh                chan struct{}
}

func newExpireSubQueue(expireCB expirationCB, timeCB itemTimeCB, newExpiringItems chan *Item, heapImplementation heap.Interface) SubQueue {
	esq := &expireSubQueue{
		expireCB:               expireCB,
		timeCB:                 timeCB,
		expireTime:             time.Now().Add(unsetItemExpiry),
		updateNextItemToExpire: newExpiringItems,
		closeCh:                make(chan struct{}),
	}

	hq := newHeapQueue(heapImplementation)
	esq.heapQueue = hq

	go esq.processExpiringItems()

	return esq
}

// newNextItemCB is our newNextItemCB, called when an Item becomes the next that
// would be pop()ed.
// func (esq *expireSubQueue) newNextItemCB(item *Item) {
// 	esq.expireMutex.RLock()
// 	fmt.Printf("checking %s vs %s\n", esq.expireTime, esq.timeCB(item))

// 	if esq.expireTime.After(esq.timeCB(item)) {
// 		fmt.Printf("expired!\n")
// 		esq.expireMutex.RUnlock()
// 		esq.updateNextItemToExpire <- item
// 	} else {
// 		esq.expireMutex.RUnlock()
// 	}
// }

// checkItemExpiry checks if the given item expires sooner than we are currently
// waiting for, and triggers an update of the wait time if so.
// func (esq *expireSubQueue) checkItemExpiry(item *Item) {
// 	esq.expireMutex.Lock()
// 	t := esq.timeCB(item)
// 	if item != nil {
// 		fmt.Printf("checkItemExpiry for %s\n", item.key)
// 	}
// 	if esq.expireTime.After(t) {
// 		if item != nil {
// 			fmt.Printf("checkItemExpiry item %s expires sooner\n", item.key)
// 		}
// 		esq.expireTime = t
// 		esq.expireMutex.Unlock()
// 		esq.updateNextItemToExpire <- item
// 	} else {
// 		esq.expireMutex.Unlock()
// 	}
// }

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
				fmt.Printf("sending %s\n", item.key)
			} else {
				fmt.Printf("sending nil\n")
			}

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

// itemExpires returns a timer for when the given item is supposed to expire. If
// the item is nil, timer effectively does not go off.
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
		fmt.Printf("sending item to expireCB\n")
		if remove := esq.expireCB(item); remove {
			fmt.Printf("item should be removed since expired\n")
			go esq.heapQueue.remove(item)
			fmt.Printf("item removed since expired\n")
		}
	}
}

// heapExpireSwap can be used to implement heap.Interface.Swap.
func heapExpireSwap(newExpiringItems chan *Item, items []*Item, i, j int) {
	fmt.Printf("heapExpireSwap called with %d items, i=%d;j=%d\n", len(items), i, j)
	heapSwap(items, i, j)
	fmt.Printf("normal heapSwap worked\n")
	sendIfItemExpiresNext(newExpiringItems, items[i], i)
	sendIfItemExpiresNext(newExpiringItems, items[j], j)
}

// heapExpirePush can be used to implement heap.Interface.Push.
func heapExpirePush(newExpiringItems chan *Item, items []*Item, x interface{}) []*Item {
	n := len(items)
	items = heapPush(items, x)

	defer sendIfItemExpiresNext(newExpiringItems, items[n], n)

	return items
}

// sendIfItemExpiresNext will send the given item down the given channel if the
// given number is 0.
func sendIfItemExpiresNext(newExpiringItems chan *Item, item *Item, n int) {
	if n == 0 {
		fmt.Printf("n is 0, so sending item %s with index %d down channel\n", item.key, item.subQueueIndex)
		newExpiringItems <- item
	}
}
