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
	"sync"
	"time"
)

// DefaultTTR is the time to release used for items that were specified with a
// 0 TTR.
const DefaultTTR = 5 * time.Second

// DefaultDelay is the time spent in the delay SubQueue for items that were
// specified with a 0 Delay.
const DefaultDelay = 5 * time.Second

const indexOfRemovedItem = -1

// ItemState is how we describe the possible item states.
type ItemState string

// ItemState* constants represent all the possible item states.
const (
	ItemStateReady     ItemState = "ready"
	ItemStateRun       ItemState = "run"
	ItemStateDelay     ItemState = "delay"
	ItemStateBury      ItemState = "bury"
	ItemStateDependent ItemState = "dependent"
	ItemStateRemoved   ItemState = "removed"
)

// ItemTransitionError is returned by Item.SwitchState() for invalid state
// transitions.
type ItemTransitionError struct {
	item string
	from ItemState
	to   ItemState
}

// Error implements the error interface.
func (e *ItemTransitionError) Error() string {
	return fmt.Sprintf("item %s cannot transition from %s to %s", e.item, e.from, e.to)
}

// checkStateTransition returns a function that returns an error if you're not
// allowed to transition from ItemState a to b.
func checkStateTransition() func(item *Item, a, b ItemState) error {
	rdr := map[ItemState]bool{
		ItemStateReady:     true,
		ItemStateDependent: true,
		ItemStateRemoved:   true,
	}
	constMap := map[ItemState]map[ItemState]bool{
		ItemStateReady: {
			ItemStateRun:       true,
			ItemStateDependent: true,
			ItemStateRemoved:   true,
		},
		ItemStateRun: {
			ItemStateDelay:     true,
			ItemStateBury:      true,
			ItemStateDependent: true,
		},
		ItemStateDelay:     rdr,
		ItemStateBury:      rdr,
		ItemStateDependent: {ItemStateReady: true, ItemStateRemoved: true},
		ItemStateRemoved:   {},
	}

	return func(item *Item, a, b ItemState) error {
		if _, exists := constMap[a][b]; !exists {
			return &ItemTransitionError{item.key, a, b}
		}

		return nil
	}
}

var stateTransitionChecker = checkStateTransition()

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
	Delay        time.Duration // if 0, defaults to DebfaultDelay
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
		delay:        ip.Delay,
		ttr:          ip.TTR,
		state:        ItemStateReady,
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
	delay         time.Duration
	ttr           time.Duration
	readyAt       time.Time
	releaseAt     time.Time
	subQueue      SubQueue
	subQueueIndex int
	state         ItemState
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
	item.setAndUpdateUint(&item.priority, p)
}

// setAndUpdateUint sets a uint property and updates the SubQueue.
func (item *Item) setAndUpdateUint(property *uint8, new uint8) {
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
	item.setAndUpdateUint(&item.size, s)
}

// Touch updates the releaseAt for the item to now+TTR, and updates the
// SubQueue it belongs to.
func (item *Item) Touch() {
	item.setAndUpdateTime(item.ttr, DefaultTTR, &item.releaseAt)
}

// setAndUpdateTime sets a time property and updates the SubQueue.
func (item *Item) setAndUpdateTime(d, defaultD time.Duration, property *time.Time) {
	item.mutex.Lock()

	if d == 0 {
		d = defaultD
	}

	*property = time.Now().Add(d)
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

// restart updates the readyAt for the item to now+delay, and updates the
// SubQueue it belongs to, for when the item is put in to the delay SubQueue.
func (item *Item) restart() {
	item.setAndUpdateTime(item.delay, DefaultDelay, &item.readyAt)
}

// ReadyAt returns the time that this item can go to the ready SubQueue. It will
// be the zero time if this item is not in the delay SubQueue.
func (item *Item) ReadyAt() time.Time {
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	return item.readyAt
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

// State returns the state of the item.
func (item *Item) State() ItemState {
	item.mutex.RLock()
	defer item.mutex.RUnlock()
	return item.state
}

// SwitchState switches the item's state from the current state to the given
// state. Returns an error if this transition isn't possible.
func (item *Item) SwitchState(to ItemState) error {
	item.mutex.Lock()

	if err := stateTransitionChecker(item, item.state, to); err != nil {
		item.mutex.Unlock()

		return err
	}

	item.state = to

	switch to {
	case ItemStateRun:
		item.mutex.Unlock()
		item.Touch()
	case ItemStateDelay:
		item.mutex.Unlock()
		item.restart()
	case ItemStateReady, ItemStateBury, ItemStateDependent, ItemStateRemoved:
		item.releaseAt = time.Time{}
		item.readyAt = time.Time{}
		item.mutex.Unlock()
	}

	return nil
}
