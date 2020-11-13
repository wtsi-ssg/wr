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

// DefaultTTR is the time to release used for items that were specified with a
// 0 TTR.
const DefaultTTR = 5 * time.Second

const indexOfRemovedItem = -1

// ItemParameters describe an item you want to add to the queue.
//
// Key and Data are required to have a meaninful, retreivable item, but this is
// not enforced.
type ItemParameters struct {
	Key          string
	ReserveGroup string
	Data         interface{}
	Priority     uint8 // highest priority is 255
	Size         uint8
	TTR          time.Duration // if 0, defaults to DefaultTTR
}

// toItem creates an item based on our parameters.
func (ip *ItemParameters) toItem() *Item {
	return &Item{
		key:          ip.Key,
		reserveGroup: ip.ReserveGroup,
		data:         ip.Data,
		created:      time.Now(),
		priority:     ip.Priority,
		size:         ip.Size,
		ttr:          ip.TTR,
	}
}

// Item represents an item that was added to the queue. It will have the
// properties of an ItemParameters that was passed to Queue.Add().
type Item struct {
	key           string
	reserveGroup  string
	data          interface{}
	created       time.Time
	priority      uint8
	size          uint8
	ttr           time.Duration
	releaseAt     time.Time
	subQueue      SubQueue
	subQueueIndex int
	mutex         sync.RWMutex
}

// Key returns the key of this item.
func (item *Item) Key() string {
	return item.key
}

// ReserveGroup returns the ReserveGroup of this item.
func (item *Item) ReserveGroup() string {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.reserveGroup
}

// SetReserveGroup sets a new reserveGroup for the item.
func (item *Item) SetReserveGroup(group string) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.reserveGroup = group
}

// Data returns the data property of this item.
func (item *Item) Data() interface{} {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.data
}

// SetData sets new data for the item.
func (item *Item) SetData(data interface{}) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.data = data
}

// Created returns the time this item was created.
func (item *Item) Created() time.Time {
	return item.created
}

// Priority returns the priority of this item.
func (item *Item) Priority() uint8 {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.priority
}

// SetPriority sets a new priority for the item, and updates the SubQueue it
// belongs to.
func (item *Item) SetPriority(p uint8) {
	item.setAndUpdate(&item.priority, p)
}

// setAndUpdate sets a property and updates the SubQueue in a thread-safe way.
func (item *Item) setAndUpdate(property *uint8, new uint8) {
	item.mutex.Lock()
	*property = new
	sq := item.subQueue
	item.mutex.Unlock()

	if sq == nil {
		return
	}

	sq.update(item)
}

// Size returns the size of this item.
func (item *Item) Size() uint8 {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.size
}

// SetSize sets a new size for the item, and updates the SubQueue it belongs to.
func (item *Item) SetSize(s uint8) {
	item.setAndUpdate(&item.size, s)
}

// Touch updates the releaseAt for the item to now+TTR, and updates the
// SubQueue it belongs to.
func (item *Item) Touch() {
	item.mutex.Lock()

	var ttr time.Duration
	if item.ttr == 0 {
		ttr = DefaultTTR
	}

	item.releaseAt = time.Now().Add(ttr)
	sq := item.subQueue
	item.mutex.Unlock()

	if sq == nil {
		return
	}

	sq.update(item)
}

// ReleaseAt returns the time that this item's TTR will run out. It will be the
// zero time if this item has not yet been Touch()ed.
func (item *Item) ReleaseAt() time.Time {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.releaseAt
}

// setSubqueue sets a new SubQueue for the item.
func (item *Item) setSubqueue(sq SubQueue) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.subQueue = sq
}

// belongsTo tells you if the item is set to the given subQueue.
func (item *Item) belongsTo(sq SubQueue) bool {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.subQueue == sq
}

// index returns the index of this item in the subQueue it belongs to.
func (item *Item) index() int {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.subQueueIndex
}

// setIndex sets a new subQueueIndex for the item.
func (item *Item) setIndex(i int) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.subQueueIndex = i
}

// remove sets the subQueueIndex of the item to indexOfRemovedItem to indicate
// the item has been removed from its subQueue.
func (item *Item) remove() {
	item.setIndex(indexOfRemovedItem)
	item.setSubqueue(nil)
}

// removed returns true if the remove() was called more recently than
// setIndex().
func (item *Item) removed() bool {
	return item.index() == indexOfRemovedItem
}
