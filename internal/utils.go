/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>, Ashwini Chhipa <ac55@sanger.ac.uk>
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

package internal

// this file has general utility functions.

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/wtsi-ssg/wr/clog"
	"github.com/wtsi-ssg/wr/math/convert"
)

// keyvalueStruct is the struct to define a key-value pair.
type keyvalueStruct struct {
	Key   string
	Value int
}

// keyvalueStructs is the struct to define a list of keyvalueStruct.
type keyvalueStructs []keyvalueStruct

// PathReadError records an path read error.
type PathReadError struct {
	path string
	Err  error
}

// Error returns an error related to path could not be read.
func (p *PathReadError) Error() string {
	return fmt.Sprintf("path [%s] could not be read: %s", p.path, p.Err)
}

// SortMapKeysByIntValue sorts the keys of a map[string]int by its values.
func SortMapKeysByIntValue(imap map[string]int) []string {
	// create keyval
	keyval := createKeyvalFromMap(imap)

	// sort the keyval
	keyval.sliceSort()

	// sort the map by its values and return the sorted keys
	return sortKeyvalstruct(len(imap), keyval)
}

// sliceSort function sorts the slice.
func (k keyvalueStructs) sliceSort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Value < k[j].Value
	})
}

// createKeyvalFromMap function creates a keyvaluestruct from map[string]int.
func createKeyvalFromMap(imap map[string]int) keyvalueStructs {
	keyval := keyvalueStructs{}

	for k, v := range imap {
		keyval = append(keyval, keyvalueStruct{k, v})
	}

	return keyval
}

// sortKeyvalstruct function sorts the keyvaluestruct by values and return the
// sorted keys.
func sortKeyvalstruct(maplen int, keyval []keyvalueStruct) []string {
	sortedKeys := make([]string, 0, maplen)
	for _, kv := range keyval {
		sortedKeys = append(sortedKeys, kv.Key)
	}

	return sortedKeys
}

// ReverseSortMapKeysByIntValue reverse sorts the keys of a map[string]int by
// its values.
func ReverseSortMapKeysByIntValue(imap map[string]int) []string {
	// create keyval
	keyval := createKeyvalFromMap(imap)

	// sort the keyval in reverse order
	keyval.sliceSortReverse()

	// sort the map by its values and return the sorted keys
	return sortKeyvalstruct(len(imap), keyval)
}

// sliceSortReverse function reverse sorts the slice.
func (k keyvalueStructs) sliceSortReverse() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Value > k[j].Value
	})
}

// SortMapKeysByMapIntValue sorts the keys of a map[string]map[string]int by a
// the values found at a given sub value.
func SortMapKeysByMapIntValue(imap map[string]map[string]int, criterion string) []string {
	// create keyval
	keyval := createKeyvalFromMapOfMap(imap, criterion)

	// sort the keyval
	keyval.sliceSort()

	// sort the map by its values and return the sorted keys
	return sortKeyvalstruct(len(imap), keyval)
}

// createKeyvalFromMapOfMap function creates a keyvaluestruct from
// map[string]map[string]int.
func createKeyvalFromMapOfMap(imap map[string]map[string]int, criterion string) keyvalueStructs {
	keyval := keyvalueStructs{}

	for k, v := range imap {
		keyval = append(keyval, keyvalueStruct{k, v[criterion]})
	}

	return keyval
}

// ReverseSortMapKeysByMapIntValue reverse sorts the keys of a
// map[string]map[string]int by a the values found at a given sub value.
func ReverseSortMapKeysByMapIntValue(imap map[string]map[string]int, criterion string) []string {
	// create keyval
	keyval := createKeyvalFromMapOfMap(imap, criterion)

	// sort the keyval in reverse order
	keyval.sliceSortReverse()

	// sort the map by its values and return the sorted keys
	return sortKeyvalstruct(len(imap), keyval)
}

// DedupSortStrings removes duplicates and then sorts the given strings,
// returning a new slice.
func DedupSortStrings(istrings []string) []string {
	if len(istrings) == 0 {
		return istrings
	}

	elementmap := make(map[string]bool)
	dedup := []string{}

	for _, entry := range istrings {
		if _, value := elementmap[entry]; !value {
			elementmap[entry] = true

			dedup = append(dedup, entry)
		}
	}

	sort.Strings(dedup)

	return dedup
}

// ProcMeminfoMBs uses gopsutil (amd64 freebsd, linux, windows, darwin, openbds
// only!) to find the total number of MBs of memory physically installed on the
// current system.
func ProcMeminfoMBs() (int, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	// convert bytes to MB
	return convert.BytesToMB(v.Total), err
}

// PathToContent takes the path to a file and returns its contents as a string.
// If path begins with a tilda, TildaToHome() is used to first convert the path
// to an absolute path, in order to find the file.
func PathToContent(path string) (string, error) {
	if path == "" {
		return "", &PathReadError{"", nil}
	}

	absPath := TildaToHome(path)

	contents, err := ioutil.ReadFile(absPath)
	if err != nil {
		return "", &PathReadError{absPath, err}
	}

	return string(contents), nil
}

// TildaToHome converts a path beginning with ~/ to the absolute path based in
// the current home directory. If that cannot be determined, path is returned
// unaltered.
func TildaToHome(path string) string {
	if path == "" {
		return ""
	}

	home, herr := os.UserHomeDir()
	if herr == nil && home != "" && strings.HasPrefix(path, "~/") {
		path = strings.TrimLeft(path, "~/")
		path = filepath.Join(home, path)
	}

	return path
}

// GetPWD returns the present working directory.
func GetPWD(ctx context.Context) string {
	pwd, err := os.Getwd()
	if err != nil {
		clog.Fatal(ctx, err.Error())
	}

	return pwd
}
