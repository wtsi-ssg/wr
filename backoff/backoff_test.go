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

package backoff

import (
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/backoff/mock"
)

func TestBackoff(t *testing.T) {
	base := time.Now()

	Convey("Given a Backoff", t, func() {
		sleeper := &mock.Sleeper{}
		b := &Backoff{
			Min:     1 * time.Millisecond,
			Max:     10 * time.Millisecond,
			Factor:  2,
			Sleeper: sleeper,
		}

		Convey("It Sleep()s for Min", func() {
			b.Sleep()
			So(sleeper.Invoked(), ShouldEqual, 1)
			So(sleeper.Elapsed(), ShouldEqual, 1*time.Millisecond)

			Convey("The next call Sleep()s for Min*Factor, with jitter", func() {
				b.Sleep()
				So(sleeper.Invoked(), ShouldEqual, 2)
				So(base.Add(sleeper.Elapsed()), ShouldHappenOnOrBetween, base.Add(1*time.Millisecond), base.Add(3*time.Millisecond))

				Convey("Subsequent calls keep increasing sleep by Factor, with jitter", func() {
					b.Sleep()
					So(sleeper.Invoked(), ShouldEqual, 3)
					So(base.Add(sleeper.Elapsed()), ShouldHappenOnOrBetween, base.Add(3*time.Millisecond), base.Add(7*time.Millisecond))

					b.Sleep()
					So(sleeper.Invoked(), ShouldEqual, 4)
					So(base.Add(sleeper.Elapsed()), ShouldHappenOnOrBetween, base.Add(7*time.Millisecond), base.Add(15*time.Millisecond))

					Convey("But not above Max", func() {
						b.Sleep()
						So(sleeper.Invoked(), ShouldEqual, 5)
						elapsed := sleeper.Elapsed()
						So(base.Add(elapsed), ShouldHappenOnOrBetween, base.Add(15*time.Millisecond), base.Add(25*time.Millisecond))

						Convey("And it can be reset back to Min", func() {
							b.Reset()
							b.Sleep()
							So(sleeper.Invoked(), ShouldEqual, 6)
							So(sleeper.Elapsed(), ShouldEqual, elapsed+1*time.Millisecond)
						})
					})
				})
			})
		})
	})

	Convey("If you create a Backoff with a Factor less than 1, it always Sleep()s for Min", t, func() {
		sleeper := &mock.Sleeper{}
		b := &Backoff{
			Min:     1 * time.Millisecond,
			Max:     10 * time.Millisecond,
			Factor:  0.5,
			Sleeper: sleeper,
		}

		b.Sleep()
		So(sleeper.Invoked(), ShouldEqual, 1)
		So(sleeper.Elapsed(), ShouldEqual, 1*time.Millisecond)

		b.Sleep()
		So(sleeper.Invoked(), ShouldEqual, 2)
		So(sleeper.Elapsed(), ShouldEqual, 2*time.Millisecond)
	})

	Convey("A Backoff treats Max less than Min as Min", t, func() {
		sleeper := &mock.Sleeper{}
		b := &Backoff{
			Min:     10 * time.Millisecond,
			Factor:  1,
			Sleeper: sleeper,
		}

		b.Sleep()
		So(sleeper.Invoked(), ShouldEqual, 1)
		So(sleeper.Elapsed(), ShouldEqual, 10*time.Millisecond)

		b.Sleep()
		So(sleeper.Invoked(), ShouldEqual, 2)
		So(sleeper.Elapsed(), ShouldEqual, 20*time.Millisecond)
	})

	Convey("A Backoff can be used concurrently", t, func() {
		sleeper := &mock.Sleeper{}
		b := &Backoff{
			Min:     1 * time.Millisecond,
			Max:     10 * time.Millisecond,
			Factor:  1,
			Sleeper: sleeper,
		}

		wg := &sync.WaitGroup{}

		sleep := func() {
			b.Sleep()
			wg.Done()
		}

		c := 4
		wg.Add(c)
		for i := 1; i <= c; i++ {
			go sleep()
		}
		wg.Add(1)
		go func() {
			b.Reset()
			b.Sleep()
			wg.Done()
		}()
		wg.Wait()

		So(sleeper.Invoked(), ShouldEqual, 5)
		So(base.Add(sleeper.Elapsed()), ShouldHappenOnOrBetween, base.Add(4*time.Millisecond), base.Add(5*time.Millisecond))
	})
}
