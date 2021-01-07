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
	"path/filepath"
	"strings"
)

// this file has general utility functions

// nanosecondsToSec converts nanoseconds to sec for cpu stats.
func nanosecondsToSec(tm uint64) int {
	var divisor uint64 = 1000000000

	return int(tm / divisor)
}

// bytesToMB converts bytes to MB for memory stats.
func bytesToMB(bt uint64) int {
	var divisor uint64 = 1024

	return int(bt / divisor / divisor)
}

// relativeToAbsolutePath returns the absolute path of a file given it's relative path
// and its directory name.
func relativeToAbsolutePath(path string, dir string) string {
	absPath := path
	if !strings.HasPrefix(absPath, "/") {
		absPath = filepath.Join(dir, absPath)
	}

	return absPath
}
