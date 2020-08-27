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

// retry is used to keep trying something until it works.
package retry

import "github.com/wtsi-ssg/wr/backoff"

// Operation is passed to Do() and is the code you would like to retry.
type Operation func() error

// Do will run op at least once, and then will keep retrying it unless the until
// returns a Reason to stop. The amount of time between retries is determined
// by bo.
//
// Note that bo is NOT Reset() during this function.
//
// It returns the number of retries done (which can be 0 if op only needed to
// be run once), the Reason it stopped retrying, and any error from your op on
// the last call.
func Do(op Operation, until Until, bo *backoff.Backoff) (retries int, reason Reason, err error) {
	for ok := true; ok; ok = tryAgain(bo, reason, &retries) {
		err = op()
		reason = until.ShouldStop(retries, err)
	}
	return retries, reason, err
}

// tryAgain tests reason to see if we should try again, and if so, increments
// retries and uses the backoff to sleep before returning.
func tryAgain(bo *backoff.Backoff, reason Reason, retries *int) bool {
	if reason != doNotStop {
		return false
	}
	*retries++
	bo.Sleep()
	return true
}
