/*******************************************************************************
 * Copyright (c) 2020, 2021 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>, <ac55@sanger.ac.uk>
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
	"bytes"
	"context"
	"log/syslog"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hpcloud/tail"
	"github.com/inconshreveable/log15"
	. "github.com/smartystreets/goconvey/convey"
	fl "github.com/wtsi-ssg/wr/fs/file"
	ft "github.com/wtsi-ssg/wr/fs/test"
)

func TestLogger(t *testing.T) {
	background := context.Background()

	Convey("GetHandler returns a log15 handler", t, func() {
		handler := GetHandler()
		So(handler, ShouldNotBeNil)
		So(handler, ShouldHaveSameTypeAs, log15.FuncHandler(func(r *log15.Record) error { return nil }))
	})

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

	Convey("JobKey context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithJobKey(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "jobkey=bar")
	})

	Convey("ServerID context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithServerID(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "serverid=bar")
	})

	Convey("CloudType context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithCloudType(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "cloudtype=bar")
	})

	Convey("SchedulerType context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithSchedulerType(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "schedulertype=bar")
	})

	Convey("CallValue context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithCallValue(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "callvalue=bar")
	})

	Convey("ServerFlavor context gets logged", t, func() {
		buff := ToBufferAtLevel("debug")
		ctx := ContextWithServerFlavor(background, "bar")
		Debug(ctx, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "serverflavor=bar")
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

			Convey("But works using ToDefaultAtLevel() set to debug", func() {
				fse, err := ft.NewMockStdErr()
				So(err, ShouldBeNil)
				ToDefaultAtLevel("debug")
				Debug(ctx, "msg", "foo", 1)
				stderr, err := fse.GetAndRestoreStdErr()
				So(err, ShouldBeNil)
				So(stderr, ShouldContainSubstring, "foo=1")
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

		Convey("Fatal works and has a stack trace", func() {
			os.Setenv("WR_FATAL_EXIT_TEST", "1")
			defer os.Unsetenv("WR_FATAL_EXIT_TEST")
			Fatal(ctx, "msg", "foo", 1)
			lmsg := buff.String()
			hasMsgAndFoo("crit", lmsg)
			So(lmsg, ShouldContainSubstring, "fatal=true")
			So(lmsg, ShouldNotContainSubstring, "caller=clog")
			So(lmsg, ShouldContainSubstring, "stack=")
			So(lmsg, ShouldContainSubstring, retryLogMsg)
		})
	})

	Convey("You can log to a file", t, func() {
		logPath := ft.FilePathInTempDir(t, "clog.log")

		err := ToFileAtLevel(logPath, "debug")
		So(err, ShouldBeNil)
		Debug(background, "msg")

		strContent, err := fl.ToString(logPath)
		So(err, ShouldBeNil)
		So(strContent, ShouldContainSubstring, "msg=msg")

		Convey("And append to a file", func() {
			err = ToFileAtLevel(logPath, "debug")
			So(err, ShouldBeNil)

			Debug(background, "foo")

			logs, err := fl.ToString(logPath)
			So(err, ShouldBeNil)
			So(logs, ShouldContainSubstring, "msg=msg")
			So(logs, ShouldContainSubstring, "msg=foo")
		})
	})

	Convey("You can't log to a file given a bad path", t, func() {
		err := ToFileAtLevel("!/*&^%$", "debug")
		So(err, ShouldNotBeNil)
	})

	Convey("You can log to an abitrary handler at desired level", t, func() {
		buff := new(bytes.Buffer)
		handler := log15.StreamHandler(buff, log15.LogfmtFormat())

		ToHandlerAtLevel(handler, "warn")
		Warn(background, "msg", "foo", 1)
		So(buff.String(), ShouldContainSubstring, "foo=1")

		trySyslogTest(t)
	})

	Convey("CreateFileHandler can be used to create a file handler", t, func() {
		logPath := ft.FilePathInTempDir(t, "clog.log")
		fh, err := CreateFileHandlerAtLevel(logPath, "warn")
		So(err, ShouldBeNil)
		So(fh, ShouldNotBeNil)

		Convey("Unless the path is invalid", func() {
			fh, err := CreateFileHandlerAtLevel("", "warn")
			So(fh, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})

	Convey("You can add a handler to log to multiple places at once", t, func() {
		buff := ToBufferAtLevel("warn")
		logPath := ft.FilePathInTempDir(t, "clog.log")
		fh, err := CreateFileHandlerAtLevel(logPath, "warn")
		So(err, ShouldBeNil)
		So(fh, ShouldNotBeNil)
		AddHandler(fh)

		Warn(context.Background(), "msg", "warn", 1)
		Debug(context.Background(), "msg", "debug", 1)

		strContent := buff.String()
		So(strContent, ShouldContainSubstring, "caller=clog.go")
		So(strContent, ShouldContainSubstring, "warn=1")
		So(strContent, ShouldNotContainSubstring, "debug=1")
		buff.Reset()

		strContent, err = fl.ToString(logPath)
		So(err, ShouldBeNil)
		So(strContent, ShouldContainSubstring, "caller=clog.go")
		So(strContent, ShouldContainSubstring, "warn=1")
		So(strContent, ShouldNotContainSubstring, "debug=1")
	})
}

// trySyslogTest does syslog tests if we can access a syslog path.
func trySyslogTest(t *testing.T) {
	t.Helper()

	syslogpath := getSyslogPath()
	if syslogpath == "" {
		return
	}

	Convey("Including a syslog handler", func() {
		handler, err := log15.SyslogHandler(syslog.LOG_USER,
			"wrrunner", log15.LogfmtFormat())
		So(err, ShouldBeNil)

		logCh := make(chan string)
		startSyslogTail(syslogpath, logCh)

		ToHandlerAtLevel(handler, "warn")
		Warn(context.Background(), "msg", "foo", 1)

		select {
		case tailedLog := <-logCh:
			So(tailedLog, ShouldContainSubstring, "lvl=warn")
			So(tailedLog, ShouldContainSubstring, "foo=1")
		case <-time.After(10 * time.Second):
			t.Error("syslog test timed out")
		}
	})
}

// getSyslogPath tries to find the path to the syslog file. If it can't be found
// or isn't readable, returns blank.
func getSyslogPath() string {
	syslogpath := "/var/log/syslog"

	if runtime.GOOS == "darwin" {
		syslogpath = "/var/log/system.log"
	} else if runtime.GOOS == "windows" {
		return ""
	}

	f, err := os.Open(syslogpath)
	if err != nil {
		return ""
	}

	err = f.Close()
	if err != nil {
		return ""
	}

	return syslogpath
}

// startSyslogTail starts reading the end of syslogpath and will send the first
// log line that contains "wrrunner" to the given logCh and then stop.
func startSyslogTail(syslogpath string, logCh chan string) {
	tailer, err := tail.TailFile(syslogpath, tail.Config{
		Follow: true,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: os.SEEK_END,
		},
		Poll:   true,
		Logger: tail.DiscardingLogger,
	})
	So(err, ShouldBeNil)

	started := make(chan bool)

	go func() {
		started <- true

		for line := range tailer.Lines {
			if strings.Contains(line.Text, "wrrunner") {
				logCh <- line.Text

				break
			}
		}
	}()

	<-started
	<-time.After(50 * time.Millisecond)
}
