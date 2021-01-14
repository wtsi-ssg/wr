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

// releaseOrder implements heapWithNext, keeping items in releaseAt order.
type releaseOrder struct {
	*basicHeapWithNext
}

func newReleaseOrder() *releaseOrder {
	return &releaseOrder{basicHeapWithNext: &basicHeapWithNext{}}
}

// Less is to implement heap.Interface.
func (ro *releaseOrder) Less(i, j int) bool {
	return ro.items[i].ReleaseAt().Before(ro.items[j].ReleaseAt())
}

// newRunSubQueue creates a SubQueue that is ordered by releaseAt and passes
// expired releaseAt items to the given callback.
func newRunSubQueue(cb func(*Item)) SubQueue {
	return newExpireSubQueue(func(item *Item) bool {
		fmt.Printf("newRunSubQueue cb will be called\n")
		cb(item)

		return true
	}, getItemRelease, newReleaseOrder())
}

// getItemRelease is run SubQueue's itemTimeCB.
func getItemRelease(item *Item) time.Time {
	return item.ReleaseAt()
}
