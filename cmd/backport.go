package cmd

import (
	"github.com/informaticon/dev.win.base.pbmanager/internal/backport"
	"github.com/spf13/cobra"
)

// backportCmd represents the conversion back from solution to workspace
var backportCmd = &cobra.Command{
	Use:   "backport <some.pbproj> [options] ",
	Short: "performs the conversion back from PB project to target",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var absoluteProjPath string

		if len(args) == 0 {
			absoluteProjPath = ""
		} else {
			absoluteProjPath = args[0]
		}
		absoluteProjPath, err := findPbProjFilePath(basePath, absoluteProjPath)
		if err != nil {
			return err
		}
		return backport.ConvertProjectToTarget(absoluteProjPath, verbose)
	},
}

// verbose will be set to true if the user provides the --verbose or -v flag.
var verbose bool

func init() {
	rootCmd.AddCommand(backportCmd)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}
