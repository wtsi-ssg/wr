/*******************************************************************************
 * Copyright (c) 2021 Genome Research Ltd.
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

// this file implements utility routines related to files.

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/smartystreets/goconvey/convey"
	fp "github.com/wtsi-ssg/wr/fs/filepath"
)

// PathReadError records an path read error.
type PathReadError struct {
	path string
	Err  error
}

// Error returns an error related to path could not be read.
func (p *PathReadError) Error() string {
	return fmt.Sprintf("path [%s] could not be read: %s", p.path, p.Err)
}

// GetFirstLine reads the content of a file given its absolute path and returns
// the first line excluding trailing newline.
func GetFirstLine(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	firstLine := strings.TrimSuffix(string(content), "\n")

	return firstLine, nil
}

// ToString returns the contents of a file as a string.
func ToString(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	convey.So(err, convey.ShouldBeNil)

	return string(content)
}

// PathToContent takes the path to a file and returns its contents as a string.
// If path begins with a tilda, TildaToHome() is used to first convert the path
// to an absolute path, in order to find the file.
func PathToContent(path string) (string, error) {
	if path == "" {
		return "", &PathReadError{"", nil}
	}

	absPath := fp.TildaToHome(path)

	contents, err := ioutil.ReadFile(absPath)
	if err != nil {
		return "", &PathReadError{absPath, err}
	}

	return string(contents), nil
}
