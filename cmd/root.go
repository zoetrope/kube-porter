package cmd

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kube-porter/pkg"
)

var rootOpts struct {
	socket string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kube-porter",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Version: pkg.Version,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return nil
	},
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

	var defaultSocketPath string
	if runtime.GOOS == "linux" {
		defaultSocketPath = "@kube-porter.sock"
	} else {
		defaultSocketPath = filepath.Join(homedir, ".kube-porter.sock")
	}
	fs.StringVar(&rootOpts.socket, "socket", defaultSocketPath, "")
}
