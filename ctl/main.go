package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/WhyNotHugo/darkman/libdarkman"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "darkmanctl",
	Short: "Query and control darkman from the command line",
}

var setCmd = &cobra.Command{
	Use:       "set",
	Short:     "Set the current mode",
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"dark", "light"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return libdarkman.SetMode(args[0])
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := libdarkman.GetMode()

		if err != nil {
			return err
		}
		fmt.Println(mode)
		return nil
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the current mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := libdarkman.ToggleMode()

		if err != nil {
			return err
		}
		fmt.Println(mode)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(toggleCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
