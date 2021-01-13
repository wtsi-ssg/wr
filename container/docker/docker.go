package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/wtsi-ssg/wr/container"
	"github.com/wtsi-ssg/wr/math/conversion"
)

type Interactor struct {
	dockerClient *client.Client
}

func NewInteractor(cl *client.Client) *Interactor {
	return &Interactor{
		dockerClient: cl,
	}
}

// ContainerList is a docker specific implementation of the container package function
// which returns the list of containers.
func (i *Interactor) ContainerList(ctx context.Context) ([]*container.Container, error) {
	containerList, err := i.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	customCntrList := make([]*container.Container, len(containerList))

	for idx, cntr := range containerList {
		customCntrList[idx] = &container.Container{ID: cntr.ID, Names: cntr.Names}
	}

	return customCntrList, nil
}

// ContainerStats is a docker specific implementation of the container package function
// which asks the docker container for the current memory usage (RSS, in MB)
// and total CPU (in seconds) usage of the container with the given id.
func (i *Interactor) ContainerStats(ctx context.Context,
	containerID string) (*container.Stats, error) {
	stats, err := i.dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}

	return decodeDockerContainerStats(stats)
}

// decodeDockerContainerStats takes type.ContainerStats and decodes it to return
// the current memory usage (RSS, in MB) and total CPU (in seconds).
func decodeDockerContainerStats(containerStats types.ContainerStats) (*container.Stats, error) {
	var ds *types.Stats

	err := json.NewDecoder(containerStats.Body).Decode(&ds)
	if err != nil {
		return nil, err
	}

	currentCustomStats := &container.Stats{}
	currentCustomStats.MemoryMB = conversion.BytesToMB(ds.MemoryStats.Stats["rss"])          // bytes to MB
	currentCustomStats.CPUSec = conversion.NanosecondsToSec(ds.CPUStats.CPUUsage.TotalUsage) // nanoseconds to seconds

	err = containerStats.Body.Close()

	return currentCustomStats, err
}

// ContainerKill is a docker specific implementation of the container package function
// which kills the container with the given id.
func (i *Interactor) ContainerKill(ctx context.Context, containerID string) error {
	return i.dockerClient.ContainerKill(ctx, containerID, "SIGKILL")
}

/*
type Interactor struct {
	dockerClient *client.Client
}

func NewInteractor(cl *client.Client) *Interactor {
	return &Interactor{
		dockerClient: cl,
	}
}

// ContainerList is a docker specific implementation of the container package function
// which returns the list of containers.
func (i *Interactor) ContainerList(ctx context.Context) ([]*container.Container, error) {
	containerList, err := i.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	customCntrList := make([]*container.Container, len(containerList))

	for idx, container := range containerList {
		customCntrList[idx].ID = container.ID
		customCntrList[idx].Names = container.Names
	}

	return customCntrList, nil
}

// ContainerStats is a docker specific implementation of the container package function
// which asks the docker container for the current memory usage (RSS, in MB)
// and total CPU (in seconds) usage of the container with the given id.
func (i *Interactor) ContainerStats(ctx context.Context,
	containerID string) (*container.Stats, error) {
	stats, err := i.dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}

	return decodeDockerContainerStats(stats)
}

// decodeDockerContainerStats takes type.ContainerStats and decodes it to return
// the current memory usage (RSS, in MB) and total CPU (in seconds).
func decodeDockerContainerStats(containerStats types.ContainerStats) (*container.Stats, error) {
	var ds *types.Stats

	err := json.NewDecoder(containerStats.Body).Decode(&ds)
	if err != nil {
		return nil, err
	}

	currentCustomStats := &container.Stats{}
	currentCustomStats.MemoryMB = conversion.BytesToMB(ds.MemoryStats.Stats["rss"])          // bytes to MB
	currentCustomStats.CPUSec = conversion.NanosecondsToSec(ds.CPUStats.CPUUsage.TotalUsage) // nanoseconds to seconds

	err = containerStats.Body.Close()

	return currentCustomStats, err
}

// ContainerKill is a docker specific implementation of the container package function
// which kills the container with the given id.
func (i *Interactor) ContainerKill(ctx context.Context, containerID string) error {
	return i.dockerClient.ContainerKill(ctx, containerID, "SIGKILL")
}
*/
