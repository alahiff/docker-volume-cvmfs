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
	"log"
	"os"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	socketAddress     = "/run/docker/plugins/cvmfs.sock"
	defaultMountpoint = "/cvmfs"
	noRepoGi
)

var (
	cfgFile    string
	mountpoint string
	socket     string
	verbose    bool
)

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "docker-volume-cvmfs",
	Short: "A docker volume plugin for cvmfs.",
	Long: `With no command given, the default is to start the cvmfs docker 
volume plugin and listen for requests. The daemon will by default listen on 
/run/docker/plugins/cvmfs.sock, you can override this with --socket.

Type --help to get more information regarding additional commands, these exist
mostly to support kubernetes volumes via the flexvolume plugin.

More information on CVMFS available at:
	https://cernvm.cern.ch/portal/filesystem
`,
	Run: func(cmd *cobra.Command, args []string) {
		d := newCvmfsDriver(mountpoint)
		log.Printf("registering with docker\n")
		h := volume.NewHandler(d)
		log.Printf("listening on %v\n", socketAddress)
		err := h.ServeUnix("root", socketAddress)
		log.Printf("%v", err)
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(detachCmd)
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(umountCmd)

	// Command flags
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "be very verbose")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "alternative config file to use")
	rootCmd.PersistentFlags().StringVar(&mountpoint, "mountpoint", defaultMountpoint, "mountpoint to use (default /cvmfs)")
	rootCmd.PersistentFlags().StringVar(&socket, "socket", socketAddress, "location for the plugin socket")

}

// Read in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("config.yaml")                // name of config file (without extension)
	viper.AddConfigPath("$HOME/.docker-volume-cvmfs") // adding home directory as first search path
	viper.AddConfigPath("/etc/docker-volume-cvmfs")   // adding /etc as the second search path
	viper.AddConfigPath(".")                          // adding /etc as the second search path
	viper.AutomaticEnv()                              // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Printf("using config file: %v\n", viper.ConfigFileUsed())
	}
}

func main() {
	log.SetPrefix("docker-volume-cvmfs: ")
	log.SetFlags(0)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
