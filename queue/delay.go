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
	"fmt"
	"time"
)

// readyOrder implements heap.Interface, keeping items in readyAt order.
type readyOrder struct {
	items []*Item
}

// Len is to implement heap.Interface.
func (ro *readyOrder) Len() int { return len(ro.items) }

// Less is to implement heap.Interface.
func (ro *readyOrder) Less(i, j int) bool {
	return ro.items[i].ReadyAt().Before(ro.items[j].ReadyAt())
}

// Swap is to implement heap.Interface.
func (ro *readyOrder) Swap(i, j int) {
	fmt.Printf("ro.Swap called\n")
	heapSwap(ro.items, i, j)
	fmt.Printf("ro.Swap returning\n")
}

// Push is to implement heap.Interface.
func (ro *readyOrder) Push(x interface{}) {
	ro.items = heapPush(ro.items, x)
}

// Pop is to implement heap.Interface.
func (ro *readyOrder) Pop() interface{} {
	fmt.Printf("ro.Pop called\n")
	var item interface{}
	ro.items, item = heapPop(ro.items)
	fmt.Printf("ro.Pop returning\n")

	return item
}

// Next is to implement heapWithNext.
func (ro *readyOrder) Next() interface{} {
	return heapNext(ro.items)
}

// newDelaySubQueue creates a SubQueue that is ordered by readyAt and passes
// expired readyAt items to the given callback.
func newDelaySubQueue(cb func(*Item)) SubQueue {
	return newExpireSubQueue(func(*Item) bool {
		return true
	}, getItemReady, &readyOrder{})
}

// getItemReady is run SubQueue's itemTimeCB.
func getItemReady(item *Item) time.Time {
	return item.ReadyAt()
}
