/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>.
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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGroup(t *testing.T) {
	Convey("Test to check the group", t, func() {
		limit := 2
		group := newGroup("g1", uint(limit))
		So(group, ShouldNotBeNil)
		So(group.name, ShouldEqual, "g1")
		So(group.limit, ShouldEqual, 2)

		Convey("Test to check setLimit", func() {
			limit := 4
			group.setLimit(uint(limit))
			So(group.limit, ShouldEqual, 4)

			Convey("Test to check canIncrement", func() {
				So(group.canIncrement(), ShouldEqual, true)
				group.current = 4
				So(group.canIncrement(), ShouldEqual, false)

				Convey("Test to check increment", func() {
					group.increment()
					So(group.current, ShouldEqual, 5)
				})

				Convey("Test to check capacity", func() {
					So(group.capacity(), ShouldEqual, 0)
					group.setLimit(uint(6))
					So(group.capacity(), ShouldEqual, 2)
				})

				Convey("Test to check notifyDecrement", func() {
					ch := make(chan bool, 1)
					So(group.toNotify, ShouldBeEmpty)
					group.notifyDecrement(ch)
					So(group.toNotify, ShouldNotBeEmpty)

					Convey("Test to check the decrement", func() {
						So(group.decrement(), ShouldEqual, false)
						So(group.decrement(), ShouldEqual, false)
						So(group.decrement(), ShouldEqual, false)
						So(group.decrement(), ShouldEqual, true)
						So(group.decrement(), ShouldEqual, true)
					})
				})
			})
		})
	})
}
