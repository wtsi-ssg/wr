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

package internal

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
)

type mockDockerClient struct {
	ContainerListFn       func() ([]types.Container, error)
	ContainerListInvoked  int
	ContainerKillFn       func(string) error
	ContainerKillInvoked  int
	ContainerStatsFn      func(string) (types.ContainerStats, error)
	ContainerStatsInvoked int
}

// ContainerList is a mock function which returns the list of containers.
func (m *mockDockerClient) ContainerList(ctx context.Context,
	options types.ContainerListOptions) ([]types.Container, error) {
	m.ContainerListInvoked++

	return m.ContainerListFn()
}

// ContainerKill is a mock function which kills (removes entry of) a container
// given its ID.
func (m *mockDockerClient) ContainerKill(ctx context.Context, containerID, signal string) error {
	m.ContainerKillInvoked++

	return m.ContainerKillFn(containerID)
}

// ContainerStats is a mock function which returns the mem and cpu stats of a container
// given its ID.
func (m *mockDockerClient) ContainerStats(ctx context.Context,
	containerID string, stream bool) (types.ContainerStats, error) {
	m.ContainerStatsInvoked++

	return m.ContainerStatsFn(containerID)
}

const (
	readerCloserStats = `{
		"read":"2021-01-05T11:42:54.959351591Z",
		"preread":"2021-01-05T11:42:53.949728039Z",
		"pids_stats":{"current":4},
		"blkio_stats":{},
		"num_procs":0,
		"storage_stats":{},
		"cpu_stats":{
			"cpu_usage":{
				"total_usage":1244741231366,
				"percpu_usage":[924236203020,320505028346],
				"usage_in_kernelmode":9190000000,
				"usage_in_usermode":653150000000
			},
			"system_cpu_usage":2053540000000,
			"online_cpus":2,
			"throttling_data":{"periods":0,"throttled_periods":0,"throttled_time":0}
		},
		"precpu_stats":{},
		"memory_stats":{
			"usage":57921536,
			"max_usage":115904512,
			"stats":{
				"active_anon":1216512,
				"active_file":41766912,
				"cache":53268480,
				"dirty":135168,
				"hierarchical_memory_limit":9223372036854771712,
				"hierarchical_memsw_limit":9223372036854771712,
				"inactive_anon":0,
				"inactive_file":11354112,
				"mapped_file":3514368,
				"pgfault":97911,
				"pgmajfault":165,
				"pgpgin":66198,
				"pgpgout":52876,
			    "rss":1048576,
				"rss_huge":0,
				"total_active_anon":1216512,
				"total_active_file":41766912,
				"total_cache":53268480,
				"total_dirty":135168,
				"total_inactive_anon":0,
				"total_inactive_file":11354112,
				"total_mapped_file":3514368,
				"total_pgfault":97911,
				"total_pgmajfault":165,
				"total_pgpgin":66198,
				"total_pgpgout":52876,
				"total_rss":1048576,
				"total_rss_huge":0,
				"total_unevictable":0,
				"total_writeback":0,
				"unevictable":0,
				"writeback":0},
				"limit":2084458496
			},
			"name":"/test_container2",
			"id":"container_id2",
			"networks":{}
		}`

	dockerFileMode os.FileMode = 0600
)

// removeContainer removes the container from container list if found
// and returns the remaining containers.
func removeContainer(cntrList []types.Container, containerID string) []types.Container {
	remainingCntr := []types.Container{}

	for _, cntr := range cntrList {
		if cntr.ID != containerID {
			remainingCntr = append(remainingCntr, cntr)
		}
	}

	return remainingCntr
}

// getContainerStats creates and returns the container's stats
// if the container id is found in the given list of containers.
func getContainerStats(cntrList []types.Container, containerID string) (types.ContainerStats, error) {
	found := 0

	for _, cntr := range cntrList {
		if cntr.ID == containerID {
			found = 1

			break
		}
	}

	if found == 0 {
		return types.ContainerStats{}, &ContainerError{Type: ErrContainerNotFound}
	}

	return createContainerStats(), nil
}

// createContainerStats returns the stats of a container, in this case it's a dummy data.
func createContainerStats() types.ContainerStats {
	// create the stats
	readerVar := ioutil.NopCloser(bytes.NewReader([]byte(readerCloserStats)))
	stats := types.ContainerStats{Body: readerVar, OSType: "linux"}

	return stats
}

func TestDockerFuncs(t *testing.T) {
	Convey("Given a NewDockerClient get the current container list", t, func() {
		// Create a context
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
		dockerClient := NewDockerClient(
			&mockDockerClient{
				ContainerListFn: func() ([]types.Container, error) {
					return cntrList, nil
				},

				ContainerKillFn: func(containerID string) error {
					remainingCntr := removeContainer(cntrList, containerID)
					if len(cntrList) == len(remainingCntr) {
						return &ContainerError{Type: ErrContainerNotFound}
					}

					// Copy the remaining containers to cntrList
					cntrList = remainingCntr

					return nil
				},

				ContainerStatsFn: func(containerID string) (types.ContainerStats, error) {
					return getContainerStats(cntrList, containerID)
				},
			},
		)

		clist, err := dockerClient.GetCurrentContainers(ctx)
		So(err, ShouldBeNil)
		So(len(clist), ShouldEqual, 3)

		// Create a docker client with no containers
		empDockerClient := NewDockerClient(&mockDockerClient{ContainerListFn: func() ([]types.Container, error) {
			return []types.Container{}, &ContainerError{Type: ErrContainerList}
		}})

		emplist, err := empDockerClient.GetCurrentContainers(ctx)
		So(err, ShouldNotBeNil)
		So(len(emplist), ShouldEqual, 0)

		// Mark container_id3 as true in exisiting container, making it an "old" container
		dockerClient.existingContainers["container_id3"] = true
		So(dockerClient.existingContainers["container_id3"], ShouldBeTrue)

		Convey("Remember the current container IDs", func() {
			Convey("For a client with empty list of containers", func() {
				err = empDockerClient.RememberCurrentContainerIDs(ctx)
				So(err, ShouldNotBeNil)
			})

			Convey("For a client with non-empty list of containers", func() {
				err = dockerClient.RememberCurrentContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(dockerClient.existingContainers["container_id1"], ShouldBeTrue)
				So(dockerClient.existingContainers["container_id2"], ShouldBeTrue)
				So(dockerClient.existingContainers["container_id3"], ShouldBeTrue)

				Convey("Check for an unknown container", func() {
					So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)
				})

				Convey("Create and check for a new container", func() {
					newCntr := types.Container{
						ID: "container_id4", Names: []string{"/test_container4", "/test_container4_new"},
					}
					cntrList = append(cntrList, newCntr)

					clist, err = dockerClient.GetCurrentContainers(ctx)
					So(err, ShouldBeNil)
					So(len(clist), ShouldEqual, 4)

					So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)

					err = dockerClient.RememberCurrentContainerIDs(ctx)
					So(err, ShouldBeNil)
					So(dockerClient.existingContainers["container_id4"], ShouldBeTrue)
				})
			})
		})

		Convey("Get the container ids of the new containers", func() {
			Convey("For a client with empty list of containers", func() {
				idList, errc := empDockerClient.GetNewDockerContainerIDs(ctx)
				So(errc, ShouldNotBeNil)
				So(len(idList), ShouldEqual, 0)
			})

			Convey("For a client with non-empty list of containers", func() {
				idList, err := dockerClient.GetNewDockerContainerIDs(ctx)
				So(err, ShouldBeNil)
				So(len(idList), ShouldEqual, 2)

				err = dockerClient.RememberCurrentContainerIDs(ctx)
				So(err, ShouldBeNil)

				idList, err = dockerClient.GetNewDockerContainerIDs(ctx)
				So(err, ShouldNotBeNil)
				So(len(idList), ShouldEqual, 0)
			})
		})

		Convey("Get container id given a name from its list of names", func() {
			container1 := clist[0]
			Convey("For a wrong name of the container", func() {
				name, err := getIDByName("wrong_name", container1)
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})

			Convey("For a correct name of the container", func() {
				name, err := getIDByName("test_container1_2", container1)
				So(name, ShouldEqual, "container_id1")
				So(err, ShouldBeNil)
			})
		})

		Convey("Get new container's id given a name from a container list", func() {
			Convey("For a client with empty list of containers", func() {
				name, err := empDockerClient.GetNewDockerContainerIDByName(ctx, "wrong_name")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})

			Convey("For a client with non-empty list of containers", func() {
				Convey("When the container name is wrong", func() {
					name, err := dockerClient.GetNewDockerContainerIDByName(ctx, "wrong_name")
					So(name, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("When a new container name is passed", func() {
					name, err := dockerClient.GetNewDockerContainerIDByName(ctx, "test_container2")
					So(name, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("When an old container name is passed", func() {
					name, err := dockerClient.GetNewDockerContainerIDByName(ctx, "test_container3")
					So(name, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Verify a container id from a container list", func() {
			Convey("For a client with an empty list of containers", func() {
				boolOut, err := empDockerClient.verifyID(ctx, "wrong_id")
				So(boolOut, ShouldBeFalse)
				So(err, ShouldNotBeNil)
			})

			Convey("For a client with a non-empty list of containers", func() {
				Convey("When the container id is wrong", func() {
					boolOut, err := dockerClient.verifyID(ctx, "wrong_id")
					So(boolOut, ShouldBeFalse)
					So(err, ShouldNotBeNil)
				})

				Convey("When the container id is correct", func() {
					boolOut, err := dockerClient.verifyID(ctx, "container_id1")
					So(boolOut, ShouldBeTrue)
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given a file path/glob path, return the container id", func() {
			// Create some files containing container id
			dockerTempDir, err := ioutil.TempDir("/tmp", "docker_temp_")
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
					id, err := dockerClient.cidPathToID(ctx, dockerNonExistingFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("has an empty file", func() {
					id, err := dockerClient.cidPathToID(ctx, dockerEmptyFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("contains a file with correct container id", func() {
					id, err := dockerClient.cidPathToID(ctx, dockerContainerFile)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("contains a file with wrong container id", func() {
					id, err := dockerClient.cidPathToID(ctx, dockerWrongCntrFile)
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})
			})

			Convey("When the file path is a glob and", func() {
				Convey("the path pattern is correct", func() {
					id, err := dockerClient.cidPathGlobToID(ctx, dockerTempDir+"/*")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("the path doesn't exist", func() {
					id, err := dockerClient.cidPathGlobToID(ctx, "/randomDockerPath/*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})

				Convey("path pattern is wrong", func() {
					id, err := dockerClient.cidPathGlobToID(ctx, "[")
					So(id, ShouldBeEmpty)
					So(err, ShouldNotBeNil)
				})

				Convey("there is no file containing the container id in the path", func() {
					id, err := dockerClient.cidPathGlobToID(ctx, dockerTempDir+"/docker*")
					So(id, ShouldBeEmpty)
					So(err, ShouldBeNil)
				})
			})

			Convey("Given a file path/glob file path and return a valid container id", func() {
				Convey("For a correct file path", func() {
					id, err := dockerClient.GetDockerContainerIDByPath(ctx, "Container.txt", dockerTempDir)
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})

				Convey("For a correct glob path", func() {
					id, err := dockerClient.GetDockerContainerIDByPath(ctx, dockerTempDir+"/*", "")
					So(id, ShouldEqual, "container_id2")
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given a readercloser stat, decode and return the mem and cpu stats", func() {
			Convey("For empty readcloser stats", func() {
				emptyRC := ioutil.NopCloser(bytes.NewReader([]byte("")))
				emptyReaderCloserStats := types.ContainerStats{Body: emptyRC, OSType: "linux"}

				mem, cpu, err := decodeContainerStats(emptyReaderCloserStats)
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)
			})

			Convey("For a non-empty readcloser stats", func() {
				nonEmptyRC := ioutil.NopCloser(bytes.NewReader([]byte(readerCloserStats)))
				nonEmptyReaderCloserStats := types.ContainerStats{Body: nonEmptyRC, OSType: "linux"}

				mem, cpu, err := decodeContainerStats(nonEmptyReaderCloserStats)
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a container id, get its stats", func() {
			Convey("For a non-existing container", func() {
				mem, cpu, err := dockerClient.ContainerStats(ctx, "container_id5")
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)
			})

			Convey("For an existing container", func() {
				mem, cpu, err := dockerClient.ContainerStats(ctx, "container_id2")
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a container id, kill this container", func() {
			clist, err := dockerClient.GetCurrentContainers(ctx)
			So(len(clist), ShouldEqual, 3)
			So(err, ShouldBeNil)

			Convey("For an existing container", func() {
				err = dockerClient.KillContainer(ctx, "container_id1")
				So(err, ShouldBeNil)

				clist, err = dockerClient.GetCurrentContainers(ctx)
				So(len(clist), ShouldEqual, 2)
				So(err, ShouldBeNil)
			})

			Convey("For a non-existing container", func() {
				err = dockerClient.KillContainer(ctx, "container_id5")
				So(err, ShouldNotBeNil)

				clist, err = dockerClient.GetCurrentContainers(ctx)
				So(len(clist), ShouldEqual, 3)
				So(err, ShouldBeNil)
			})
		})
	})
}
