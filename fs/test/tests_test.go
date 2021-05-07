/*******************************************************************************
 * Copyright (c) 2021 Genome Research Ltd.
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

package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/clog"
)

func TestTestFuncs(t *testing.T) {
	Convey("FilePathInTempDir returns a non-existent path in an existing tmp dir", t, func() {
		basename := "foo"
		path := FilePathInTempDir(t, basename)
		fmt.Printf("got path %s\n", path)
		So(path, ShouldStartWith, os.TempDir())
		So(path, ShouldEndWith, basename)
		_, err := os.Open(filepath.Dir(path))
		So(err, ShouldBeNil)
		_, err = os.Open(path)
		So(err, ShouldNotBeNil)
	})

	Convey("We can mock stdin", t, func() {
		origStdin, stdinWriter, err := mockStdinRW("test")
		So(origStdin, ShouldNotBeNil)
		So(stdinWriter, ShouldNotBeNil)
		So(err, ShouldBeNil)

		os.Stdin = origStdin
		stdinWriter.Close()
	})

	Convey("We can mock stderr", t, func() {
		origStderr, stderrReader, outCh, err := mockStderrRW()
		So(origStderr, ShouldNotBeNil)
		So(stderrReader, ShouldNotBeNil)
		So(outCh, ShouldNotBeNil)
		So(err, ShouldBeNil)

		os.Stderr = origStderr
		stderrReader.Close()
	})

	Convey("We can mock stdin and stderr", t, func() {
		ctx := context.Background()

		mockedStdinerr, err := NewMockStdInErr("test")
		So(mockedStdinerr, ShouldNotBeNil)
		So(err, ShouldBeNil)

		Convey("and read the data written to stderr", func() {
			clog.ToDefaultAtLevel("debug")
			clog.Debug(ctx, "msg", "foo", 1)
			stderr, err := mockedStdinerr.ReadAndRestoreStderr()
			So(err, ShouldBeNil)
			So(stderr, ShouldContainSubstring, "foo=1")
			So(mockedStdinerr.stderrReader, ShouldBeNil)
			So(mockedStdinerr.origStderr, ShouldNotBeNil)

			Convey("but not when mock error reader is already closed", func() {
				_, err = mockedStdinerr.ReadAndRestoreStderr()
				So(err, ShouldNotBeNil)
			})
		})

		Convey("and restore stdin to default", func() {
			mockedStdinerr.RestoreStdin()
			So(mockedStdinerr.stdinWriter, ShouldBeNil)
			So(mockedStdinerr.origStdin, ShouldNotBeNil)
		})
	})
}
