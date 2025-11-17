package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	logging "github.com/informaticon/lib.go.base.logging"
	"github.com/informaticon/lib.go.base.logging/filter/level"
	"github.com/informaticon/lib.go.base.logging/rule"
	"github.com/informaticon/lib.go.base.logging/sender/eventlog"
	"github.com/informaticon/lib.go.base.logging/sender/std"
	"github.com/informaticon/lib.go.base.logging/transformer/pretty"
	"github.com/spf13/cobra"
)

// BuildTime must be set by the build script
var BuildTime = ""

// Version of the executable. Must be set via ldflags at build time
var Version = "0.0.0-trunk"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pbmanager",
	Short: "PowerBuilder management tools",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var logMinSeverity slog.Level
		switch flagLogLevel {
		case "debug":
			logMinSeverity = slog.LevelDebug
		case "info":
			logMinSeverity = slog.LevelInfo
		case "warn":
			logMinSeverity = slog.LevelWarn
		case "error":
			logMinSeverity = slog.LevelError
		default:
			fmt.Printf("invalid log level: %s\n", flagLogLevel)
			os.Exit(2)
		}

		slog.SetDefault(slog.New(logging.New(nil,
			rule.New().
				Transform(pretty.New()).
				Filter(level.New(level.WithMin(logMinSeverity))).
				Send(std.New()),
			rule.New().
				Filter(level.New(level.WithMin(logMinSeverity))).
				Send(eventlog.New("dev.win.base.pbmanager", eventlog.WithExternal())),
		)))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		flagV, err := cmd.Flags().GetBool("version")
		if err != nil {
			return err
		}
		if flagV {
			fmt.Println(getVersion())
			return nil
		}
		return errors.New("subcommand missing")
	},
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
	pbVersion       int
	pbRuntimeFolder string
	timeoutSeconds  uint
	serverAddr      string
	serverApiKey    string
}
var (
	basePath     string
	flagLogLevel string
)

func init() {
	b, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().IntVar(&orcaVars.pbVersion, "orca-version", 22, "PowerBuilder version to use (only 22 works atm).")
	rootCmd.PersistentFlags().StringVar(&orcaVars.pbRuntimeFolder, "orca-runtime", "", "PowerBuilder runtime folder to use (pbmanager will search the runtime folder automatically if not set).")
	rootCmd.PersistentFlags().UintVar(&orcaVars.timeoutSeconds, "orca-timeout", 7200, "Timeout (seconds) for PowerBuilder ORCA commands.")
	rootCmd.PersistentFlags().StringVar(&orcaVars.serverAddr, "orca-server", "", "Orca server address to use. If not specified, a server will be started automatically.")
	rootCmd.PersistentFlags().StringVar(&orcaVars.serverApiKey, "orca-apikey", "", "Orca server API key to use.")
	rootCmd.PersistentFlags().StringVarP(&basePath, "base-path", "b", b, "Working directory to use. Needed if you want to provide relative paths. If omitted, pbmanager will choose the current working directory as base path.")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "warn", "Minimum log level to print. [debug, info, warn, error]")
	rootCmd.Flags().Bool("version", false, "Print pbmanager version")
}

// getVersion returns a version string to describe the current axp version.
func getVersion() string {
	return fmt.Sprintf("v%s, BuildTime: %s", Version, BuildTime)
}
