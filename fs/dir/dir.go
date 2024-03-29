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

package dir

// this file implements utility routines related to directories.

import (
	"context"
	"os"

	"github.com/wtsi-ssg/wr/clog"
)

// GetPWD returns the present working directory and exits on error.
func GetPWD(ctx context.Context) string {
	pwd, err := os.Getwd()
	if err != nil {
		clog.Fatal(ctx, err.Error())
	}

	return pwd
}

// GetHome returns the home directory of current user and exits on error.
func GetHome(ctx context.Context) string {
	home, herr := os.UserHomeDir()

	if herr != nil || home == "" {
		clog.Fatal(ctx, "could not find home dir", "err", herr)
	}

	return home
}
