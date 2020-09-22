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

package clog

import (
	"context"
	"testing"

	"github.com/inconshreveable/log15"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/internal"
)

func TestLogger(t *testing.T) {
	background := context.Background()

	Convey("lvlFromString returns appropriate levels", t, func() {
		So(lvlFromString("debug"), ShouldEqual, log15.LvlDebug)
		So(lvlFromString("info"), ShouldEqual, log15.LvlInfo)
		So(lvlFromString("warn"), ShouldEqual, log15.LvlWarn)
		So(lvlFromString("error"), ShouldEqual, log15.LvlError)
		So(lvlFromString("crit"), ShouldEqual, log15.LvlCrit)
		So(lvlFromString("foo"), ShouldEqual, log15.LvlDebug)
	})

	Convey("RetrySet context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		activity := "doing foo"
		ctx := ContextForRetries(background, activity)
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "retryset=")
		So(buff.String(), ShouldContainSubstring, "retryactivity=\""+activity)
	})

	Convey("RetryNum context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithRetryNum(background, 3)
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "retrynum=3")
	})

	Convey("With logging set to a buffer at warn level, and some context", t, func() {
		buff := ToBufferAtLevel("warn")
		retryNum := 3
		ctx := ContextWithRetryNum(background, retryNum)
		retryLogMsg := "retrynum=3"

		hasMsgAndFoo := func(lvl, lmsg string) {
			So(lmsg, ShouldContainSubstring, "lvl="+lvl)
			So(lmsg, ShouldContainSubstring, "msg=msg")
			So(lmsg, ShouldContainSubstring, "foo=1")
		}

		Convey("Debug does nothing at level warn", func() {
			Debug(ctx, "msg", "foo", 1)
			So(buff.String(), ShouldBeBlank)

			Convey("But works at level debug", func() {
				buff = ToBufferAtLevel("debug")

				Debug(ctx, "msg", "foo", 1)
				lmsg := buff.String()
				hasMsgAndFoo("dbug", lmsg)
				So(lmsg, ShouldContainSubstring, "caller=clog")
				So(lmsg, ShouldContainSubstring, retryLogMsg)
				buff.Reset()

				Convey("And then stops working when you go back to default", func() {
					ToDefault()
					Debug(ctx, "msg", "foo", 1)
					So(buff.String(), ShouldBeBlank)
				})

				Convey("And works without context", func() {
					Debug(context.Background(), "msg", "foo", 1)
					lmsg := buff.String()
					hasMsgAndFoo("dbug", lmsg)
					So(lmsg, ShouldNotContainSubstring, retryLogMsg)
				})
			})
		})

		Convey("Info does nothing at level warn", func() {
			Info(ctx, "msg", "foo", 1)
			So(buff.String(), ShouldBeBlank)

			Convey("But works at level debug and info", func() {
				buff = ToBufferAtLevel("debug")

				Info(ctx, "msg", "foo", 1)
				lmsg := buff.String()
				hasMsgAndFoo("info", lmsg)
				So(lmsg, ShouldNotContainSubstring, "caller=clog")
				So(lmsg, ShouldContainSubstring, retryLogMsg)
				buff.Reset()

				buff = ToBufferAtLevel("info")
				Info(ctx, "msg", "foo", 1)
				lmsg = buff.String()
				hasMsgAndFoo("info", lmsg)
				So(lmsg, ShouldContainSubstring, retryLogMsg)
				buff.Reset()
			})
		})

		checkMethod := func(method func(context.Context, string, ...interface{}), lvl1, lvl2 string) {
			method(ctx, "msg", "foo", 1)
			lmsg := buff.String()
			hasMsgAndFoo(lvl1, lmsg)
			So(lmsg, ShouldContainSubstring, "caller=clog")
			So(lmsg, ShouldContainSubstring, retryLogMsg)

			Convey("But not at a higher level", func() {
				buff = ToBufferAtLevel(lvl2)
				So(buff.String(), ShouldBeBlank)
			})
		}

		Convey("Warn works", func() {
			checkMethod(Warn, "warn", "error")
		})

		Convey("Error works", func() {
			checkMethod(Error, "eror", "crit")
		})

		Convey("Crit always works and has a stack trace", func() {
			Crit(ctx, "msg", "foo", 1)
			lmsg := buff.String()
			hasMsgAndFoo("crit", lmsg)
			So(lmsg, ShouldNotContainSubstring, "caller=clog")
			So(lmsg, ShouldContainSubstring, "stack=")
			So(lmsg, ShouldContainSubstring, retryLogMsg)
		})
	})

	Convey("You can log to a file", t, func() {
		logPath, toDefer := internal.FilePathInTempDir("clog.log")
		defer toDefer()

		err := ToFileAtLevel(logPath, "debug")
		So(err, ShouldBeNil)
		Debug(background, "msg")

		So(internal.FileAsString(logPath), ShouldContainSubstring, "msg=msg")

		Convey("And append to a file", func() {
			err = ToFileAtLevel(logPath, "debug")
			So(err, ShouldBeNil)

			Debug(background, "foo")

			logs := internal.FileAsString(logPath)
			So(logs, ShouldContainSubstring, "msg=msg")
			So(logs, ShouldContainSubstring, "msg=foo")
		})
	})

	Convey("You can't log to a file given a bad path", t, func() {
		err := ToFileAtLevel("!/*&^%$", "debug")
		So(err, ShouldNotBeNil)
	})
}
