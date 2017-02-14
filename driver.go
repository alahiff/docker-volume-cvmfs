// Copyright ©2016 CERN
//
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
)

type cvmfsCache struct {
	Path   string
	Number int
}

type cvmfsDriver struct {
	cvmfs   Cvmfs
	volumes map[string]cvmfsCache
	m       *sync.Mutex
	cache   string
}

func newCvmfsDriver(mountPoint string) cvmfsDriver {
	log.Printf("initializing driver :: mount point: %v\n", mountPoint)
	cache := make(map[string]cvmfsCache)

	cacheLocation := filepath.Join(mountPoint, "docker.cache")
	cacheFile, err := os.Open(cacheLocation)
	if err != nil {
		cacheFile, _ = os.Create(cacheLocation)
	}
	defer cacheFile.Close()
	data, err := ioutil.ReadAll(cacheFile)
	if err == nil {
		err = json.Unmarshal(data, &cache)
		if err != nil {
			log.Printf("failed to unmarshal cache data, ignoring :: %v", err)
		}
	}
	d := cvmfsDriver{
		cvmfs:   newCvmfs(mountPoint),
		volumes: cache,
		m:       &sync.Mutex{},
		cache:   cacheLocation,
	}
	// TODO: the mounts become stale when the container dies, so we need
	// to umount and remount all that was previously in cache when the
	// container is launched
	return d
}

func (d cvmfsDriver) repoTag(volume string) (string, string, string, TagType) {
	var repoTag []string
	var tagType TagType
	volumeName := volume
	if strings.Contains(volume, "#") {
		repoTag = strings.SplitN(volume, "#", 2)
		tagType = HASH
	} else if strings.Contains(volume, "@") {
		repoTag = strings.SplitN(volume, "@", 2)
		tagType = TAG
	} else {
		repoTag = []string{volume, "trunk"}
		volumeName = volume + "@trunk"
	}
	return volumeName, repoTag[0], repoTag[1], tagType
}

func (d cvmfsDriver) updateCache(volumeName string, n int) error {

	d.m.Lock()
	defer d.m.Unlock()
	cache, ok := d.volumes[volumeName]
	if ok {
		cache.Number = cache.Number + n
		d.volumes[volumeName] = cache
	}

	// flush the cache for persistency
	data, err := json.Marshal(d.volumes)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(d.cache, data, os.ModePerm)
	if err != nil {
		return err
	}

	log.Printf("%v had %v mounts, now %v", volumeName, cache.Number-n, cache.Number)
	return nil
}

func (d cvmfsDriver) Create(r volume.Request) volume.Response {
	volumeName, repo, tag, tagType := d.repoTag(r.Name)
	log.Printf("create %v :: %v :: %v", r.Name, volumeName, r)

	// mount cvmfs
	var volumePath string
	var err error
	volumePath, err = d.cvmfs.MountTag(repo, tag, tagType)
	if err != nil {
		log.Printf("failed to mount :: %v", err.Error())
		return volume.Response{Err: err.Error()}
	}

	_, ok := d.volumes[volumeName]
	if !ok {
		d.volumes[volumeName] = cvmfsCache{Path: volumePath, Number: 0}
		d.updateCache(volumeName, 0) // this still trigger a flush
		if err != nil {
			log.Printf("failed to update cache :: %v", err)
			return volume.Response{Err: err.Error()}
		}
	}
	return volume.Response{}
}

func (d cvmfsDriver) Path(r volume.Request) volume.Response {
	volumeName, _, _, _ := d.repoTag(r.Name)
	log.Printf("path %v :: %v :: %v", r.Name, volumeName, r)

	if cache, ok := d.volumes[volumeName]; ok {
		log.Printf("%v found at %v", volumeName, cache.Path)
		return volume.Response{Mountpoint: cache.Path}
	}

	msg := fmt.Sprintf("%v volume not found", volumeName)
	log.Printf(msg)
	return volume.Response{Err: msg}
}

func (d cvmfsDriver) Remove(r volume.Request) volume.Response {
	volumeName, repo, tag, _ := d.repoTag(r.Name)
	log.Printf("remove %v :: %v :: %v", r.Name, volumeName, r)

	_, err := d.cvmfs.UmountTag(repo, tag)
	if err != nil {
		log.Printf("failed to unmount :: %v", err.Error())
	}
	delete(d.volumes, volumeName)
	err = d.updateCache(volumeName, 0) // this still triggers a flush
	if err != nil {
		log.Printf("failed to update cache :: %v", err)
		return volume.Response{Err: err.Error()}
	}

	log.Printf("%v dropped from cache", volumeName)
	return volume.Response{}
}

func (d cvmfsDriver) Mount(r volume.Request) volume.Response {
	volumeName, _, _, _ := d.repoTag(r.Name)
	log.Printf("mount %v :: %v :: %v", r.Name, volumeName, r)

	// make sure we already have it available
	resp := d.Create(r)
	if resp.Err != "" {
		return resp
	}

	// update volume cache
	err := d.updateCache(volumeName, 1)
	if err != nil {
		log.Printf("failed to update cache :: %v", err)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: d.volumes[volumeName].Path}
}

func (d cvmfsDriver) Unmount(r volume.Request) volume.Response {
	volumeName, _, _, _ := d.repoTag(r.Name)
	log.Printf("unmount %v :: %v :: %v", r.Name, volumeName, r)

	// update volume cache
	err := d.updateCache(volumeName, -1)
	if err != nil {
		log.Printf("failed to update cache :: %v", err)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (d cvmfsDriver) Get(r volume.Request) volume.Response {
	volumeName, _, _, _ := d.repoTag(r.Name)
	log.Printf("get %v :: %v", r, volumeName)

	if cache, ok := d.volumes[volumeName]; ok {
		return volume.Response{Volume: &volume.Volume{Name: volumeName, Mountpoint: cache.Path}}
	}

	msg := fmt.Sprintf("volume %s does not exist", volumeName)
	log.Printf(msg)
	return volume.Response{Err: msg}
}

func (d cvmfsDriver) List(r volume.Request) volume.Response {
	log.Printf("list %v\n", r)

	volumes := []*volume.Volume{}

	for name, cache := range d.volumes {
		volumes = append(volumes, &volume.Volume{Name: name, Mountpoint: cache.Path})
	}

	return volume.Response{Volumes: volumes}
}
