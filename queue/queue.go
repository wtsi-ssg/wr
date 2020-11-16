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

/*
Package queue provides an in-memory queue structure suitable for the safe and
low latency implementation of a real job queue.

It's like beanstalkd, but faster, with the ability to query the queue for
desired items, reject duplicates, and wait on dependencies.

Like beanstalkd, when you add items to the queue, they move between different
sub-queues:

Items start in the ready queue. From there you can Reserve() an item to get the
highest priority (or for those with equal priority, the largest, and for those
with equal size, the oldest - fifo) one which switches it from the ready queue
to the run queue.

Items can also have dependencies, in which case they start in the dependency
queue and only move to the ready queue once all its dependencies have been
Remove()d from the queue. Items can also belong to a reservation group, in which
case you can Reserve() an item in a desired group.

In the run queue the item starts a time-to-release (ttr) countdown; when that
runs out the item is placed (by default) in the delay queue. After the delay
period it will automatically switch to the ready queue. This is to handle a
process Reserving an item but then crashing before it deals with the item; with
it back on the ready queue, some other process can pick it up.

To stop it going to the delay queue you either Remove() the item (you dealt
with the item successfully), Touch() it to give yourself more time to handle the
item, or you Bury() the item (the item can't be dealt with until the user takes
some action). When you know you have a transient problem preventing you from
handling the item right now, you can manually Release() the item to the delay
queue.
*/
package queue

import (
	"context"
	"sync"
	"time"

	"github.com/wtsi-ssg/wr/clog"
)

// SubQueue is something that an Item belongs to, which stores the item in a
// certain order for later retrieval.
type SubQueue interface {
	// push adds an item to the queue.
	push(*Item)

	// pop removes and returns an item in the queue based on a certain order.
	pop(context.Context) *Item

	// peek returns the next item that pop() would return, but without actually
	// removing it from the SubQueue
	peek() *Item

	// remove removes a given item from the queue.
	remove(*Item)

	// update changes the item's position in the queue if relevant item
	// properties have changed.
	update(*Item)

	// len returns the number of items in the queue.
	len() int

	// newNextItem should be sent an item when it newly becomes the next item
	// that would be pop()ed.
	newNextItem(*Item)
}

// Queue is an in-memory poll-free queue with various heap-based ordered
// SubQueues for managing item progress.
type Queue struct {
	items             map[string]*Item
	itemsMutex        sync.RWMutex
	readyQueues       *readyQueues
	runQueue          SubQueue
	releaseTime       time.Time
	releaseMutex      sync.RWMutex
	updateReleaseTime chan bool
	delayQueue        SubQueue
	close             chan struct{}
}

// New returns a new *Queue.
func New() *Queue {
	q := &Queue{
		items:             make(map[string]*Item),
		readyQueues:       newReadyQueues(),
		runQueue:          newRunSubQueue(func(*Item) {}),
		updateReleaseTime: make(chan bool),
		delayQueue:        newRunSubQueue(func(*Item) {}),
		close:             make(chan struct{}),
	}

	go q.processTTRItems()

	return q
}

// Add creates items with the given parameters and adds them to the queue.
//
// It tells you how many items were really added just now, and how many
// ItemParameters had a Key that was already in the queue, and were therefore
// ignored.
//
// If it was added, an item will be in the ready sub-queue and can be
// Reserve()d.
func (q *Queue) Add(params ...*ItemParameters) (added, dups int) {
	for _, p := range params {
		item := p.toItem()
		q.addToItemsIfNotDuplicate(item, &added, &dups)
	}

	return added, dups
}

// addToItemsIfNotDuplicate adds the item to the items maps if it isn't a
// duplicate.
func (q *Queue) addToItemsIfNotDuplicate(item *Item, added, dups *int) {
	q.threadSafeItemsWriteOperation(func() {
		if _, exists := q.items[item.Key()]; exists {
			*dups++
		} else {
			q.items[item.Key()] = item
			q.readyQueues.push(item)
			*added++
		}
	})
}

// operation is a function that we want to wrap to make thread safe.
type operation func()

// threadSafeItemsWriteOperation wraps the given function in a mutex lock and
// unlock.
func (q *Queue) threadSafeItemsWriteOperation(op operation) {
	q.itemsMutex.Lock()
	op()
	q.itemsMutex.Unlock()
}

// Get searches for and returns the item with the given key. If one doesn't
// exist, returns nil.
func (q *Queue) Get(key string) *Item {
	q.itemsMutex.RLock()
	defer q.itemsMutex.RUnlock()

	return q.items[key]
}

// Reserve is a way to get the highest priority (or for those with equal
// priority, the largest, or for those with equal size, the oldest (by time
// since the item was first Add()ed) to the ready sub-queue, switching it from
// the ready sub-queue to the run sub-queue, and in so doing starting its ttr
// countdown.
//
// If the context is cancellable, we will wait until it is cancelled for an item
// to appear in the ready sub-queue, if at least 1 isn't already there. If after
// this time there is still nothing in the ready sub-queue, a nil item is
// returned. Use a context.Background() to not wait.
//
// You will only get an item that was Add()ed with the given ReserveGroup.
//
// You need to Remove() the item when you're done with it. If you're still doing
// something and ttr is approaching, Touch() it, otherwise it will be assumed
// you died and the item will be released to the delay sub-queue automatically,
// to be handled by someone else that gets it from a Reserve() call. If you know
// you can't handle it right now, but someone else might be able to later, you
// can manually call Release(), which moves it to the delay sub-queue.
func (q *Queue) Reserve(ctx context.Context, reserveGroup string) *Item {
	item := q.readyQueues.pop(ctx, reserveGroup)
	q.pushToRunQueue(ctx, item)

	return item
}

// pushToRunQueue Touch()es the item and pushes it to the runQueue, if non-nil
// and allowed.
func (q *Queue) pushToRunQueue(ctx context.Context, item *Item) {
	if item == nil {
		return
	}

	err := item.SwitchState(ItemStateRun)
	if err != nil {
		clog.Error(ctx, "queue failure", "err", err)

		return
	}

	q.runQueue.push(item)
	q.updateRelease(item)
}

// Remove removes an item from the queue.
func (q *Queue) Remove(key string) {
	q.threadSafeItemsWriteOperation(func() {
		if item, exists := q.items[key]; exists {
			delete(q.items, key)
			q.readyQueues.remove(item)
		}
	})
}

// ChangeReserveGroup changes the ReserveGroup of an item given its key, so that
// the next time it is Reserve()ed, you would have had to have supplied the
// given group to get it.
func (q *Queue) ChangeReserveGroup(key string, newGroup string) {
	q.threadSafeItemsWriteOperation(func() {
		if item, exists := q.items[key]; exists {
			// *** item.doIfInState(ItemStateReady)
			q.readyQueues.changeItemReserveGroup(item, newGroup)
		}
	})
}

// processTTRItems starts waiting for run items to be released and switches them
// to the delay SubQueue.
func (q *Queue) processTTRItems() {
	// *** this stuff should happen in run SubQueue code. Creating a run
	// SubQueue should return an item channel that we read from here.
	for {
		q.releaseMutex.Lock()

		var timeUntilNextRelease time.Duration
		if q.runQueue.len() > 0 {
			timeUntilNextRelease = time.Until(q.runQueue.peek().ReleaseAt())
		} else {
			timeUntilNextRelease = 1 * time.Hour
		}

		q.releaseTime = time.Now().Add(timeUntilNextRelease)
		nextRelease := time.After(time.Until(q.releaseTime))
		q.releaseMutex.Unlock()

		select {
		case <-nextRelease:
			length := q.runQueue.len()
			for i := 0; i < length; i++ {
				item := q.runQueue.peek()

				if !item.Releasable() {
					break
				}

				q.runQueue.remove(item)

				if err := item.SwitchState(ItemStateDelay); err != nil {
					clog.Error(context.Background(), "queue failure", "err", err)

					return
				}

				q.delayQueue.push(item)
			}
		case <-q.updateReleaseTime:
			continue
		case <-q.close:
			return
		}
	}
}

// updateRelease checks if this item's ReleaseAt() is before the next time we
// are going to check for a released item, and if so triggers processTTRItems()
// to check at the earlier time.
func (q *Queue) updateRelease(item *Item) {
	q.releaseMutex.RLock()
	if q.releaseTime.After(item.ReleaseAt()) {
		q.releaseMutex.RUnlock()
		q.updateReleaseTime <- true
	} else {
		q.releaseMutex.RUnlock()
	}
}

// Close should be called when you're done with a Queue to stop TTR and Delay
// processing. Don't use the queue after calling this, since TTR and Delay will
// no longer work, but you won't get any errors.
func (q *Queue) Close() {
	close(q.close)
}
