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

package math

// this file has general utility functions

import "math"

const (
	floatFixedPrecision int    = 3
	nanoDivisor         uint64 = 1000000000
	megabyteDivisor     uint64 = 1024
)

// nanosecondsToSec converts nanoseconds to sec for cpu stats.
func NanosecondsToSec(tm uint64) int {
	return int(tm / nanoDivisor)
}

// bytesToMB converts bytes to MB for memory stats.
func BytesToMB(bt uint64) int {
	return int(bt / megabyteDivisor / megabyteDivisor)
}

// FloatLessThan tells you if a < b, treating both float64s rounded to 3 decimal
// places of precision.
func FloatLessThan(a, b float64) bool {
	return toFixed(a) < toFixed(b)
}

// toFixed rounds down a float64 to 3 decimal places.
func toFixed(num float64) float64 {
	baseExpOutput := math.Pow10(floatFixedPrecision)

	return math.Round(num*baseExpOutput) / baseExpOutput
}

// FloatSubtract does a - b, treating both float64s rounded to 3 decimal places
// of precision.
func FloatSubtract(a, b float64) float64 {
	return toFixed(toFixed(a) - toFixed(b))
}

// FloatAdd does a + b, treating both float64s rounded to 3 decimal places of
// precision.
func FloatAdd(a, b float64) float64 {
	return toFixed(toFixed(a) + toFixed(b))
}
