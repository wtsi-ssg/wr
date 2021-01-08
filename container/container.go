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

package container

// this file has functions for dealing with containers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	fs "github.com/wtsi-ssg/wr/fs/filepath"
	"github.com/wtsi-ssg/wr/math"
)

const (
	ErrContainerNotFound   = "Container not found"
	ErrContainerList       = "Could not list the containers"
	ErrNoNewContainerFound = "No new container found"
)

// ErrorType records a container-related error.
type ErrorType struct {
	Type string // ErrContainerNotFound or ErrContainerList or ErrNoNewContainerFound
	Err  error
}

func (et ErrorType) Error() string {
	msg := et.Type
	if et.Err != nil {
		msg += " [" + et.Err.Error() + "]"
	}

	return msg
}

type Interactor interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerKill(ctx context.Context, containerID, signal string) error
	ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
}

// Operator offers some methods for querying a container. You must use the
// NewContainerOperator() method to make one, or the methods won't work.
type Operator struct {
	client             Interactor
	existingContainers map[string]bool
}

// NewContainerOperator creates a new Operator, connecting to a container using
// the standard environment variables to define options.
// For docker refer: https://github.com/moby/moby/blob/1.13.x/client/client.go
// eg of creating a operator for docker container in production code...
/*
	import	"github.com/docker/docker/client"
	....
{
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	dockerOperator := &Operator{
		client:             cli,
		existingContainers: make(map[string]bool),
	}

	cntrList, err := dockerOperator.GetCurrentContainers(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(cntrList)
}.
*/
func NewContainerOperator(cntrInteractor Interactor) *Operator {
	return &Operator{
		client:             cntrInteractor,
		existingContainers: make(map[string]bool),
	}
}

// GetCurrentContainers returns current docker containers.
func (o *Operator) GetCurrentDockerContainers(ctx context.Context) ([]types.Container, error) {
	return o.client.ContainerList(ctx, types.ContainerListOptions{})
}

// RememberCurrentDockerContainerIDs calls GetCurrentDockerContainerIDs() and stores the
// results, for the benefit of a future GetNewDockerContainerID() call.
func (o *Operator) RememberCurrentDockerContainerIDs(ctx context.Context) error {
	containers, err := o.GetCurrentDockerContainers(ctx)
	if err != nil {
		return &ErrorType{Type: ErrContainerList, Err: err}
	}

	for _, container := range containers {
		o.existingContainers[container.ID] = true
	}

	return nil
}

// GetNewDockerContainers returns the list of all the new docker containers that now exists
// and weren't previously remembered by a RememberCurrentDockerContainerIDs() call.
func (o *Operator) GetNewDockerContainers(ctx context.Context) ([]types.Container, error) {
	containers, err := o.GetCurrentDockerContainers(ctx)
	if err != nil {
		return []types.Container{}, &ErrorType{Type: ErrContainerList, Err: err}
	}

	var newContainers []types.Container

	for _, container := range containers {
		if !o.existingContainers[container.ID] {
			newContainers = append(newContainers, container)
		}
	}

	if newContainers == nil {
		return newContainers, &ErrorType{Type: ErrNoNewContainerFound}
	}

	return newContainers, nil
}

// GetNewDockerContainerIDs returns the ids of all the docker containers that now exist
// and weren't previously remembered by a RememberCurrentDockerContainerIDs() call.
func (o *Operator) GetNewDockerContainerIDs(ctx context.Context) ([]string, error) {
	newContainers, err := o.GetNewDockerContainers(ctx)
	if err != nil {
		return []string{}, &ErrorType{Type: ErrContainerList, Err: err}
	}

	var newCntrsIDs = []string{}

	for _, container := range newContainers {
		newCntrsIDs = append(newCntrsIDs, container.ID)
	}

	return newCntrsIDs, nil
}

// GetNewDockerContainerIDByName returns the id of the docker container with given name.
// Only new containers (those not remembered by the last RememberCurrentDockerContainerIDs() call)
// are considered when searching for a container with that name.
func (o *Operator) GetNewDockerContainerIDByName(ctx context.Context, name string) (string, error) {
	newContainers, err := o.GetNewDockerContainers(ctx)
	if err != nil {
		return "", &ErrorType{Type: ErrNoNewContainerFound, Err: err}
	}

	for _, cid := range newContainers {
		id, err := getIDByName(name, cid)
		if id != "" {
			return id, err
		}
	}

	return "", &ErrorType{Type: ErrNoNewContainerFound}
}

// getIDByName returns the id of a Docker container with a given name,
// from list of names for a container.
func getIDByName(name string, container types.Container) (string, error) {
	for _, cname := range container.Names {
		cname = strings.TrimPrefix(cname, "/")
		if cname == name {
			return container.ID, nil
		}
	}

	return "", &ErrorType{Type: ErrContainerNotFound}
}

// GetDockerContainerIDByPath returns the ID of the docker container from a file path
// you gave to --cidfile argument of your docker command. Docker writes the
// container ID out to this file. If you have some other process that generates a --cidfile
// name for you, your file path name can contain ? (any non-separator character)
// and * (any number of non-separator characters) wildcards.
// The first file to match the glob that also contains a valid id is the one used.
//
// In case the name is a relative file path, the second argument is the
// absolute path to the working directory where your container was created.
func (o *Operator) GetDockerContainerIDByPath(ctx context.Context, path string, dir string) (string, error) {
	// if name is a relative file path, then create the absolute path with dir
	cidPath := fs.RelativeToAbsolutePath(path, dir)

	_, err := os.Stat(cidPath)
	if err == nil {
		return o.cidPathToID(ctx, cidPath)
	}

	// or it might be a file path with globs
	return o.cidPathGlobToID(ctx, cidPath)
}

// cidPathToID takes the absolute path to a file that exists, reads the first
// line, and checks that it is the ID of a current docker container. If so, returns
// that ID.
func (o *Operator) cidPathToID(ctx context.Context, cidPath string) (string, error) {
	b, err := ioutil.ReadFile(cidPath)
	if err != nil {
		return "", err
	}

	id := strings.TrimSuffix(string(b), "\n")

	ok, err := o.verifyID(ctx, id)
	if ok {
		return id, nil
	}

	return "", err
}

// verifyID checks if the given id is the ID of a current docker container.
func (o *Operator) verifyID(ctx context.Context, id string) (bool, error) {
	containers, err := o.GetCurrentDockerContainers(ctx)
	if err != nil {
		return false, &ErrorType{Type: ErrContainerList, Err: err}
	}

	for _, container := range containers {
		if container.ID == id {
			return true, nil
		}
	}

	return false, &ErrorType{Type: ErrContainerNotFound}
}

// cidPathGlobToID is like cidPathToID, but cidPath (which should not be
// relative) can contain standard glob characters such as ? and * and matching
// files will be checked until 1 contains a valid id, which gets returned.
func (o *Operator) cidPathGlobToID(ctx context.Context, cidGlobPath string) (string, error) {
	paths, err := filepath.Glob(cidGlobPath)
	if err != nil {
		return "", err
	}

	for _, path := range paths {
		id, err := o.cidPathToID(ctx, path)
		if err != nil {
			continue
		}

		if id != "" {
			return id, nil
		}
	}

	return "", nil
}

// ContainerStats asks the docker container for the current memory usage (RSS) and total CPU
// usage of the container with the given id.
// This memory usages is returned in MB and cpu usages is returned in seconds.
func (o *Operator) ContainerStats(ctx context.Context,
	containerID string) (memMB int, cpuSec int, err error) {
	stats, err := o.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return 0, 0, err
	}

	return decodeDockerContainerStats(stats)
}

// decodeDockerContainerStats takes type.ContainerStats and return memory and cpu stats.
// The memory stat is returned in MB and the cpu stat is returned in seconds.
func decodeDockerContainerStats(cntrStats types.ContainerStats) (int, int, error) {
	var ds *types.Stats

	err := json.NewDecoder(cntrStats.Body).Decode(&ds)
	if err != nil {
		return 0, 0, err
	}

	memMB := math.BytesToMB(ds.MemoryStats.Stats["rss"])             // bytes to MB
	cpuSec := math.NanosecondsToSec(ds.CPUStats.CPUUsage.TotalUsage) // nanoseconds to seconds

	err = cntrStats.Body.Close()

	return memMB, cpuSec, err
}

// KillContainer kills the Docker container with the given ID.
func (o *Operator) KillDockerContainer(ctx context.Context, containerID string) error {
	return o.client.ContainerKill(ctx, containerID, "SIGKILL")
}
