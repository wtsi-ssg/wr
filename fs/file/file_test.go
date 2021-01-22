/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 ******************************************************************************/

package file

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// fileMode is the mode of the temp file created for testing.
const fileMode os.FileMode = 0600

func TestFile(t *testing.T) {
	Convey("Get the first line of a file", t, func() {
		tempDir, err := ioutil.TempDir("", "temp_filepath")
		if err != nil {
			log.Fatal(err)
		}

		defer os.RemoveAll(tempDir)

		Convey("when the file exists", func() {
			tempFile := filepath.Join(tempDir, "tempFile.txt")
			err = ioutil.WriteFile(tempFile, []byte("id1"), fileMode)
			So(err, ShouldBeNil)

			id, err := GetFirstLine(tempFile)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "id1")

			tempFile1 := filepath.Join(tempDir, "tempFile1.txt")
			err = ioutil.WriteFile(tempFile1, []byte("id1\n"), fileMode)
			So(err, ShouldBeNil)

			id, err = GetFirstLine(tempFile1)
			So(err, ShouldBeNil)
			So(id, ShouldNotEqual, "id1\n")
			So(id, ShouldEqual, "id1")
		})

		Convey("when the file doesn't exist", func() {
			tempNonExistingFile := filepath.Join(tempDir, "tempNonExisting.txt")
			noID, err := GetFirstLine(tempNonExistingFile)
			So(err, ShouldNotBeNil)
			So(noID, ShouldBeEmpty)
		})
	})
}
