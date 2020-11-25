/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>. Ashwini Chhipa <ac55@sanger.ac.uk>
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

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func BenchmarkLimiterIncDec(b *testing.B) {
	limits := make(map[string]int)
	limits["l1"] = 5
	limits["l2"] = 6
	cb := func(name string) int {
		if limit, exists := limits[name]; exists {
			return limit
		}

		return -1
	}
	both := []string{"l1", "l2"}
	first := []string{"l1"}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l := New(cb)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Increment(both)
		l.Decrement(both)
		l.Decrement(both)
		l.Decrement(both)
		l.Decrement(both)
		l.Decrement(both)
		l.Decrement(both)

		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Increment(first)
		l.Decrement(first)
		l.Decrement(first)
		l.Decrement(first)
		l.Decrement(first)
		l.Decrement(first)
		l.Decrement(first)
	}
}
func BenchmarkLimiterCapacity(b *testing.B) {
	limits := make(map[string]int)
	limits["l1"] = 5
	limits["l2"] = 6
	cb := func(name string) int {
		if limit, exists := limits[name]; exists {
			return limit
		}

		return -1
	}
	both := []string{"l1", "l2"}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l := New(cb)

		for {
			l.Increment(both)

			cap := l.GetRemainingCapacity(both)
			if cap == 0 {
				break
			}
		}

		for {
			l.Decrement(both)

			cap := l.GetRemainingCapacity(both)
			if cap == 5 {
				break
			}
		}
	}
}

func TestLimiter(t *testing.T) {
	Convey("You can make a new Limiter with a limit defining callback", t, func() {
		limits := make(map[string]int)
		limits["l1"] = 3
		limits["l2"] = 2
		limits["l4"] = 100
		limits["l5"] = 200
		cb := func(name string) int {
			if limit, exists := limits[name]; exists {
				return limit
			}

			return -1
		}

		l := New(cb)
		So(l, ShouldNotBeNil)

		Convey("Increment and Decrement work as expected", func() {
			So(l.Increment([]string{"l1", "l2"}), ShouldBeTrue)
			l.Decrement([]string{"l1", "l2"})

			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeFalse)
			So(l.Increment([]string{"l1", "l2"}), ShouldBeFalse)
			l.Decrement([]string{"l1", "l2"})
			So(l.Increment([]string{"l1", "l2"}), ShouldBeTrue)
			l.Decrement([]string{"l2"})
			So(l.Increment([]string{"l1", "l2"}), ShouldBeTrue)

			So(l.Increment([]string{"l3"}), ShouldBeTrue)
			l.Decrement([]string{"l3"})
		})

		Convey("You can change limits with SetLimit(), and Decrement() forgets about unused groups", func() {
			groups := []string{"l1", "l2"}
			two := []string{"l2"}
			So(l.GetLowestLimit(groups), ShouldEqual, 2)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 2)
			So(l.Increment(two), ShouldBeTrue)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 1)
			So(l.Increment(two), ShouldBeTrue)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 0)
			So(l.Increment(two), ShouldBeFalse)
			l.SetLimit("l2", 3)
			So(l.GetLowestLimit(groups), ShouldEqual, 3)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 1)
			So(l.Increment(two), ShouldBeTrue)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 0)
			So(l.Increment(two), ShouldBeFalse)
			l.Decrement(two)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 1)
			l.Decrement(two)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 2)
			l.Decrement(two)
			// at this point l2 should have been forgotten about, which means
			// we forgot we set the limit to 3
			So(l.GetRemainingCapacity(groups), ShouldEqual, 2)
			l.Decrement(two) // doesn't panic or something
			So(l.GetLowestLimit(groups), ShouldEqual, 2)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 2)
			So(l.Increment(two), ShouldBeTrue)
			So(l.Increment(two), ShouldBeTrue)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 0)
			So(l.Increment(two), ShouldBeFalse)
			l.Decrement(two)
			l.Decrement(two)
			limits["l2"] = 3
			So(l.GetRemainingCapacity(groups), ShouldEqual, 3)
			So(l.Increment(two), ShouldBeTrue)
			So(l.GetLowestLimit(groups), ShouldEqual, 3)
			So(l.GetRemainingCapacity(groups), ShouldEqual, 2)
			So(l.Increment(two), ShouldBeTrue)
			So(l.Increment(two), ShouldBeTrue)
			So(l.Increment(two), ShouldBeFalse)

			testGroup := l.vivifyGroup(groups[0])
			So(getLowestCapacity(testGroup, -1), ShouldEqual, 3)
		})

		Convey("You can have limits of 0 and also RemoveLimit()s", func() {
			l.SetLimit("l2", 0)
			So(l.Increment([]string{"l2"}), ShouldBeFalse)

			limits["l2"] = 0
			l.RemoveLimit("l2")
			So(l.Increment([]string{"l2"}), ShouldBeFalse)
			So(l.GetLimit("l2"), ShouldEqual, 0)

			limits["l2"] = -1
			So(l.Increment([]string{"l2"}), ShouldBeFalse)
			So(l.GetLimit("l2"), ShouldEqual, 0)

			l.RemoveLimit("l2")
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.Increment([]string{"l2"}), ShouldBeTrue)
			So(l.GetLimit("l2"), ShouldEqual, -1)
		})

		Convey("Concurrent SetLimit(), Increment() and Decrement() work", func() {
			var incs uint64
			var fails uint64
			var wg sync.WaitGroup
			for i := 0; i < 200; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					groups := []string{"l4", "l5"}
					if i%2 == 0 {
						groups = []string{"l5", "l4"}
					}

					if l.Increment(groups) {
						atomic.AddUint64(&incs, 1)
						time.Sleep(100 * time.Millisecond)
						l.Decrement(groups)
					} else {
						atomic.AddUint64(&fails, 1)
						if atomic.LoadUint64(&fails) == 50 {
							l.SetLimit("l4", 125)
						}
					}
				}(i)
			}
			wg.Wait()

			So(atomic.LoadUint64(&incs), ShouldEqual, 125)
			So(atomic.LoadUint64(&fails), ShouldEqual, 75)
		})

		Convey("Concurrent Increment()s at the limit work with wait times", func() {
			groups := []string{"l1", "l2"}
			So(l.Increment(groups), ShouldBeTrue)
			So(l.Increment(groups), ShouldBeTrue)
			So(l.Increment(groups), ShouldBeFalse)
			start := time.Now()

			go func() {
				l.Decrement(groups)
				l.Decrement(groups)
				<-time.After(50 * time.Millisecond)
				l.Decrement(groups)
			}()

			go func() {
				<-time.After(60 * time.Millisecond)
				// (decrementing the higher capacity group doesn't make an
				// increment of the lower capacity group work)
				l.Decrement([]string{"l1"})
			}()

			var quickIncs uint64
			var slowIncs uint64
			var fails uint64
			wait := 125 * time.Millisecond
			var wg sync.WaitGroup
			for i := 0; i < 4; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					if l.Increment(groups, wait) {
						if time.Since(start) < 35*time.Millisecond {
							atomic.AddUint64(&quickIncs, 1)
						} else {
							atomic.AddUint64(&slowIncs, 1)
						}
					} else {
						if time.Since(start) > 100*time.Millisecond {
							atomic.AddUint64(&fails, 1)
						}
					}
				}()
			}
			wg.Wait()

			So(atomic.LoadUint64(&quickIncs), ShouldEqual, 2)
			So(atomic.LoadUint64(&slowIncs), ShouldEqual, 1)
			So(atomic.LoadUint64(&fails), ShouldEqual, 1)
		})
	})
}
