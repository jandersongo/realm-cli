package app

import (
	"fmt"

	"github.com/10gen/realm-cli/internal/cli"
	"github.com/10gen/realm-cli/internal/cli/user"
	"github.com/10gen/realm-cli/internal/terminal"

	"github.com/spf13/pflag"
)

// CommandList is the `app list` command
type CommandList struct {
	inputs cli.ProjectInputs
}

// Flags is the command flags
func (cmd *CommandList) Flags(fs *pflag.FlagSet) {
	cmd.inputs.Flags(fs)
}

// Handler is the command handler
func (cmd *CommandList) Handler(profile *user.Profile, ui terminal.UI, clients cli.Clients) error {
	apps, err := clients.Realm.FindApps(cmd.inputs.Filter())
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		ui.Print(terminal.NewTextLog("No available apps to show"))
		return nil
	}

	rows := make([]interface{}, 0, len(apps))
	for _, app := range apps {
		rows = append(rows, app.Option())
	}
	ui.Print(terminal.NewListLog(fmt.Sprintf("Found %d apps", len(rows)), rows...))
	return nil
}
