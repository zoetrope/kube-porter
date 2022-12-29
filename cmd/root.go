package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootOpts struct {
	socket string
	debug  bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "declarative-port-forwarder",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.PersistentFlags()

	//TODO: move to PreRun
	homedir, err := os.UserHomeDir()
	cobra.CheckErr(err)
	defaultSocketPath := filepath.Join(homedir, ".declarative-port-forwarder.sock")
	fs.StringVar(&rootOpts.socket, "socket", defaultSocketPath, "")
	fs.BoolVar(&rootOpts.debug, "debug", false, "Enable debug logging")
}
