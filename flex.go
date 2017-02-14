package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// KubeOptions has the options to be passed to the kube pod.
type KubeOptions struct {
	Mountpoint string `json:"mountpoint"`
	Repository string `json:"repository"`
}

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "init the cvmfs setup",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("init\n")
	},
}

var attachCmd = &cobra.Command{
	Use:           "attach",
	Short:         "attach the cvmfs setup",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return flexError(RepoNotFoundError{})
		}
		var options KubeOptions
		err := json.Unmarshal([]byte(args[0]), &options)
		if err != nil {
			return flexError(err)
		}
		cvmfs := newCvmfs(defaultMountpoint)
		volumePath, err := cvmfs.Mount(options.Repository)
		if err != nil {
			return flexError(err)
		}
		flexSuccess(volumePath)
		return nil
	},
}

var detachCmd = &cobra.Command{
	Use:           "detach",
	Short:         "detach the cvmfs setup",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
		}
		var options KubeOptions
		err := json.Unmarshal([]byte(args[0]), &options)
		if err != nil {
			return flexError(err)
		}
		cvmfs := newCvmfs(defaultMountpoint)
		volumePath, err := cvmfs.Umount(args[0])
		if err != nil {
			return flexError(err)
		}
		flexSuccess(volumePath)
		return nil
	},
}

var mountCmd = &cobra.Command{
	Use:           "mount [repository]",
	Short:         "mount the given cvmfs repository",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			err := RepoNotFoundError{}
			return flexError(err)
		}
		err := bindMount(args[1], args[0])
		if err != nil {
			return flexError(fmt.Errorf("bind mount failed :: %v", err))
		}
		flexSuccess("")
		return nil
	},
}

var umountCmd = &cobra.Command{
	Use:           "unmount [repository]",
	Short:         "unmount the given cvmfs repository",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return flexError(RepoNotFoundError{})
		}

		cvmfs := newCvmfs(defaultMountpoint)
		_, err := cvmfs.Umount(args[0])
		if err != nil {
			return flexError(err)
		}
		flexSuccess("")
		return nil
	},
}

func bindMount(src string, dest string) error {

	// check if mount directory exists, create if not
	_, err := os.Lstat(dest)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory :: %v :: %v", dest, err)
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(dest, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory :: %v :: %v", dest, err)
		}
	}

	// bind mount src into dest
	cmd := exec.Command("mount", "--bind", src, dest)
	result, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount :: %v :: %v", err, string(result))
	}

	return nil
}

func flexError(err error) error {
	fmt.Printf("{\"status\": \"Failure\", \"message\": \"%v\"}", err)
	return err
}

func flexSuccess(volumePath string) {
	if volumePath != "" {
		fmt.Printf("{\"status\": \"Success\", \"device\": \"%v\"}", volumePath)
	} else {
		fmt.Printf("{\"status\": \"Success\"}")
	}
}
