/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Authors: Ashwini Chhipa <ac55@sanger.ac.uk>
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

package internal

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUtilsFuncs(t *testing.T) {
	Convey("nanoseconds to seconds conversion", t, func() {
		So(nanosecondsToSec(634736438394834), ShouldEqual, 634736)
		So(nanosecondsToSec(634736), ShouldEqual, 0)
		So(nanosecondsToSec(0), ShouldEqual, 0)
		So(nanosecondsToSec(1000000000), ShouldEqual, 1)
	})

	Convey("bytes to MB conversion", t, func() {
		So(bytesToMB(634736438), ShouldEqual, 605)
		So(bytesToMB(1048576), ShouldEqual, 1)
		So(bytesToMB(0), ShouldEqual, 0)
	})

	Convey("Get the absolute path of a file given its relative path and directory name", t, func() {
		So(relativeToAbsolutePath("testing1.txt", "/home_directory"), ShouldEqual, "/home_directory/testing1.txt")
		So(relativeToAbsolutePath("/testing1.txt", "/home_directory"), ShouldEqual, "/testing1.txt")
		So(relativeToAbsolutePath("testing1.txt", "/"), ShouldEqual, "/testing1.txt")
		So(relativeToAbsolutePath("testing1.txt", "."), ShouldEqual, "testing1.txt")
		So(relativeToAbsolutePath("testing1.txt", ""), ShouldEqual, "testing1.txt")
	})
}
