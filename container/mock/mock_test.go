/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Authors: Ashwini Chhipa <ac55@sanger.ac.uk>
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

package mock

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/container"
)

func TestContainerMock(t *testing.T) {
	ctx := context.Background()

	Convey("Interactor implements container.Interactor", t, func() {
		var _ container.Interactor = (*Interactor)(nil)
	})

	Convey("ContainerList, ContainerStats and ContainerKill methods are just mocks", t, func() {
		mockInteractor := &Interactor{
			ContainerListFn: func() ([]types.Container, error) {
				return nil, nil
			},

			ContainerKillFn: func(containerID string) error {
				return nil
			},

			ContainerStatsFn: func(containerID string) (types.ContainerStats, error) {
				return types.ContainerStats{}, nil
			},
		}

		Convey("List the containers", func() {
			cnList, err := mockInteractor.ContainerList(ctx, types.ContainerListOptions{})
			So(cnList, ShouldBeNil)
			So(err, ShouldBeNil)
			So(mockInteractor.ContainerListInvoked, ShouldEqual, 1)
		})

		Convey("Get the container's stats given it's ID", func() {
			cnStats, err := mockInteractor.ContainerStats(ctx, "container_id1", false)
			So(cnStats.Body, ShouldBeNil)
			So(err, ShouldBeNil)
			So(mockInteractor.ContainerStatsInvoked, ShouldEqual, 1)
		})

		Convey("Kill the container, given it's ID", func() {
			err := mockInteractor.ContainerKill(ctx, "container_id1", "SIGKILL")
			So(err, ShouldBeNil)
			So(mockInteractor.ContainerKillInvoked, ShouldEqual, 1)
		})

		Convey("CreateContainerStats()", func() {
			createdStats := CreateContainerStats()
			So(createdStats, ShouldNotBeNil)
		})

		Convey("GetContainerStats() given it's id and list of containers", func() {
			cntrList := []types.Container{{
				ID: "container_id1", Names: []string{"/test_container1", "/test_container1_2"},
			}}

			Convey("when container id is correct", func() {
				stats, err := GetContainerStats(cntrList, "container_id1")
				So(stats.Body, ShouldNotBeNil)
				So(err, ShouldBeNil)
			})

			Convey("when container id is wrong", func() {
				stats, err := GetContainerStats(cntrList, "container_id3")
				So(stats.Body, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("RemoveContainer() given it's id and list of containers", func() {
			cntrList := []types.Container{{
				ID: "container_id1", Names: []string{"/test_container1", "/test_container1_2"},
			}}

			Convey("when container id is wrong", func() {
				remCntr := RemoveContainer(cntrList, "container_id3")
				So(remCntr, ShouldNotBeEmpty)
			})

			Convey("when container id is corrent", func() {
				remCntr := RemoveContainer(cntrList, "container_id1")
				So(remCntr, ShouldBeEmpty)
			})
		})
	})
}
