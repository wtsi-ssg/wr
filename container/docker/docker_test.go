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

package docker

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	cn "github.com/docker/docker/api/types/container"
	nw "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/container"
)

// testReaderCloserStats is the dummy ReaderCloserStats data used for
// ContainerStats testing.
const testReaderCloserStats = `{
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

// createContainers creates and starts the test containers, given a list of
// container names.
func createContainers(ctx context.Context, cli *client.Client, containerNames []string) ([]string, error) {
	cntrIDs := make([]string, len(containerNames))

	for idx, cname := range containerNames {
		// create the docker container
		cbody, err := cli.ContainerCreate(ctx, &cn.Config{Image: "ubuntu", Tty: true}, &cn.HostConfig{},
			&nw.NetworkingConfig{}, cname)
		if err != nil {
			return nil, err
		}

		// start the docker container
		err = cli.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{})
		if err != nil {
			return nil, err
		}

		cntrIDs[idx] = cbody.ID
	}

	return cntrIDs, nil
}

// removeContainers removes all the containers given a list of containers as a
// part of clean up step.
func removeContainers(ctx context.Context, cli *client.Client, containerIDs []string) error {
	for _, cid := range containerIDs {
		err := cli.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}

	return nil
}

func TestDocker(t *testing.T) {
	ctx := context.Background()

	// Create a new docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		fmt.Printf("Failed to create a new docker client.\n Skipping tests with error: %s\n", err)
		t.Skip("skipping test; new docker client cannot be created.")
	}

	// Test if server is running, if not then skip the tests.
	_, err = cli.Ping(ctx)
	if err != nil {
		fmt.Printf("Ping to server failed.\n Skipping tests with error: %s\n", err)
		t.Skip("skipping test; docker deamon is not running")
	}

	// create and start the test containers
	var cntrNames = []string{"container_1", "container_2"}

	cntrIDs, err := createContainers(ctx, cli, cntrNames)
	if err != nil {
		fmt.Printf("Failed to create the docker containers.\n Skipping tests with error: %s\n", err)
		t.Skip("skipping tests; containers could not be created.")
	}

	Convey("Interactor implements container.Interactor", t, func() {
		var _ container.Interactor = (*Interactor)(nil)
	})

	Convey("Decode the Container stats", t, func() {
		Convey("for empty ReaderCloser stats", func() {
			emptyRC := ioutil.NopCloser(bytes.NewReader([]byte("")))
			emptyReaderCloserStats := types.ContainerStats{Body: emptyRC, OSType: "linux"}

			stats, err := decodeDockerContainerStats(emptyReaderCloserStats)
			So(stats, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})

		Convey("for non-empty ReaderCloser stats", func() {
			nonEmptyRC := ioutil.NopCloser(bytes.NewReader([]byte(testReaderCloserStats)))
			nonEmptyReaderCloserStats := types.ContainerStats{Body: nonEmptyRC, OSType: "linux"}

			stats, err := decodeDockerContainerStats(nonEmptyReaderCloserStats)
			So(stats, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
	})

	Convey("Given a Docker Operator", t, func() {
		dockerInterator := NewInteractor(cli)
		dockerOperator := container.NewOperator(dockerInterator)

		Convey("it can get the list of containers", func() {
			cntrList, err := dockerOperator.GetCurrentContainers(ctx)
			So(len(cntrList), ShouldBeGreaterThanOrEqualTo, len(cntrIDs))
			So(err, ShouldBeNil)

			Convey("it can get the stats of a container", func() {
				cntrID := cntrIDs[0]
				stats, err := dockerInterator.ContainerStats(ctx, cntrID)
				So(err, ShouldBeNil)
				So(stats, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a Docker Interator", t, func() {
		// with a nonempty client
		dockerInterator := NewInteractor(cli)

		// with an empty client
		dockerEmptyInterator := NewInteractor(&client.Client{})

		Convey("it can list the current containers", func() {
			Convey("when the docker client is nonempty", func() {
				cntList, err := dockerInterator.ContainerList(ctx)
				So(err, ShouldBeNil)
				So(len(cntList), ShouldBeGreaterThanOrEqualTo, len(cntrIDs))

				Convey("it can get the stats of a container", func() {
					Convey("for a correct container ID", func() {
						cntrID := cntrIDs[0]
						stats, err1 := dockerInterator.ContainerStats(ctx, cntrID)
						So(err1, ShouldBeNil)
						So(stats, ShouldNotBeNil)
					})

					Convey("not for a wrong container ID ", func() {
						cntrID := "wrongID"
						stats, err1 := dockerInterator.ContainerStats(ctx, cntrID)
						So(err1, ShouldNotBeNil)
						So(stats, ShouldBeNil)
					})
				})

				Convey("it can kill a container", func() {
					cntrID := cntrIDs[0]
					err = dockerInterator.ContainerKill(ctx, cntrID)
					So(err, ShouldBeNil)

					remainList, err := dockerInterator.ContainerList(ctx)
					So(err, ShouldBeNil)
					So(remainList, ShouldNotBeNil)

					// Cleanup: Remove all the dummy container
					err = removeContainers(ctx, cli, cntrIDs)
					if err != nil {
						fmt.Printf("Containers could not be removed")
					}
				})
			})

			Convey("not when the docker client is empty", func() {
				cntList, err := dockerEmptyInterator.ContainerList(ctx)
				So(err, ShouldNotBeNil)
				So(cntList, ShouldBeEmpty)
			})
		})
	})
}
