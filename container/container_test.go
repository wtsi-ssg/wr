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
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/container/mock"
)

const (
	dockerFileMode os.FileMode = 0600
)

func TestDockerFuncs(t *testing.T) {
	Convey("Given a NewContainerOperator", t, func() {
		ctx := context.Background()

		// Create a list of dummy containers
		cntrList := []types.Container{{
			ID: "container_id1", Names: []string{"/test_container1", "/test_container1_2"},
		}, {
			ID: "container_id2", Names: []string{"/test_container2"},
		}, {
			ID: "container_id3", Names: []string{"/test_container3"},
		}}

		// Create a docker client with list of dummy containers
		dockerOperator := NewContainerOperator(&mock.Interactor{
			ContainerListFn: func() ([]types.Container, error) {
				return cntrList, nil
			},

			ContainerKillFn: func(containerID string) error {
				remainingCntr := mock.RemoveContainer(cntrList, containerID)
				if len(cntrList) == len(remainingCntr) {
					return &ErrorType{Type: ErrContainerNotFound}
				}

				// Copy the remaining containers to cntrList
				cntrList = remainingCntr

				return nil
			},

			ContainerStatsFn: func(containerID string) (types.ContainerStats, error) {
				return mock.GetContainerStats(cntrList, containerID)
			},
		},
		)

		// Create a docker client with no containers
		empDockerOperator := NewContainerOperator(&mock.Interactor{ContainerListFn: func() ([]types.Container, error) {
			return []types.Container{}, &ErrorType{Type: ErrContainerList}
		}})

		Convey("it can get the list of containers if exists", func() {
			clist, err := dockerOperator.GetCurrentDockerContainers(ctx)
			So(err, ShouldBeNil)
			So(len(clist), ShouldEqual, 3)

			emplist, err := empDockerOperator.GetCurrentDockerContainers(ctx)

			So(err, ShouldNotBeNil)
			So(len(emplist), ShouldEqual, 0)
		})

		// Mark container_id3 as true in exisiting container, making it an "old" container
		dockerOperator.existingContainers["container_id3"] = true

		Convey("it can remember the current container IDs", func() {
			Convey("when the list of containers is non-empty", func() {
				err := dockerOperator.RememberCurrentDockerContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(dockerOperator.existingContainers["container_id1"], ShouldBeTrue)
				So(dockerOperator.existingContainers["container_id2"], ShouldBeTrue)
				So(dockerOperator.existingContainers["container_id3"], ShouldBeTrue)

				Convey("and check for an unknown container in that list", func() {
					So(dockerOperator.existingContainers["container_id4"], ShouldBeFalse)
				})

				Convey("and check for a newly created container", func() {
					newCntr := types.Container{
						ID: "container_id4", Names: []string{"/test_container4", "/test_container4_new"},
					}
					cntrList = append(cntrList, newCntr)

					clist, err := dockerOperator.GetCurrentDockerContainers(ctx)
					So(err, ShouldBeNil)
					So(len(clist), ShouldEqual, 4)

					So(dockerOperator.existingContainers["container_id4"], ShouldBeFalse)

					err = dockerOperator.RememberCurrentDockerContainerIDs(ctx)
					So(err, ShouldBeNil)
					So(dockerOperator.existingContainers["container_id4"], ShouldBeTrue)
				})
			})

			Convey("not when the list of containers is empty", func() {
				err := empDockerOperator.RememberCurrentDockerContainerIDs(ctx)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("it can get the list of new containers", func() {
			Convey("when the list is non-empty", func() {
				cntList, err := dockerOperator.GetNewDockerContainers(ctx)
				So(err, ShouldBeNil)
				So(len(cntList), ShouldEqual, 2)

				err = dockerOperator.RememberCurrentDockerContainerIDs(ctx)
				So(err, ShouldBeNil)

				cntList, err = dockerOperator.GetNewDockerContainers(ctx)
				So(err, ShouldNotBeNil)
				So(len(cntList), ShouldEqual, 0)
			})

			Convey("not when the list is empty", func() {
				cntList, errc := empDockerOperator.GetNewDockerContainers(ctx)
				So(errc, ShouldNotBeNil)
				So(len(cntList), ShouldEqual, 0)
			})
		})

		Convey("it can get the container ids of the new containers", func() {
			Convey("when the list of containers is non-empty", func() {
				idList, err := dockerOperator.GetNewDockerContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(len(idList), ShouldEqual, 2)
			})

			Convey("not when the list of containers is empty", func() {
				idList, errc := empDockerOperator.GetNewDockerContainerIDs(ctx)
				So(errc, ShouldNotBeNil)
				So(len(idList), ShouldEqual, 0)
			})
		})

		Convey("it can get a container's id given a name from its list of names", func() {
			clist, err := dockerOperator.GetCurrentDockerContainers(ctx)
			So(err, ShouldBeNil)
			So(len(clist), ShouldEqual, 3)

			container1 := clist[0]
			Convey("for a correct name of the container", func() {
				name, err := getIDByName("test_container1_2", container1)
				So(name, ShouldEqual, "container_id1")
				So(err, ShouldBeNil)
			})

			Convey("but not for a wrong name of the container", func() {
				name, err := getIDByName("wrong_name", container1)
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("it can get a new container's id given a name", func() {
			Convey("when the list of containers is non-empty", func() {
				Convey("and a new container name is given", func() {
					name, err := dockerOperator.GetNewDockerContainerIDByName(ctx, "test_container2")
					So(name, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("but not when an old container name is given", func() {
					name, err := dockerOperator.GetNewDockerContainerIDByName(ctx, "test_container3")
					So(name, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("but not when an non-existing container name is given", func() {
					name, err := dockerOperator.GetNewDockerContainerIDByName(ctx, "wrong_name")
					So(name, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})
			})

			Convey("not when the list of containers is empty", func() {
				name, err := empDockerOperator.GetNewDockerContainerIDByName(ctx, "wrong_name")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("it can verify a container id", func() {
			Convey("for a client with a non-empty list of containers", func() {
				Convey("when the container id is correct", func() {
					boolOut, err := dockerOperator.verifyID(ctx, "container_id1")
					So(boolOut, ShouldBeTrue)
					So(err, ShouldBeNil)
				})

				Convey("when the container id is wrong", func() {
					boolOut, err := dockerOperator.verifyID(ctx, "wrong_id")
					So(boolOut, ShouldBeFalse)
					So(err, ShouldNotBeNil)
				})
			})

			Convey("but not for a client with an empty list of containers", func() {
				boolOut, err := empDockerOperator.verifyID(ctx, "wrong_id")
				So(boolOut, ShouldBeFalse)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("and given a file path/glob path, return the container id", func() {
			// Create some files containing container id
			dockerTempDir, err := ioutil.TempDir("", "docker_temp_")
			if err != nil {
				log.Fatal(err)
			}

			dockerContainerFile := filepath.Join(dockerTempDir, "Container.txt")
			err = ioutil.WriteFile(dockerContainerFile, []byte("container_id2"), dockerFileMode)
			So(err, ShouldBeNil)

			dockerNewContainerFile := filepath.Join(dockerTempDir, "NewContainer.txt")
			err = ioutil.WriteFile(dockerNewContainerFile, []byte("container_id4"), dockerFileMode)
			So(err, ShouldBeNil)

			dockerWrongCntrFile := filepath.Join(dockerTempDir, "dockerWrongCntr.txt")
			err = ioutil.WriteFile(dockerWrongCntrFile, []byte("container_id5"), dockerFileMode)
			So(err, ShouldBeNil)

			dockerEmptyFile := filepath.Join(dockerTempDir, "dockerEmpty.txt")
			err = ioutil.WriteFile(dockerEmptyFile, []byte(""), dockerFileMode)
			So(err, ShouldBeNil)

			Convey("When the file path", func() {
				Convey("doesn't exist", func() {
					dockerNonExistingFile := filepath.Join(dockerTempDir, "dockerNonExisting.txt")
					id, err := dockerOperator.cidPathToID(ctx, dockerNonExistingFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("has an empty file", func() {
					id, err := dockerOperator.cidPathToID(ctx, dockerEmptyFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("contains a file with correct container id", func() {
					id, err := dockerOperator.cidPathToID(ctx, dockerContainerFile)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("contains a file with wrong container id", func() {
					id, err := dockerOperator.cidPathToID(ctx, dockerWrongCntrFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})
			})

			Convey("When the file path is a glob and", func() {
				Convey("the path pattern is correct", func() {
					id, err := dockerOperator.cidPathGlobToID(ctx, dockerTempDir+"/*")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("the path doesn't exist", func() {
					id, err := dockerOperator.cidPathGlobToID(ctx, "/randomDockerPath/*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("path pattern is wrong", func() {
					id, err := dockerOperator.cidPathGlobToID(ctx, "[")
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("there is no file containing the container id in the path", func() {
					id, err := dockerOperator.cidPathGlobToID(ctx, dockerTempDir+"/docker*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})
			})

			Convey("Given a file path/glob file path and return a valid container id", func() {
				Convey("For a correct file path", func() {
					id, err := dockerOperator.GetDockerContainerIDByPath(ctx, "Container.txt", dockerTempDir)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("For a correct glob path", func() {
					id, err := dockerOperator.GetDockerContainerIDByPath(ctx, dockerTempDir+"/*", "")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("and given a readercloser stat, decode and return the mem and cpu stats", func() {
			Convey("when the readcloser stats are empty", func() {
				emptyRC := ioutil.NopCloser(bytes.NewReader([]byte("")))
				emptyReaderCloserStats := types.ContainerStats{Body: emptyRC, OSType: "linux"}

				mem, cpu, err := decodeDockerContainerStats(emptyReaderCloserStats)
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)
			})

			Convey("when the readcloser stats are non-empty", func() {
				nonEmptyRC := ioutil.NopCloser(bytes.NewReader([]byte(mock.DockerReaderCloserStats)))
				nonEmptyReaderCloserStats := types.ContainerStats{Body: nonEmptyRC, OSType: "linux"}

				mem, cpu, err := decodeDockerContainerStats(nonEmptyReaderCloserStats)
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})
		})

		Convey("and a container id, it can get its stats", func() {
			Convey("when the container exists", func() {
				mem, cpu, err := dockerOperator.ContainerStats(ctx, "container_id2")
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})

			Convey("not when the container doesn't exist", func() {
				mem, cpu, err := dockerOperator.ContainerStats(ctx, "container_id5")
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("and a container id, it can kill a container", func() {
			Convey("when the container exists", func() {
				err := dockerOperator.KillDockerContainer(ctx, "container_id1")
				So(err, ShouldBeNil)

				clist, err := dockerOperator.GetCurrentDockerContainers(ctx)
				So(len(clist), ShouldEqual, 2)
				So(err, ShouldBeNil)
			})

			Convey("not when the container doesn't exist", func() {
				err := dockerOperator.KillDockerContainer(ctx, "container_id5")
				So(err, ShouldNotBeNil)

				clist, err := dockerOperator.GetCurrentDockerContainers(ctx)
				So(len(clist), ShouldEqual, 3)
				So(err, ShouldBeNil)
			})
		})
	})
}
