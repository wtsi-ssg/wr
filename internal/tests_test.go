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

package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestFuncs(t *testing.T) {
	Convey("FilePathInTempDir returns a non-existent path in an existing tmp dir", t, func() {
		basename := "foo"
		path := FilePathInTempDir(t, basename)
		fmt.Printf("got path %s\n", path)
		So(path, ShouldStartWith, "/tmp")
		So(path, ShouldEndWith, basename)
		_, err := os.Open(filepath.Dir(path))
		So(err, ShouldBeNil)
		_, err = os.Open(path)
		So(err, ShouldNotBeNil)

		Convey("FileAsString returns file content", func() {
			content := "foo\nbar\n"
			err = ioutil.WriteFile(path, []byte(content), 0600)
			So(err, ShouldBeNil)
			read := FileAsString(path)
			So(read, ShouldEqual, content)
		})
	})
}
