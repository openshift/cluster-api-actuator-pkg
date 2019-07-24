package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	componentName = "test-server"
)

var (
	rootCmd = &cobra.Command{
		Use:   componentName,
		Short: "Run server",
		Long:  "",
	}
	config string
)

func init() {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		glog.Exitf("Error executing server: %v", err)
	}
}
