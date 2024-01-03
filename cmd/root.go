package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pbmanager",
	Short: "PowerBuilder management tools",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var orcaVars struct {
	pbVersion      int
	timeoutSeconds uint
	serverAddr     string
	serverApiKey   string
}
var basePath string

func init() {
	basePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().IntVar(&orcaVars.pbVersion, "orca-version", 22, "PowerBuilder version to use (only 22 works atm).")
	rootCmd.PersistentFlags().UintVar(&orcaVars.timeoutSeconds, "orca-timeout", 7200, "Timeout (seconds) for PowerBuilder ORCA commands.")
	rootCmd.PersistentFlags().StringVar(&orcaVars.serverAddr, "orca-server", "", "Orca server address to use. If not specified, a server will be started automatically.")
	rootCmd.PersistentFlags().StringVar(&orcaVars.serverApiKey, "orca-apikey", "", "Orca server API key to use.")
	rootCmd.PersistentFlags().StringVarP(&basePath, "base-path", "b", basePath, "Working directory to use. Needed if you want to provide relative paths. If omitted, pbmanager will choose the current working directory as base path.")
}
