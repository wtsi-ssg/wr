/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>.
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

package limiter

// This file contains the implementation of the group stuct.

// group struct describes an individual limit group.
type group struct {
	name     string
	limit    uint
	current  uint
	toNotify []chan bool
}

// newGroup creates a new group.
func newGroup(name string, limit uint) *group {
	return &group{
		name:  name,
		limit: limit,
	}
}

// setLimit updates the group's limit.
func (g *group) setLimit(limit uint) {
	g.limit = limit
}

// canIncrement tells you if the current count of this group is less than the
// limit.
func (g *group) canIncrement() bool {
	return g.current < g.limit
}

// increment increases the current count of this group. You must call
// canIncrement() first to make sure you won't go over the limit (and hold a
// lock over the 2 calls to avoid a race condition).
func (g *group) increment() {
	g.current++
}

// decrement decreases the current count of this group. Returns true if the
// current count indicates the group is unused.
func (g *group) decrement() bool {
	// (decrementing a uint under 0 makes it a large positive value, so we must
	// check first)
	if g.current == 0 {
		return true
	}

	g.current--

	// notify callers who passed a channel to notifyDecrement(), but do it
	// defensively in a go routine so if the caller doesn't read from the
	// channel, we don't block forever
	if len(g.toNotify) > 0 {
		chans := g.toNotify
		g.toNotify = []chan bool{}

		go func() {
			for _, ch := range chans {
				ch <- true
			}
		}()
	}

	return g.current < 1
}

// capacity tells you how many more increments you could do on this group before
// breaching the limit.
func (g *group) capacity() int {
	if g.current >= g.limit {
		return 0
	}

	return int(g.limit - g.current)
}

// notifyDecrement will result in true being sent on the given channel the next
// time decrement() is called. (And then the channel is discarded.)
func (g *group) notifyDecrement(ch chan bool) {
	g.toNotify = append(g.toNotify, ch)
}
