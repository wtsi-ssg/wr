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
	"bytes"
	"context"
	"io/ioutil"

	"github.com/docker/docker/api/types"
)

const (
	DockerReaderCloserStats = `{
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

	ErrContainerNotFound   = "Container not found"
	ErrContainerList       = "Could not list the containers"
	ErrNoNewContainerFound = "No new container found"
)

// ErrorType records a container-related error.
type ErrorType struct {
	Type string // ErrContainerNotFound or ErrContainerListEmpty
	Err  error  // In the case of ErrContainerNotFound, Container not found
}

func (et ErrorType) Error() string {
	msg := et.Type

	return msg
}

// Interactor represents a mock implementation of container.Interactor.
type Interactor struct {
	ContainerListFn       func() ([]types.Container, error)
	ContainerListInvoked  int
	ContainerKillFn       func(string) error
	ContainerKillInvoked  int
	ContainerStatsFn      func(string) (types.ContainerStats, error)
	ContainerStatsInvoked int
}

// ContainerList is a mock function which returns the list of containers.
func (c *Interactor) ContainerList(ctx context.Context,
	options types.ContainerListOptions) ([]types.Container, error) {
	c.ContainerListInvoked++

	return c.ContainerListFn()
}

// ContainerKill is a mock function which kills (removes entry of) a container
// given its ID.
func (c *Interactor) ContainerKill(ctx context.Context, containerID, signal string) error {
	c.ContainerKillInvoked++

	return c.ContainerKillFn(containerID)
}

// ContainerStats is a mock function which returns the mem and cpu stats of a container
// given its ID.
func (c *Interactor) ContainerStats(ctx context.Context,
	containerID string, stream bool) (types.ContainerStats, error) {
	c.ContainerStatsInvoked++

	return c.ContainerStatsFn(containerID)
}

// RemoveContainer removes the container from container list if found
// and returns the remaining containers.
func RemoveContainer(cntrList []types.Container, containerID string) []types.Container {
	remainingCntr := []types.Container{}

	for _, cntr := range cntrList {
		if cntr.ID != containerID {
			remainingCntr = append(remainingCntr, cntr)
		}
	}

	return remainingCntr
}

// GetContainerStats calls CreateContainerStats to creates the dummy stats for of a container
// when the container id is found in the given list of containers.
func GetContainerStats(cntrList []types.Container, containerID string) (types.ContainerStats, error) {
	for _, cntr := range cntrList {
		if cntr.ID == containerID {
			return CreateContainerStats(), nil
		}
	}

	return types.ContainerStats{}, &ErrorType{Type: ErrContainerNotFound, Err: nil}
}

// createContainerStats creates and returns the dummy stats of a container.
func CreateContainerStats() types.ContainerStats {
	readerVar := ioutil.NopCloser(bytes.NewReader([]byte(DockerReaderCloserStats)))
	stats := types.ContainerStats{Body: readerVar, OSType: "linux"}

	return stats
}
