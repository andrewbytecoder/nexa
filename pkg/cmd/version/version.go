package version

import "github.com/spf13/cobra"

func GetVersionCmd() []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmdVersion())

	return cmds
}

// newCmdVersion returns a cobra command for fetching versions
func newCmdVersion() *cobra.Command {
	var client bool
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information",
		Long:    "Print the client and server version information for the current context.",
		Example: "Print the client and server versions for the current context kubectl version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	cmd.Flags().BoolVar(&client, "client", true, "If true, shows client version only (no server required).")
	return cmd
}
