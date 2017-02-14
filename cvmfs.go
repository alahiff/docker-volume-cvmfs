package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

// TagType is the type of cvmfs repo tag (hash or tag)
type TagType int

const (
	// HASH is a hash repo tag type
	HASH TagType = iota
	// TAG is a tag repo tag type
	TAG TagType = iota
)

var cvmfsBaseConfig = []string{"/etc/cvmfs/default.conf",
	"/etc/cvmfs/default.local", "/etc/cvmfs/domain.d/cern.ch.conf"}

var cvmfsUID = 995

// Cvmfs is the main CVMFS object to manage local repositories.
type Cvmfs struct {
	mountPoint string
	m          *sync.Mutex
}

func newCvmfs(mountPoint string) Cvmfs {
	c := Cvmfs{mountPoint: mountPoint, m: &sync.Mutex{}}
	return c
}

func (c Cvmfs) volumePath(repo string, tag string) string {
	return filepath.Join(c.mountPoint, repo, tag)
}

func (c Cvmfs) cacheBase(repo string, tag string, tagType TagType) string {
	return filepath.Join("/var/cache", repo, tag)
}

// prepare sets the local config for the given repo, tag, hash combination.
// if hash is not an empty string, it is used instead of the tag even if both are given.
func (c Cvmfs) prepare(repo string, tag string, tagType TagType) (string, error) {

	cPath := filepath.Join("/etc/cvmfs", repo+"-"+tag)
	cvmfsConfig, err := os.Create(cPath)
	if err != nil {
		return "", nil
	}
	defer cvmfsConfig.Close()

	for _, f := range cvmfsBaseConfig {
		content, err := ioutil.ReadFile(f)
		if err != nil {
			return "", err
		}
		_, err = cvmfsConfig.Write(content)
		if err != nil {
			return "", err
		}
	}
	cvmfsConfig.WriteString("\n")
	cacheBase := c.cacheBase(repo, tag, tagType)
	cacheBaseShared := filepath.Join(cacheBase, "shared")
	err = os.MkdirAll(cacheBaseShared, os.ModePerm)
	if err != nil {
		return "", err
	}
	err = os.Chown(cacheBaseShared, cvmfsUID, 0)
	if err != nil {
		return "", err
	}
	cvmfsConfig.WriteString("CVMFS_CACHE_BASE=" + cacheBase + "\n")
	if tagType == HASH {
		cvmfsConfig.WriteString("CVMFS_ROOT_HASH=" + tag + "\n")
		cvmfsConfig.WriteString("CVMFS_AUTO_UPDATE=no\n")
	} else {
		cvmfsConfig.WriteString("CVMFS_REPOSITORY_TAG=" + tag + "\n")
	}
	return cPath, nil
}

func (c Cvmfs) cleanup(repo string, tag string, tagType TagType) error {

	cacheBase := c.cacheBase(repo, tag, tagType)
	return os.RemoveAll(cacheBase)
}

// Mount locally mounts the given repo, using the trunk tag.
func (c Cvmfs) Mount(repo string) (string, error) {
	return c.MountTag(repo, "trunk", HASH)
}

// MountTag locally mounts the given repo using the given repo and tag.
func (c Cvmfs) MountTag(repo string, tag string, tagType TagType) (string, error) {

	c.m.Lock()
	defer c.m.Unlock()

	volumePath := c.volumePath(repo, tag)
	repoConfig, err := c.prepare(repo, tag, tagType)
	if err != nil {
		msg := fmt.Sprintf("failed to generate config :: %v/%v :: %v", repo, tag, err)
		return "", Error{repo: repo, msg: msg}
	}

	// check if mount directory exists, create if not
	_, err = os.Lstat(volumePath)
	if err != nil && !os.IsNotExist(err) {
		msg := fmt.Sprintf("failed to check directory :: %v :: %v", volumePath, err)
		return "", Error{repo: repo, msg: msg}
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(volumePath, os.ModePerm)
		if err != nil {
			msg := fmt.Sprintf("failed to create directory :: %v :: %v", volumePath, err)
			return "", Error{repo: repo, msg: msg}
		}
	}

	// check if directory is a fuse mount, mount if not
	statfs := syscall.Statfs_t{}
	err = syscall.Statfs(volumePath, &statfs)
	if err != nil {
		msg := fmt.Sprintf("failed to check mount dir :: %v", err)
		return "", Error{repo: repo, msg: msg}
	}
	if statfs.Type != 0x65735546 {
		log.Printf("mounting %v in %v", repo, volumePath)
		cmd := exec.Command("mount", "-t", "cvmfs", "-o", "config="+repoConfig, repo, volumePath)
		result, err := cmd.CombinedOutput()
		if err != nil {
			msg := fmt.Sprintf("failed to mount :: %v :: %v", err, string(result))
			c.cleanup(repo, tag, tagType)
			return "", Error{repo: repo, msg: msg}
		}
	}

	return volumePath, nil
}

// Umount locally unmounts the given repo, using the trunk tag.
func (c Cvmfs) Umount(repo string) (string, error) {
	return c.UmountTag(repo, "trunk")
}

// UmountTag locally unmounts the given repo, using the given tag.
func (c Cvmfs) UmountTag(repo string, tag string) (string, error) {

	volumePath := c.volumePath(repo, tag)

	// check dir is a fuse mount, umount if it is
	statfs := syscall.Statfs_t{}
	err := syscall.Statfs(volumePath, &statfs)
	if err != nil {
		msg := fmt.Sprintf("failed to check unmount dir :: %v :: %v", volumePath, err)
		return "", Error{repo: repo, msg: msg}
	}

	log.Printf("unmounting %v", volumePath)
	cmd := exec.Command("umount", volumePath)
	result, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("failed to unmount :: %v :: %v", err, string(result))
		return "", Error{repo: repo, msg: msg}
	}

	// TODO(ricardo): should we cleanup the cache after a umount?

	return repo, nil
}
