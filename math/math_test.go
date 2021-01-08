/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>
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

package math

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMathFuncs(t *testing.T) {
	Convey("nanoseconds to seconds conversion", t, func() {
		So(NanosecondsToSec(634736438394834), ShouldEqual, 634736)
		So(NanosecondsToSec(634736), ShouldEqual, 0)
		So(NanosecondsToSec(0), ShouldEqual, 0)
		So(NanosecondsToSec(1000000000), ShouldEqual, 1)
	})

	Convey("bytes to MB conversion", t, func() {
		So(BytesToMB(634736438), ShouldEqual, 605)
		So(BytesToMB(1048576), ShouldEqual, 1)
		So(BytesToMB(0), ShouldEqual, 0)
	})

	Convey("Test to check toFixed to round down to 3 places", t, func() {
		So(toFixed(123.123456), ShouldEqual, 123.123)
		So(toFixed(234.144444), ShouldEqual, 234.144)
	})

	Convey("Test to check FloatLessThan", t, func() {
		So(FloatLessThan(123.123456, 123.1245678), ShouldEqual, true)
		So(FloatLessThan(234.144444, 123.123456), ShouldEqual, false)
	})

	Convey("Test to check FloatSubtract", t, func() {
		So(FloatSubtract(234.144444, 123.123456), ShouldEqual, 111.021)
		So(FloatSubtract(123.123456, 234.144444), ShouldEqual, -111.021)
	})

	Convey("Test to check FloatAdd", t, func() {
		So(FloatAdd(234.144444, 123.123456), ShouldEqual, 357.267)
	})
}
