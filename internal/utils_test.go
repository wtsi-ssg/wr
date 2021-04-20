/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
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

package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/math/convert"
	"golang.org/x/sys/unix"
)

// memIssue1 is the directory containing wrong memory info.
// testdata/linux/virtualmemory/issue1/proc/meminfo .
const memIssue1 string = "issue1"

func TestUtilsFuncs(t *testing.T) {
	testmap := map[string]int{
		"k1": 10,
		"k2": 30,
		"k3": 5,
		"k4": 15,
	}

	testmapmap := map[string]map[string]int{
		"k1": {"ka1": 10,
			"ka2": 20,
		},
		"k2": {"ka1": 5,
			"ka2": 30,
		},
		"k3": {"ka1": 100,
			"ka2": 200,
		},
		"k4": {"ka1": 50,
			"ka2": 2,
		},
	}

	Convey("Given a map[string]int create Keyval struct from it", t, func() {
		testmapNil := map[string]int{}
		tnil := createKeyvalFromMap(testmapNil)
		So(len(tnil), ShouldEqual, 0)

		t := createKeyvalFromMap(testmap)
		So(len(t), ShouldEqual, len(testmap))
	})

	Convey("Given a map[string]int create Keyval struct and sort the slice", t, func() {
		t := createKeyvalFromMap(testmap)
		t.sliceSort()
		So(t[0].Value, ShouldBeLessThanOrEqualTo, t[1].Value)
		So(t[1].Value, ShouldBeLessThanOrEqualTo, t[2].Value)
		So(t[2].Value, ShouldBeLessThanOrEqualTo, t[3].Value)
	})

	Convey("Given a map[string]map[string]int create Keyval struct and reverse sort the slice", t, func() {
		t := createKeyvalFromMapOfMap(testmapmap, "ka1")
		t.sliceSortReverse()
		So(t[3].Value, ShouldBeLessThanOrEqualTo, t[2].Value)
		So(t[2].Value, ShouldBeLessThanOrEqualTo, t[1].Value)
		So(t[1].Value, ShouldBeLessThanOrEqualTo, t[0].Value)
	})

	Convey("Given a map[string]int sort the Keyval struct", t, func() {
		So(sortKeyvalstruct(0, []keyvalueStruct{}), ShouldBeEmpty)

		t := createKeyvalFromMap(testmap)
		t.sliceSort()
		So(sortKeyvalstruct(len(testmap), t), ShouldResemble, []string{"k3", "k1", "k4", "k2"})
	})

	Convey("Given a map[string]int sort and reverse sort the map by value", t, func() {
		So(SortMapKeysByIntValue(testmap), ShouldResemble, []string{"k3", "k1", "k4", "k2"})
		So(ReverseSortMapKeysByIntValue(testmap), ShouldResemble, []string{"k2", "k4", "k1", "k3"})
	})

	Convey("Given a map[string]map[string]int sort and reverse sort it by value with a given criterion", t, func() {
		criterion := "ka1"
		So(SortMapKeysByMapIntValue(testmapmap, criterion), ShouldResemble, []string{"k2", "k1", "k4", "k3"})

		criterion = "ka2"
		So(ReverseSortMapKeysByMapIntValue(testmapmap, criterion), ShouldResemble, []string{"k3", "k2", "k1", "k4"})
	})

	Convey("Given a slice remove the duplicates from it and then sort it", t, func() {
		So(DedupSortStrings([]string{}), ShouldBeEmpty)

		testlist := []string{"k3", "k3", "k4", "k1"}

		So(DedupSortStrings(testlist), ShouldResemble, []string{"k1", "k3", "k4"})
	})

	Convey("Given a path starting with ~/ check it's absolute path", t, func() {
		So(TildaToHome(""), ShouldBeEmpty)

		home, herr := os.UserHomeDir()
		So(herr, ShouldEqual, nil)
		filepth := filepath.Join(home, "testing_absolute_path.text")
		_, err := os.Create(filepth)
		So(err, ShouldEqual, nil)

		So(TildaToHome("~/testing_absolute_path.text"), ShouldEqual, filepth)
		defer os.Remove(filepth)
	})

	Convey("Given a path to a file check it's content", t, func() {
		empContent, err := PathToContent("")
		So(err, ShouldNotBeNil)
		So(empContent, ShouldBeEmpty)

		home, herr := os.UserHomeDir()
		So(herr, ShouldEqual, nil)
		filepth := filepath.Join(home, "testing_pathtocontent.text")

		file, err := os.Create(filepth)
		So(err, ShouldEqual, nil)

		wrtn, err := file.WriteString("hello")
		So(err, ShouldEqual, nil)
		fmt.Printf("wrote %d bytes\n", wrtn)

		content, err := PathToContent(filepth)
		So(content, ShouldEqual, "hello")
		So(err, ShouldEqual, nil)

		content, err = PathToContent("random.txt")
		So(content, ShouldEqual, "")
		So(err, ShouldNotBeNil)

		defer os.Remove(filepth)
	})

	Convey("It can get the virtual memory of the system in MB", t, func() {
		memStat, err := ProcMeminfoMBs()
		So(memStat, ShouldNotEqual, 0)
		So(err, ShouldBeNil)

		if runtime.GOOS != "linux" {
			t.Skip("skipping test; test coverage is only for linux machines.")
		}

		memStat, err = ProcMeminfoMBs()
		So(memStat, ShouldNotEqual, 0)
		So(err, ShouldBeNil)

		totalSysMem, err := unix.SysctlUint64("hw.memsize")
		So(err, ShouldBeNil)

		So(convert.BytesToMB(totalSysMem), ShouldEqual, memStat)

		Convey("not with the wrong test data in linux", func() {
			origProc := os.Getenv("HOST_PROC")
			defer os.Setenv("HOST_PROC", origProc)

			// set HOST_PROC to testdata for wrong meminfo
			os.Setenv("HOST_PROC", filepath.Join("testdata/linux/virtualmemory/", memIssue1, "proc"))
			memWStat, errw := ProcMeminfoMBs()
			So(memWStat, ShouldEqual, 0)
			So(errw, ShouldNotBeNil)
		})
	})

	Convey("It can get the present working directory", t, func() {
		ctx := context.Background()
		pWD := GetPWD(ctx)
		So(pWD, ShouldNotBeEmpty)

		tempDir := os.TempDir()
		err := os.Chdir(tempDir)
		So(err, ShouldBeNil)

		pWD = GetPWD(ctx)
		So(strings.TrimSuffix(pWD, "/"), ShouldEndWith, strings.TrimSuffix(tempDir, "/"))
	})
}
