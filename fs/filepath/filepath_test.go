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

package filepath

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

func TestPath(t *testing.T) {
	Convey("Get the absolute path of a file given its relative path and directory name", t, func() {
		So(RelToAbsPath("testing1.txt", "/home_directory"), ShouldEqual, "/home_directory/testing1.txt")
		So(RelToAbsPath("/testing1.txt", "/home_directory"), ShouldEqual, "/testing1.txt")
		So(RelToAbsPath("testing1.txt", "/"), ShouldEqual, "/testing1.txt")
		So(RelToAbsPath("testing1.txt", "."), ShouldEqual, "testing1.txt")
		So(RelToAbsPath("testing1.txt", ""), ShouldEqual, "testing1.txt")
	})

	Convey("Read the contents of a file", t, func() {
		tempDir, err := ioutil.TempDir("", "temp_filepath")
		if err != nil {
			log.Fatal(err)
		}

		Convey("when the file exists", func() {
			tempFile := filepath.Join(tempDir, "tempFile.txt")
			err = ioutil.WriteFile(tempFile, []byte("test"), fileMode)
			So(err, ShouldBeNil)

			content, err := ReadFile(tempFile)
			So(err, ShouldBeNil)
			So(content, ShouldNotBeNil)
		})

		Convey("when the file doesn't exist", func() {
			tempNonExistingFile := filepath.Join(tempDir, "tempNonExisting.txt")
			noContent, err := ReadFile(tempNonExistingFile)
			So(err, ShouldNotBeNil)
			So(noContent, ShouldBeNil)
		})
	})
}
