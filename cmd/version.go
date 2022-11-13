package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = ""
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run:   versionRun,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func versionRun(cmd *cobra.Command, args []string) {
	fmt.Println(Version)
}
