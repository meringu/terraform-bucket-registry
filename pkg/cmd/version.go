package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string
var Commit string

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of terraform-bucket-registry",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Terraform Bucket Registry %s -- %s\n", Version, Commit)
	},
}
