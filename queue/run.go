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

import "time"

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

// Peek is to implement heapWithPeek.
func (ro *releaseOrder) Peek(n int) *Item {
	return heapPeek(ro.items, n)
}

// newRunSubQueue creates a SubQueue that is ordered by releaseAt and passes
// expired releaseAt items to the given callback.
func newRunSubQueue(expireCB expirationCB) SubQueue {
	return newExpireSubQueue(expireCB, getItemRelease, &releaseOrder{})
}

// getItemRelease is run SubQueue's itemTimeCB.
func getItemRelease(item *Item) time.Time {
	return item.ReleaseAt()
}
