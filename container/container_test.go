/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>
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
package container

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// fileMode is the mode of the temp file created for testing.
const fileMode os.FileMode = 0600

// mockInteractor represents a mock implementation of container.Interactor.
type mockInteractor struct {
	ContainerListFn       func() ([]*Container, error)
	ContainerListInvoked  int
	ContainerStatsFn      func(string) (*Stats, error)
	ContainerStatsInvoked int
	ContainerKillFn       func(string) error
	ContainerKillInvoked  int
}

// ContainerList is a mock function which returns the list of containers.
func (m *mockInteractor) ContainerList(ctx context.Context) ([]*Container, error) {
	m.ContainerListInvoked++

	return m.ContainerListFn()
}

// ContainerStats is a mock function which returns the mem and cpu stats of a container
// given its ID.
func (m *mockInteractor) ContainerStats(ctx context.Context,
	containerID string) (*Stats, error) {
	m.ContainerStatsInvoked++

	return m.ContainerStatsFn(containerID)
}

// ContainerKill is a mock function which kills (removes the entry of) a container
// given its ID.
func (m *mockInteractor) ContainerKill(ctx context.Context, containerID string) error {
	m.ContainerKillInvoked++

	return m.ContainerKillFn(containerID)
}

// RemoveContainer removes the container from container list if found
// and returns the remaining containers.
func RemoveContainer(containerList []*Container, containerID string) []*Container {
	var remainingContainers []*Container

	for _, container := range containerList {
		if container.ID != containerID {
			remainingContainers = append(remainingContainers, container)
		}
	}

	return remainingContainers
}

func TestContainer(t *testing.T) {
	ctx := context.Background()

	Convey("Given a NewOperator", t, func() {
		// Create a list of dummy containers
		cntrList := []*Container{{
			ID: "container_id1", Names: []string{"/test_container1", "/test_container1_2"},
		}, {
			ID: "container_id2", Names: []string{"/test_container2"},
		}, {
			ID: "container_id3", Names: []string{"/test_container3"},
		}}

		// Create a client with list of dummy containers
		newOperator := NewOperator(&mockInteractor{
			ContainerListFn: func() ([]*Container, error) {
				return cntrList, nil
			},

			ContainerStatsFn: func(containerID string) (*Stats, error) {
				return &Stats{}, nil
			},

			ContainerKillFn: func(containerID string) error {
				remainContainers := RemoveContainer(cntrList, containerID)
				if len(cntrList) == len(remainContainers) {
					return &OperatorErr{Type: ErrContainerKill}
				}

				// Copy the remaining containers to cntrList
				cntrList = remainContainers

				return nil
			},
		},
		)

		// Create a client with no containers
		empNewOperator := NewOperator(&mockInteractor{
			ContainerListFn: func() ([]*Container, error) {
				return nil, &OperatorErr{Type: ErrContainerList}
			},

			ContainerStatsFn: func(containerID string) (*Stats, error) {
				return nil, &OperatorErr{Type: ErrContainerStats}
			},
		},
		)

		Convey("it can get the list of containers if exists", func() {
			clist, err := newOperator.GetCurrentContainers(ctx)
			So(err, ShouldBeNil)
			So(len(clist), ShouldEqual, 3)

			emplist, err := empNewOperator.GetCurrentContainers(ctx)
			So(err, ShouldNotBeNil)
			So(len(emplist), ShouldEqual, 0)
		})

		// Mark container_id3 as true in exisiting container, making it an "old" container
		newOperator.existingContainers["container_id3"] = true

		Convey("it can remember the current container IDs", func() {
			Convey("when the list of containers is non-empty", func() {
				err := newOperator.RememberCurrentContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(newOperator.existingContainers["container_id1"], ShouldBeTrue)
				So(newOperator.existingContainers["container_id2"], ShouldBeTrue)
				So(newOperator.existingContainers["container_id3"], ShouldBeTrue)

				Convey("and check for an unknown container in that list", func() {
					So(newOperator.existingContainers["container_id4"], ShouldBeFalse)
				})

				Convey("and check for a newly created container", func() {
					newCntr := &Container{
						ID: "container_id4", Names: []string{"/test_container4", "/test_container4_new"},
					}
					cntrList = append(cntrList, newCntr)

					clist, err := newOperator.GetCurrentContainers(ctx)
					So(err, ShouldBeNil)
					So(len(clist), ShouldEqual, 4)

					So(newOperator.existingContainers["container_id4"], ShouldBeFalse)

					err = newOperator.RememberCurrentContainerIDs(ctx)
					So(err, ShouldBeNil)
					So(newOperator.existingContainers["container_id4"], ShouldBeTrue)
				})
			})

			Convey("not when the list of containers is empty", func() {
				err := empNewOperator.RememberCurrentContainerIDs(ctx)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("it can get the list of new containers", func() {
			Convey("when the list is non-empty", func() {
				cntList, err := newOperator.GetNewContainers(ctx)
				So(err, ShouldBeNil)
				So(len(cntList), ShouldEqual, 2)

				err = newOperator.RememberCurrentContainerIDs(ctx)
				So(err, ShouldBeNil)

				cntList, err = newOperator.GetNewContainers(ctx)
				So(err, ShouldBeNil)
				So(len(cntList), ShouldEqual, 0)
			})

			Convey("not when the list is empty", func() {
				cntList, errc := empNewOperator.GetNewContainers(ctx)
				So(errc, ShouldNotBeNil)
				So(len(cntList), ShouldEqual, 0)
			})
		})

		Convey("it can get the container ids of the new containers", func() {
			Convey("when the list of containers is non-empty", func() {
				idList, err := newOperator.GetNewContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(len(idList), ShouldEqual, 2)
			})

			Convey("not when the list of containers is empty", func() {
				idList, errc := empNewOperator.GetNewContainerIDs(ctx)
				So(errc, ShouldNotBeNil)
				So(len(idList), ShouldEqual, 0)
			})
		})

		Convey("it can check for a valid container name", func() {
			clist, err := newOperator.GetCurrentContainers(ctx)
			So(err, ShouldBeNil)
			So(len(clist), ShouldEqual, 3)

			container1 := clist[0]
			Convey("for a correct name of the container", func() {
				So(newOperator.HasName("test_container1_2", container1), ShouldBeTrue)
			})

			Convey("but not for a wrong name of the container", func() {
				So(newOperator.HasName("wrong_name", container1), ShouldBeFalse)
			})
		})

		Convey("it can get a new container's id given a name", func() {
			Convey("when the list of containers is non-empty", func() {
				Convey("and a new container name is given", func() {
					name, err := newOperator.GetNewContainerIDByName(ctx, "test_container2")
					So(name, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("but not when an old container name is given", func() {
					name, err := newOperator.GetNewContainerIDByName(ctx, "test_container3")
					So(name, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("but not when an non-existing container name is given", func() {
					name, err := newOperator.GetNewContainerIDByName(ctx, "wrong_name")
					So(name, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})
			})

			Convey("not when the list of containers is empty", func() {
				name, err := empNewOperator.GetNewContainerIDByName(ctx, "wrong_name")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("it can verify a container id", func() {
			Convey("for a client with a non-empty list of containers", func() {
				Convey("when the container id is correct", func() {
					boolOut, err := newOperator.verifyID(ctx, "container_id1")
					So(boolOut, ShouldBeTrue)
					So(err, ShouldBeNil)
				})

				Convey("when the container id is wrong", func() {
					boolOut, err := newOperator.verifyID(ctx, "wrong_id")
					So(boolOut, ShouldBeFalse)
					So(err, ShouldBeNil)
				})
			})

			Convey("but not for a client with an empty list of containers", func() {
				boolOut, err := empNewOperator.verifyID(ctx, "wrong_id")
				So(boolOut, ShouldBeFalse)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("and given a file path/glob path, return the container id", func() {
			// Create some files containing container id
			containerTempDir, err := ioutil.TempDir("", "container_temp_")
			if err != nil {
				log.Fatal(err)
			}

			containerFile := filepath.Join(containerTempDir, "Container.txt")
			err = ioutil.WriteFile(containerFile, []byte("container_id2"), fileMode)
			So(err, ShouldBeNil)

			newContainerFile := filepath.Join(containerTempDir, "NewContainer.txt")
			err = ioutil.WriteFile(newContainerFile, []byte("container_id4"), fileMode)
			So(err, ShouldBeNil)

			wrongContainerFile := filepath.Join(containerTempDir, "WrongContainer.txt")
			err = ioutil.WriteFile(wrongContainerFile, []byte("container_id5"), fileMode)
			So(err, ShouldBeNil)

			containerEmptyFile := filepath.Join(containerTempDir, "containerEmpty.txt")
			err = ioutil.WriteFile(containerEmptyFile, []byte(""), fileMode)
			So(err, ShouldBeNil)

			Convey("When the file path", func() {
				Convey("doesn't exist", func() {
					containerNonExistingFile := filepath.Join(containerTempDir, "containerNonExisting.txt")
					id, err := newOperator.cidPathToID(ctx, containerNonExistingFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("has an empty file", func() {
					id, err := newOperator.cidPathToID(ctx, containerEmptyFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("contains a file with correct container id", func() {
					id, err := newOperator.cidPathToID(ctx, containerFile)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("contains a file with wrong container id", func() {
					id, err := newOperator.cidPathToID(ctx, wrongContainerFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})
			})

			Convey("When the file path is a glob and", func() {
				Convey("the path pattern is correct", func() {
					id, err := newOperator.cidPathGlobToID(ctx, containerTempDir+"/*")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("the path doesn't exist", func() {
					id, err := newOperator.cidPathGlobToID(ctx, "/randomContainerPath/*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("path pattern is wrong", func() {
					id, err := newOperator.cidPathGlobToID(ctx, "[")
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("there is no file containing the container id in the path", func() {
					id, err := newOperator.cidPathGlobToID(ctx, containerTempDir+"/container*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("not for a client with a empty list of containers", func() {
					id, err := empNewOperator.cidPathGlobToID(ctx, containerTempDir+"/*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})
			})

			Convey("Given a file path/glob file path and return a valid container id", func() {
				Convey("For a correct file path", func() {
					id, err := newOperator.GetContainerIDByPath(ctx, "Container.txt", containerTempDir)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("For a correct glob path", func() {
					id, err := newOperator.GetContainerIDByPath(ctx, containerTempDir+"/*", "")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("and a container id, it can get its stats", func() {
			Convey("for a client with a non-empty list of containers", func() {
				stats, err := newOperator.ContainerStats(ctx, "container_id2")
				So(stats.MemoryMB, ShouldBeZeroValue)
				So(stats.CPUSec, ShouldBeZeroValue)
				So(err, ShouldBeNil)
			})

			Convey("for a client with a empty list of containers", func() {
				stats, err := empNewOperator.ContainerStats(ctx, "container_id2")
				So(stats, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("and a container id, it can kill a container", func() {
			Convey("when the container exists", func() {
				err := newOperator.KillContainer(ctx, "container_id1")
				So(err, ShouldBeNil)

				clist, err := newOperator.GetCurrentContainers(ctx)
				So(len(clist), ShouldEqual, 2)
				So(err, ShouldBeNil)
			})

			Convey("not when the container doesn't exist", func() {
				err := newOperator.KillContainer(ctx, "container_id5")
				So(err, ShouldNotBeNil)

				clist, err := newOperator.GetCurrentContainers(ctx)
				So(len(clist), ShouldEqual, 3)
				So(err, ShouldBeNil)
			})
		})
	})
}
