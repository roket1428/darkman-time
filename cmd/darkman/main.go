package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/integrii/flaggy"
	"gitlab.com/WhyNotHugo/darkman"
	"gitlab.com/WhyNotHugo/darkman/libdarkman"
)

var Version = "0.0.0-dev"

func NewSubcommand(name, description string) *flaggy.Subcommand {
	cmd := flaggy.NewSubcommand(name)
	cmd.Description = description

	flaggy.AttachSubcommand(cmd, 1)
	return cmd
}

func main() {
	var setVal string

	get := NewSubcommand("get", "Get and print the current mode")
	set := NewSubcommand("set", "Set the current mode")
	toggle := NewSubcommand("toggle", "Toggle the current mode")
	run := NewSubcommand("run", `Run the darkman service`)

	run.AdditionalHelpPrepend = strings.Join([]string{
		"",
		"This command starts the darkman service itself.",
		"It should only be used by a service manager, by a  session init script or alike.",
		"",
		"The service will run in foreground.",
	}, "\n")

	set.AddPositionalValue(&setVal, "mode", 1, true, "New mode (light or dark)")

	flaggy.SetName("darkman")
	flaggy.SetVersion(Version)
	flaggy.SetDescription("Query and control darkman from the command line")
	flaggy.Parse()

	var err error
	switch {
	case get.Used:
		mode, err := libdarkman.GetMode()
		if err == nil {
			fmt.Println(mode)
		}
	case set.Used:
		// SetMode validates the mode.
		err = libdarkman.SetMode(setVal)
	case toggle.Used:
		mode, err := libdarkman.ToggleMode()
		if err == nil {
			fmt.Println(mode)
		}

	case run.Used:
		err = darkman.ExecuteService()
	default:
		flaggy.ShowHelpAndExit("No command specified")
	}

	if err != nil {
		log.Fatalf(err.Error())
	}
}
