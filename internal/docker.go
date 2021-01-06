package internal

// this file has functions for dealing with docker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
)

const (
	ErrContainerNotFound   = "Container not found"
	ErrContainerListEmpty  = "Container list is empty"
	ErrNoNewContainerFound = "No new container found"
)

// ContainerError records a container-related error.
type ContainerError struct {
	Type string // ErrContainerNotFound or ErrContainerListEmpty
	Err  error  // In the case of ErrContainerNotFound, Container not found
}

func (c ContainerError) Error() string {
	msg := c.Type
	if c.Err != nil {
		msg += " [" + c.Err.Error() + "]"
	}

	return msg
}

type DockerInteractor interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerKill(ctx context.Context, containerID, signal string) error
	ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
}

// DockerClient offers some methods for querying docker. You must use the
// NewDockerClient() method to make one, or the methods won't work.
type DockerClient struct {
	client             DockerInteractor
	existingContainers map[string]bool
}

// NewDockerClient creates a new DockerClient, connecting to docker using
// the standard environment variables to define options.
//
// eg of creating.....
// client, err := client.NewEnvClient()
// Refer: https://github.com/moby/moby/blob/1.13.x/client/client.go.
func NewDockerClient(docInteractor DockerInteractor) *DockerClient {
	return &DockerClient{
		client:             docInteractor,
		existingContainers: make(map[string]bool),
	}
}

// GetCurrentContainers returns current containers.
func (d *DockerClient) GetCurrentContainers() ([]types.Container, error) {
	return d.client.ContainerList(context.Background(), types.ContainerListOptions{})
}

// RememberCurrentContainerIDs calls GetCurrentContainerIDs() and stores the
// results, for the benefit of a future GetNewDockerContainerID() call.
func (d *DockerClient) RememberCurrentContainerIDs() error {
	containers, err := d.GetCurrentContainers()
	if err != nil {
		return &ContainerError{Type: ErrContainerListEmpty, Err: err}
	}

	for _, container := range containers {
		d.existingContainers[container.ID] = true
	}

	return nil
}

// GetNewDockerContainerID returns the first docker container ID that now exists
// that wasn't previously remembered by a RememberCurrentContainerIDs() call.
func (d *DockerClient) GetNewDockerContainerID() (string, error) {
	containers, err := d.GetCurrentContainers()
	if err != nil {
		return "", &ContainerError{Type: ErrContainerListEmpty}
	}

	for _, container := range containers {
		if !d.existingContainers[container.ID] {
			return container.ID, nil
		}
	}

	return "", &ContainerError{Type: ErrNoNewContainerFound}
}

// verifyID checks if the given id is the ID of a current container.
func (d *DockerClient) verifyID(id string) (bool, error) {
	containers, err := d.GetCurrentContainers()
	if err != nil {
		return false, &ContainerError{Type: ErrContainerListEmpty}
	}

	for _, container := range containers {
		if container.ID == id {
			return true, nil
		}
	}

	return false, &ContainerError{Type: ErrContainerNotFound}
}

// getIDByName returns the id of a container with a given name,
// from list of names for a container.
func getIDByName(name string, container types.Container) (string, error) {
	for _, cname := range container.Names {
		cname = strings.TrimPrefix(cname, "/")
		if cname == name {
			return container.ID, nil
		}
	}

	return "", &ContainerError{Type: ErrContainerNotFound}
}

// GetNewDockerContainerIDByName returns id of the container with given name
// only new containers (those not remembered by the last RememberCurrentContainerIDs() call)
// are considered when searching for a container with that name.
func (d *DockerClient) GetNewDockerContainerIDByName(name string) (string, error) {
	containers, err := d.GetCurrentContainers()
	if err != nil {
		return "", &ContainerError{Type: ErrContainerListEmpty}
	}

	for _, container := range containers {
		if !d.existingContainers[container.ID] {
			return getIDByName(name, container)
		}
	}

	return "", &ContainerError{Type: ErrNoNewContainerFound}
}

// cidPathToID takes the absolute path to a file that exists, reads the first
// line, and checks that it is the ID of a current container. If so, returns
// that ID.
func (d *DockerClient) cidPathToID(cidPath string) (string, error) {
	b, err := ioutil.ReadFile(cidPath)
	if err != nil {
		return "", err
	}

	id := strings.TrimSuffix(string(b), "\n")

	ok, err := d.verifyID(id)
	if ok {
		return id, nil
	}

	return "", err
}

// cidPathGlobToID is like cidPathToID, but cidPath (which should not be
// relative) can contain standard glob characters such as ? and * and matching
// files will be checked until 1 contains a valid id, which gets returned.
func (d *DockerClient) cidPathGlobToID(cidGlobPath string) (string, error) {
	paths, err := filepath.Glob(cidGlobPath)
	if err != nil {
		return "", err
	}

	for _, path := range paths {
		id, err := d.cidPathToID(path)
		if err != nil {
			return "", err
		}

		if id != "" {
			return id, nil
		}
	}

	return "", nil
}

// getContainerIDByPath returns the container id from a file given it's path.
func getContainerIDByPath(d *DockerClient, path string) (string, error) {
	id, err := d.cidPathToID(path)
	if id != "" {
		return id, nil
	}

	return "", err
}

// GetDockerContainerIDByPath returns the ID of the container from a file path
// you gave to --cidfile. If you have some // other process that generates a --cidfile
// name for you, your file path name
// can contain ? (any non-separator character) and * (any number of
// non-separator characters) wildcards. The first file to match the glob that
// also contains a valid id is the one used.
//
// In the case that name is relative file path, the second argument is the
// absolute path to the working directory where your container was created.
func (d *DockerClient) GetDockerContainerIDByPath(path string, dir string) (string, error) {
	// if name is a relative file path, then create the absolute path with dir
	cidPath := path
	if !strings.HasPrefix(cidPath, "/") {
		cidPath = filepath.Join(dir, cidPath)
	}

	_, err := os.Stat(cidPath)
	if err == nil {
		return getContainerIDByPath(d, cidPath)
	}

	// or it might be a file path with globs
	return d.cidPathGlobToID(cidPath)
}

// nanosecondsToSec converts nanoseconds to sec for cpu stats.
func nanosecondsToSec(tm uint64) int {
	var divisor uint64 = 1000000000

	return int(tm / divisor)
}

// bytesToMB converts bytes to MB for memory stats.
func bytesToMB(bt uint64) int {
	var divisor uint64 = 1024

	return int(bt / divisor / divisor)
}

// decodeContainerStats takes type.ContainerStats and return mem and cpu stats.
func decodeContainerStats(cntrStats types.ContainerStats) (int, int, error) {
	var ds *types.Stats

	err := json.NewDecoder(cntrStats.Body).Decode(&ds)
	if err != nil {
		return 0, 0, err
	}

	memMB := bytesToMB(ds.MemoryStats.Stats["rss"])             // bytes to MB
	cpuSec := nanosecondsToSec(ds.CPUStats.CPUUsage.TotalUsage) // nanoseconds to seconds

	err = cntrStats.Body.Close()

	return memMB, cpuSec, err
}

// ContainerStats asks docker for the current memory usage (RSS) and total CPU
// usage of the container with the given id.
func (d *DockerClient) ContainerStats(containerID string) (memMB int, cpuSec int, err error) {
	stats, err := d.client.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		return 0, 0, err
	}

	return decodeContainerStats(stats)
}

// KillContainer kills the container with the given ID.
func (d *DockerClient) KillContainer(containerID string) error {
	return d.client.ContainerKill(context.Background(), containerID, "SIGKILL")
}
