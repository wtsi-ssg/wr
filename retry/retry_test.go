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

package retry

import (
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/backoff"
	bm "github.com/wtsi-ssg/wr/backoff/mock"
)

func TestRetry(t *testing.T) {
	wait := 1 * time.Millisecond
	backoff := &backoff.Backoff{Min: wait, Max: wait, Factor: 1}

	Convey("You can Retry things until they succeed", t, func() {
		count := 0
		op := func() error {
			count++
			if count == 3 {
				return nil
			}
			return errors.New("err")
		}

		sleeper := &bm.Sleeper{}
		backoff.Sleeper = sleeper

		status := Do(op, &UntilNoError{}, backoff)
		So(status.Retried, ShouldEqual, 2)
		So(status.StoppedBecause, ShouldEqual, BecauseErrorNil)
		So(status.Err, ShouldBeNil)
		msg := "after 2 retries, stopped trying because there was no error"
		So(status.String(), ShouldEqual, msg)
		So(status.Error(), ShouldEqual, msg)
		So(status.Unwrap(), ShouldBeNil)
		So(count, ShouldEqual, 3)
		So(sleeper.Elapsed(), ShouldEqual, 2*time.Millisecond)
	})

	Convey("You can Retry things until you give up", t, func() {
		count := 0
		opErr := errors.New("a problem")
		op := func() error {
			count++
			return opErr
		}

		sleeper := &bm.Sleeper{}
		backoff.Sleeper = sleeper

		status := Do(op, &UntilLimit{Max: 2}, backoff)
		So(status.Retried, ShouldEqual, 2)
		So(status.StoppedBecause, ShouldEqual, BecauseLimitReached)
		So(status.Err, ShouldEqual, opErr)
		msg := "after 2 retries, stopped trying because limit reached; err: a problem"
		So(status.String(), ShouldEqual, msg)
		So(status.Error(), ShouldEqual, msg)
		So(status.Unwrap(), ShouldEqual, opErr)
		So(count, ShouldEqual, 3)
		So(sleeper.Elapsed(), ShouldEqual, 2*time.Millisecond)
	})
}
