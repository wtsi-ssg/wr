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

	. "github.com/smartystreets/goconvey/convey"
)

func TestContext(t *testing.T) {
	background := context.Background()

	checkValIsString := func(val interface{}) string {
		strVal, isString := val.(string)
		So(isString, ShouldBeTrue)

		return strVal
	}

	checkValIsUniqueID := func(val interface{}) {
		id := checkValIsString(val)
		So(len(id), ShouldEqual, uniqueIDLength)
	}

	Convey("ContextForRetries returns a context with a retryset and retryactivity", t, func() {
		activity := "doing foo"
		ctx := ContextForRetries(background, activity)
		val := ctx.Value(retrySetKey)
		checkValIsUniqueID(val)

		val = ctx.Value(retryActivityKey)
		So(checkValIsString(val), ShouldEqual, activity)
	})

	Convey("ContextWithRetryNum returns a context with a retrynum", t, func() {
		retrynum := 3
		ctx := ContextWithRetryNum(background, retrynum)
		val := ctx.Value(retryNumKey)
		num, isInt := val.(int)
		So(isInt, ShouldBeTrue)
		So(num, ShouldEqual, retrynum)
	})
}
