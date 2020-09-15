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

import (
	"fmt"

	"github.com/wtsi-ssg/wr/backoff"
)

// Operation is passed to Do() and is the code you would like to retry.
type Operation func() error

// Status is returned by Do() to explain what happened when retrying your
// Operation. It can be stringified or used as an error that wraps Err.
type Status struct {
	// Retried is the number of retries done (which can be 0 if the Operation
	// only needed to be run once).
	Retried int

	// StoppedBecause is the Reason retries were stopped.
	StoppedBecause Reason

	// Err is the last return value of the Operation.
	Err error
}

// String returns a string representation of the Status.
func (s *Status) String() string {
	errString := ""
	if s.Err != nil {
		errString = fmt.Sprintf("; err: %s", s.Err.Error())
	}

	return fmt.Sprintf("after %d retries, stopped trying because %s%s", s.Retried, s.StoppedBecause, errString)
}

// Error implements the error interface, returning the same as String().
func (s *Status) Error() string {
	return s.String()
}

// Unwrap implements the error interface, returning our wrapped error.
func (s *Status) Unwrap() error {
	return s.Err
}

// Do will run op at least once, and then will keep retrying it unless the until
// returns a Reason to stop. The amount of time between retries is determined
// by bo.
//
// Note that bo is NOT Reset() during this function.
func Do(op Operation, until Until, bo *backoff.Backoff) *Status {
	var (
		reason  Reason
		retries int
		err     error
	)

	for ok := true; ok; ok = tryAgain(bo, reason, &retries) {
		err = op()
		reason = until.ShouldStop(retries, err)
	}

	return &Status{Retried: retries, StoppedBecause: reason, Err: err}
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
