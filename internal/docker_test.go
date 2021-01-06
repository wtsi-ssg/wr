package internal

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	. "github.com/smartystreets/goconvey/convey"
)

type mockDockerClient struct {
	// ContainerListEmptyFn      func() error
	// ContainerListEmptyInvoked int
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
)

// removeContainer removed the container from container list if found
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

// getContainerStats returns the container's stats given it's id and a list of all the containers.
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

	readerVar := ioutil.NopCloser(bytes.NewReader([]byte(readerCloserStats)))
	stats := types.ContainerStats{Body: readerVar, OSType: "linux"}

	return stats, nil
}

func TestDockerFuncs(t *testing.T) {
	Convey("Test to check nanosecondsToSec", t, func() {
		So(nanosecondsToSec(634736438394834), ShouldEqual, 634736)
		So(nanosecondsToSec(634736), ShouldEqual, 0)
		So(nanosecondsToSec(0), ShouldEqual, 0)
	})

	Convey("Test to check bytesToMB", t, func() {
		So(bytesToMB(634736438), ShouldEqual, 605)
		So(bytesToMB(1048576), ShouldEqual, 1)
		So(bytesToMB(0), ShouldEqual, 0)
	})

	Convey("Given a NewDockerClient get the current container list", t, func() {
		// Create a list of dummy containers
		cntrList := []types.Container{{
			ID: "container_id1", Names: []string{"/test_container1"},
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

		clist, err := dockerClient.GetCurrentContainers()
		So(err, ShouldBeNil)
		So(clist, ShouldNotBeNil)
		So(len(clist), ShouldEqual, 3)

		// Create a docker client with no containers
		empDockerClient := NewDockerClient(&mockDockerClient{ContainerListFn: func() ([]types.Container, error) {
			return []types.Container{}, &ContainerError{Type: ErrContainerListEmpty}
		}})

		emplist, err := empDockerClient.GetCurrentContainers()
		So(err, ShouldNotBeNil)
		So(emplist, ShouldBeEmpty)
		So(len(emplist), ShouldEqual, 0)

		// Create some files containing container id
		dockerTempDir, err := ioutil.TempDir("/tmp", "docker_temp_")
		if err != nil {
			log.Fatal(err)
		}

		dockerContainerFile := filepath.Join(dockerTempDir, "Container.txt")
		err = ioutil.WriteFile(dockerContainerFile, []byte("container_id2"), 0600)
		So(err, ShouldBeNil)

		dockerNewContainerFile := filepath.Join(dockerTempDir, "NewContainer.txt")
		err = ioutil.WriteFile(dockerNewContainerFile, []byte("container_id4"), 0600)
		So(err, ShouldBeNil)

		dockerNonExistingFile := filepath.Join(dockerTempDir, "dockerNonExisting.txt")

		dockerWrongCntrFile := filepath.Join(dockerTempDir, "dockerWrongCntr.txt")
		err = ioutil.WriteFile(dockerWrongCntrFile, []byte("container_id5"), 0600)
		So(err, ShouldBeNil)

		dockerEmptyFile := filepath.Join(dockerTempDir, "dockerEmpty.txt")
		err = ioutil.WriteFile(dockerEmptyFile, []byte(""), 0600)
		So(err, ShouldBeNil)

		Convey("Remember the current container IDs", func() {
			// For client with empty list of containers
			err := empDockerClient.RememberCurrentContainerIDs()
			So(err, ShouldNotBeNil)

			// For client with list of containers
			err = dockerClient.RememberCurrentContainerIDs()
			So(err, ShouldBeNil)
			So(dockerClient.existingContainers["container_id1"], ShouldBeTrue)
			So(dockerClient.existingContainers["container_id2"], ShouldBeTrue)
			So(dockerClient.existingContainers["container_id3"], ShouldBeTrue)

			// Check for an unknown container
			So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)

			// Create and add new dummy container
			newCntr := types.Container{
				ID: "container_id4", Names: []string{"/test_container4", "/test_container4_new"},
			}
			cntrList = append(cntrList, newCntr)

			clist, err = dockerClient.GetCurrentContainers()
			So(err, ShouldBeNil)
			So(clist, ShouldNotBeEmpty)
			So(len(clist), ShouldEqual, 4)

			So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)

			// Remember the new container and check it gets added to exisiting containers
			err = dockerClient.RememberCurrentContainerIDs()
			So(err, ShouldBeNil)
			So(dockerClient.existingContainers["container_id4"], ShouldBeTrue)

			// Mark container_id4 marked as false in exisiting container, making it "new" container
			dockerClient.existingContainers["container_id4"] = false
			So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)

			Convey("Get the container id of a new container", func() {
				// For client with empty list of containers
				id, err := empDockerClient.GetNewDockerContainerID()
				So(err, ShouldNotBeNil)
				So(id, ShouldBeEmpty)

				// For client with list of containers with container_id4 considered as a new container
				id, err = dockerClient.GetNewDockerContainerID()
				So(err, ShouldBeNil)
				So(id, ShouldEqual, "container_id4")

				// Mark the new container as seen now
				err = dockerClient.RememberCurrentContainerIDs()
				So(err, ShouldBeNil)

				id, err = dockerClient.GetNewDockerContainerID()
				So(err, ShouldNotBeNil)
				So(id, ShouldBeEmpty)
			})

			Convey("Verify a container id from a container list", func() {
				// For client with empty list of containers
				boolOut, err := empDockerClient.verifyID("wrong_id")
				So(boolOut, ShouldBeFalse)
				So(err, ShouldNotBeNil)

				// For client with list of containers
				boolOut, err = dockerClient.verifyID("wrong_id")
				So(boolOut, ShouldBeFalse)
				So(err, ShouldNotBeNil)

				boolOut, err = dockerClient.verifyID("container_id1")
				So(boolOut, ShouldBeTrue)
				So(err, ShouldBeNil)
			})

			Convey("Get container id given a name from its list of names", func() {
				container4 := clist[3]
				name, err := getIDByName("wrong_name", container4)
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				name, err = getIDByName("test_container4_new", container4)
				So(name, ShouldEqual, "container_id4")
				So(err, ShouldBeNil)
			})

			Convey("Get new container's id given a name from a container list", func() {
				// For client with empty list of containers
				name, err := empDockerClient.GetNewDockerContainerIDByName("wrong_name")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// For client with list of containers
				name, err = dockerClient.GetNewDockerContainerIDByName("wrong_name")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// New Container
				name, err = dockerClient.GetNewDockerContainerIDByName("test_container4")
				So(name, ShouldEqual, "container_id4")
				So(err, ShouldBeNil)

				// Remember the new container and check it gets added to exisiting containers
				err = dockerClient.RememberCurrentContainerIDs()
				So(err, ShouldBeNil)
				So(dockerClient.existingContainers["container_id4"], ShouldBeTrue)

				// Get id if there is no new container
				name, err = dockerClient.GetNewDockerContainerIDByName("test_container1")
				So(name, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// Mark container_id4 marked as false in exisiting container, making it "new" container
				dockerClient.existingContainers["container_id4"] = false
				So(dockerClient.existingContainers["container_id4"], ShouldBeFalse)
			})

			Convey("Given a file path read and return container id from a file", func() {
				// cidPathToID
				// File doesn't exists
				id, err := dockerClient.cidPathToID(dockerNonExistingFile)
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// Empty file
				id, err = dockerClient.cidPathToID(dockerEmptyFile)
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// Correct container id in file
				id, err = dockerClient.cidPathToID(dockerContainerFile)
				So(id, ShouldEqual, "container_id2")
				So(err, ShouldBeNil)

				// Wrong container id in file
				id, err = dockerClient.cidPathToID(dockerWrongCntrFile)
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// getContainerIDByPath
				// Correct container id in file
				id, err = getContainerIDByPath(dockerClient, dockerContainerFile)
				So(id, ShouldEqual, "container_id2")
				So(err, ShouldBeNil)

				// Wrong container id in file
				id, err = getContainerIDByPath(dockerClient, dockerWrongCntrFile)
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})

			Convey("Given a glob file path read and return a valid container id from a file", func() {
				// Correct paths pattern
				id, err := dockerClient.cidPathGlobToID(dockerTempDir + "/*")
				So(id, ShouldEqual, "container_id2")
				So(err, ShouldBeNil)

				// Path doesn't exists
				id, err = dockerClient.cidPathGlobToID("/tmp/randomDockerPath/*")
				So(id, ShouldBeEmpty)
				So(err, ShouldBeNil)

				// Bad Pattern to glob
				id, err = dockerClient.cidPathGlobToID("[")
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)

				// Paths without container id containing file
				id, err = dockerClient.cidPathGlobToID(dockerTempDir + "/docker*")
				So(id, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})

			Convey("Given a file path/glab file path and return a valid container id", func() {
				// Correct path
				id, err := dockerClient.GetDockerContainerIDByPath("Container.txt", dockerTempDir)
				So(id, ShouldEqual, "container_id2")
				So(err, ShouldBeNil)

				// Correct glob path
				id, err = dockerClient.GetDockerContainerIDByPath(dockerTempDir+"/*", "")
				So(id, ShouldEqual, "container_id2")
				So(err, ShouldBeNil)
			})

			Convey("Given a container id, kill this container", func() {
				So(dockerClient, ShouldNotBeNil)
				clist, err := dockerClient.GetCurrentContainers()
				So(len(clist), ShouldEqual, 4)
				So(err, ShouldBeNil)

				err = dockerClient.KillContainer("container_id1")
				So(err, ShouldBeNil)

				clist, err = dockerClient.GetCurrentContainers()
				So(len(clist), ShouldEqual, 3)
				So(err, ShouldBeNil)

				err = dockerClient.KillContainer("container_id5")
				So(err, ShouldNotBeNil)

				clist, err = dockerClient.GetCurrentContainers()
				So(len(clist), ShouldEqual, 3)
				So(err, ShouldBeNil)
			})

			Convey("Given a readercloser stat, decode and return the mem and cpu stats", func() {
				emptyRC := ioutil.NopCloser(bytes.NewReader([]byte("")))
				emptyReaderCloserStats := types.ContainerStats{Body: emptyRC, OSType: "linux"}

				mem, cpu, err := decodeContainerStats(emptyReaderCloserStats)
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)

				nonEmptyRC := ioutil.NopCloser(bytes.NewReader([]byte(readerCloserStats)))
				nonEmptyReaderCloserStats := types.ContainerStats{Body: nonEmptyRC, OSType: "linux"}

				mem, cpu, err = decodeContainerStats(nonEmptyReaderCloserStats)
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})

			Convey("Given a container id, get its stats", func() {
				So(dockerClient, ShouldNotBeNil)
				clist, err := dockerClient.GetCurrentContainers()
				So(len(clist), ShouldEqual, 4)
				So(err, ShouldBeNil)

				// Non existing container
				mem, cpu, err := dockerClient.ContainerStats("container_id5")
				So(mem, ShouldEqual, 0)
				So(cpu, ShouldEqual, 0)
				So(err, ShouldNotBeNil)

				// Existing container
				mem, cpu, err = dockerClient.ContainerStats("container_id2")
				So(mem, ShouldEqual, 1)
				So(cpu, ShouldEqual, 1244)
				So(err, ShouldBeNil)
			})
		})
	})
}
