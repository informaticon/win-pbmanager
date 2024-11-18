package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/registry"
)

var (
	registerSvnDiff  bool
	registerSvnMerge bool
)

// configCmd represents the build command
var configCmd = &cobra.Command{
	Use:   "config [options]",
	Short: "Sets config values",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not retrieve absolute path of executable starting this process: %v", err)
		}

		if registerSvnMerge {
			err := installDiffMerge(`Software\TortoiseSVN\MergeTools`, exePath+` diff %base %mine %theirs %merged %fname --base-name %bname --mine-name %yname --theirs-name %tname`)
			if err != nil {
				return err
			}
		}
		if registerSvnDiff {
			err := installDiffMerge(`Software\TortoiseSVN\DiffTools`, exePath+` diff %base %mine --base-name %bname --mine-name %yname`)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVarP(&registerSvnDiff, "register-svn-diff", "d", false, "Register pbmanager as Diff tool for PBL files in TortoiseSVN")
	configCmd.Flags().BoolVarP(&registerSvnMerge, "register-svn-merge", "m", false, "Register pbmanager as Merge tool for PBL files in TortoiseSVN")
	rootCmd.AddCommand(configCmd)
}

func installDiffMerge(parentKey, commandLine string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, parentKey,
		registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("could not open registry key of TortoiseSVN: %v", err)
	}
	defer k.Close()

	err = k.SetStringValue(".pbl", commandLine)
	if err != nil {
		return fmt.Errorf("could not add new diff/merge) command: %v", err)
	}
	return nil
}
