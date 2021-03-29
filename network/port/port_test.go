package port

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPort(t *testing.T) {
	Convey("Given a Checker", t, func() {
		checker, err := NewChecker("localhost")
		So(err, ShouldBeNil)
		So(checker, ShouldNotBeNil)

		Convey("You can get an available port number", func() {
			port, err := checker.availablePort()
			So(err, ShouldBeNil)
			So(port, ShouldBeBetweenOrEqual, 1, maxPort)
			So(len(checker.ports), ShouldEqual, 1)
			So(checker.ports[port], ShouldBeTrue)

			err = checker.release(err)
			So(err, ShouldBeNil)
		})

		Convey("portsAfter works", func() {
			portsBeforeAfterTest(checker,
				func() []int { return checker.portsAfter(10) },
				[]int{9, 12, 13, 15}, 11, []int{11, 12, 13})
		})

		Convey("portsBefore works", func() {
			portsBeforeAfterTest(checker,
				func() []int { return checker.portsBefore(10) },
				[]int{11, 8, 7, 5}, 9, []int{7, 8, 9})
		})

		Convey("checkRange returns nothing with no available ports", func() {
			set, has := checker.checkRange(10, 4)
			So(has, ShouldBeFalse)
			So(len(set), ShouldEqual, 0)

			Convey("but returns ports above starting point", func() {
				rangeTest(checker, []int{9, 11, 12, 13, 14}, []int{10, 11, 12, 13})
			})

			Convey("but returns ports below starting point", func() {
				rangeTest(checker, []int{11, 9, 8, 7, 6}, []int{7, 8, 9, 10})
			})

			Convey("but returns ports below and above starting point", func() {
				rangeTest(checker, []int{8, 9, 11, 12}, []int{8, 9, 10, 11})
			})

			Convey("and returns nothing with non-contiguous available ports", func() {
				setPortsTrue(checker, 7, 8, 12, 13)

				set, has := checker.checkRange(10, 4)
				So(has, ShouldBeFalse)
				So(len(set), ShouldEqual, 0)
			})
		})

		Convey("You can get a range of available ports multiple times in a row", func() {
			min, max, err := checker.AvailableRange(2)
			So(err, ShouldBeNil)
			So(min, ShouldBeBetweenOrEqual, 1, maxPort)
			So(max, ShouldEqual, min+1)

			min, max, err = checker.AvailableRange(67)
			So(err, ShouldBeNil)
			So(min, ShouldBeBetweenOrEqual, 1, maxPort)
			So(max, ShouldEqual, min+66)

			min, max, err = checker.AvailableRange(67)
			So(err, ShouldBeNil)
			So(min, ShouldBeBetweenOrEqual, 1, maxPort)
			So(max, ShouldEqual, min+66)
		})
	})
}

func setPortsTrue(checker *Checker, ports ...int) {
	for _, port := range ports {
		checker.ports[port] = true
	}
}

func portsBeforeAfterTest(checker *Checker, cb func() []int, truePorts []int, changePort int, expected []int) {
	result := cb()
	So(len(result), ShouldEqual, 0)

	setPortsTrue(checker, truePorts...)

	result = cb()
	So(len(result), ShouldEqual, 0)

	setPortsTrue(checker, changePort)

	result = cb()
	So(len(result), ShouldEqual, 3)
	So(result, ShouldResemble, expected)
}

func rangeTest(checker *Checker, truePorts []int, expected []int) {
	setPortsTrue(checker, truePorts...)
	set, has := checker.checkRange(10, 4)
	So(has, ShouldBeTrue)
	So(len(set), ShouldEqual, 4)
	So(set, ShouldResemble, expected)
}
