/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Authors: Sendu Bala <sb10@sanger.ac.uk>, Ashwini Chhipa <ac55@sanger.ac.uk>
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
	ErrContainerList       = "Could not list the containers"
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
// Refer: https://github.com/moby/moby/blob/1.13.x/client/client.go
// eg of creating in production code...
/*
	import	"github.com/docker/docker/client"
	....
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	dockerClient := &DockerClient{
		client:             cli,
		existingContainers: make(map[string]bool),
	}

	cntrList, err := dockerClient.GetCurrentContainers(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(cntrList)
*/
func NewDockerClient(docInteractor DockerInteractor) *DockerClient {
	return &DockerClient{
		client:             docInteractor,
		existingContainers: make(map[string]bool),
	}
}

// GetCurrentContainers returns current containers.
func (d *DockerClient) GetCurrentContainers(ctx context.Context) ([]types.Container, error) {
	return d.client.ContainerList(ctx, types.ContainerListOptions{})
}

// RememberCurrentContainerIDs calls GetCurrentContainerIDs() and stores the
// results, for the benefit of a future GetNewDockerContainerID() call.
func (d *DockerClient) RememberCurrentContainerIDs(ctx context.Context) error {
	containers, err := d.GetCurrentContainers(ctx)
	if err != nil {
		return &ContainerError{Type: ErrContainerList, Err: err}
	}

	for _, container := range containers {
		d.existingContainers[container.ID] = true
	}

	return nil
}

// GetNewDockerContainerIDs returns the ids of all the docker containers that now exist
// and weren't previously remembered by a RememberCurrentContainerIDs() call.
func (d *DockerClient) GetNewDockerContainerIDs(ctx context.Context) ([]string, error) {
	containers, err := d.GetCurrentContainers(ctx)
	if err != nil {
		return []string{}, &ContainerError{Type: ErrContainerList, Err: err}
	}

	newCntrsIDs := []string{}

	for _, container := range containers {
		if !d.existingContainers[container.ID] {
			newCntrsIDs = append(newCntrsIDs, container.ID)
		}
	}

	if len(newCntrsIDs) == 0 {
		return []string{}, &ContainerError{Type: ErrNoNewContainerFound}
	}

	return newCntrsIDs, nil
}

// GetNewDockerContainerIDByName returns the id of the container with given name.
// Only new containers (those not remembered by the last RememberCurrentContainerIDs() call)
// are considered when searching for a container with that name.
func (d *DockerClient) GetNewDockerContainerIDByName(ctx context.Context, name string) (string, error) {
	containers, err := d.GetCurrentContainers(ctx)
	if err != nil {
		return "", &ContainerError{Type: ErrContainerList, Err: err}
	}

	for _, container := range containers {
		if !d.existingContainers[container.ID] {
			id, err := getIDByName(name, container)

			if id != "" {
				return id, err
			}
		}
	}

	return "", &ContainerError{Type: ErrNoNewContainerFound}
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

// GetDockerContainerIDByPath returns the ID of the container from a file path
// you gave to --cidfile argument of your docker container. Docker writes the
// container ID out to this file. If you have some other process that generates a --cidfile
// name for you, your file path name can contain ? (any non-separator character)
// and * (any number of non-separator characters) wildcards.
// The first file to match the glob that also contains a valid id is the one used.
//
// In case the name is a relative file path, the second argument is the
// absolute path to the working directory where your container was created.
func (d *DockerClient) GetDockerContainerIDByPath(ctx context.Context, path string, dir string) (string, error) {
	// if name is a relative file path, then create the absolute path with dir
	cidPath := relativeToAbsolutePath(path, dir)

	_, err := os.Stat(cidPath)
	if err == nil {
		return d.cidPathToID(ctx, cidPath)
	}

	// or it might be a file path with globs
	return d.cidPathGlobToID(ctx, cidPath)
}

// cidPathToID takes the absolute path to a file that exists, reads the first
// line, and checks that it is the ID of a current container. If so, returns
// that ID.
func (d *DockerClient) cidPathToID(ctx context.Context, cidPath string) (string, error) {
	b, err := ioutil.ReadFile(cidPath)
	if err != nil {
		return "", err
	}

	id := strings.TrimSuffix(string(b), "\n")

	ok, err := d.verifyID(ctx, id)
	if ok {
		return id, nil
	}

	return "", err
}

// verifyID checks if the given id is the ID of a current container.
func (d *DockerClient) verifyID(ctx context.Context, id string) (bool, error) {
	containers, err := d.GetCurrentContainers(ctx)
	if err != nil {
		return false, &ContainerError{Type: ErrContainerList, Err: err}
	}

	for _, container := range containers {
		if container.ID == id {
			return true, nil
		}
	}

	return false, &ContainerError{Type: ErrContainerNotFound}
}

// cidPathGlobToID is like cidPathToID, but cidPath (which should not be
// relative) can contain standard glob characters such as ? and * and matching
// files will be checked until 1 contains a valid id, which gets returned.
func (d *DockerClient) cidPathGlobToID(ctx context.Context, cidGlobPath string) (string, error) {
	paths, err := filepath.Glob(cidGlobPath)
	if err != nil {
		return "", err
	}

	for _, path := range paths {
		id, err := d.cidPathToID(ctx, path)
		if err != nil {
			continue
		}

		if id != "" {
			return id, nil
		}
	}

	return "", nil
}

// ContainerStats asks docker for the current memory usage (RSS) and total CPU
// usage of the container with the given id.
// This memory uses is returned in MB and cpu usages is returned in seconds.
func (d *DockerClient) ContainerStats(ctx context.Context, containerID string) (memMB int, cpuSec int, err error) {
	stats, err := d.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return 0, 0, err
	}

	return decodeContainerStats(stats)
}

// decodeContainerStats takes type.ContainerStats and return memory and cpu stats.
// The memory stat is returned in MB and the cpu stat is returned in seconds.
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

// KillContainer kills the container with the given ID.
func (d *DockerClient) KillContainer(ctx context.Context, containerID string) error {
	return d.client.ContainerKill(ctx, containerID, "SIGKILL")
}
