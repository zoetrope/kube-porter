package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/declarative-port-forwarder/pkg"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := pkg.NewClient(rootOpts.socket)

		err := c.Ready()
		if err == nil {
			return errors.New("server is already running")
		}

		exe, err := os.Executable()
		if err != nil {
			return err
		}

		opts := []string{"serve"}
		if len(rootOpts.socket) != 0 {
			opts = append(opts, "--socket", rootOpts.socket)
		}
		if serveOpts.debug {
			opts = append(opts, "--debug")
		}
		if len(serveOpts.kubeconfig) != 0 {
			opts = append(opts, "--kubeconfig", serveOpts.kubeconfig)
		}
		if len(serveOpts.manifest) != 0 {
			opts = append(opts, "--manifest", serveOpts.manifest)
		}
		if len(serveOpts.logdir) != 0 {
			opts = append(opts, "--logdir", serveOpts.logdir)
		}

		serve := exec.Command(exe, opts...)
		if err := serve.Start(); err != nil {
			return err
		}
		fmt.Println("Starting...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return errors.New("failed to start server")
			case <-ticker.C:
				if err := c.Ready(); err == nil {
					//TODO: show log file path
					fmt.Println("Done.")
					return nil
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	AddServeFlags(startCmd.Flags())
}
