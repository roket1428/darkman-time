package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/WhyNotHugo/darkman"
	"gitlab.com/WhyNotHugo/darkman/libdarkman"
)

var Version = "0.0.0-dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "darkman",
	Short:   "Query and control darkman from the command line",
	Version: Version,
}

var setCmd = &cobra.Command{
	Use:       "set",
	Short:     "Set the current mode",
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"dark", "light"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return libdarkman.SetMode(args[0])
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get and print the current mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := libdarkman.GetMode()
		if err == nil {
			fmt.Println(mode)
		}
		return err
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the current mode and print the new mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := libdarkman.ToggleMode()
		if err == nil {
			fmt.Println(mode)
		}
		return err
	},
}

func newRunCmd() *cobra.Command {
	var readyFdRaw uint
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the darkman service",
		Long: `This command starts the darkman service itself. It should only
be used by a service manager, by a  session init script or alike.

The service will run in foreground.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var readyFd *os.File
			if cmd.Flags().Changed("ready-fd") {
				readyFd = os.NewFile(uintptr(readyFdRaw), "ready-fd")
			} else {
				readyFd = nil
			}

			// Avoid showing usage if service fails to start.
			// See: https://github.com/spf13/cobra/issues/340#issuecomment-374617413
			cmd.SilenceUsage = true
			return darkman.ExecuteService(readyFd)
		},
	}
	cmd.Flags().UintVar(&readyFdRaw, "ready-fd", 0, "File descriptor for readiness notification")
	return cmd
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := darkman.ReadConfig()
		return err
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(checkCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
