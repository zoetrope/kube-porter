package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kube-porter/pkg"
)

var statusOpts struct {
	output string
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all forwarders",
	Long:  `Show the status of all forwarders`,
	RunE: func(cmd *cobra.Command, args []string) error {

		c := pkg.NewClient(rootOpts.socket)
		err := c.Ready()
		if err != nil {
			fmt.Fprintln(os.Stderr, "kube-porter is not yet running")
			return err
		}

		switch statusOpts.output {
		case "json":
			status, err := c.Get("/status")
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get status: %v\n", err)
				return err
			}
			fmt.Fprintf(os.Stdout, status)
		case "text":
			var forwarderList []pkg.ForwarderStatus
			err = c.GetJson("/status", &forwarderList)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get status: %v\n", err)
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 1, 1, ' ', 0)
			w.Write([]byte("Type\tNamespace\tName\tPorts\tForwarding\n"))
			for _, f := range forwarderList {
				w.Write([]byte(fmt.Sprintf("%s\t%s\t%s\t%s\t%v\n", f.ObjectType, f.Namespace, f.Name, strings.Join(f.Ports, ","), f.Forwarding)))
			}
			return w.Flush()
		}

		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		isSupported := false
		for _, format := range supportedFormats {
			if format == statusOpts.output {
				isSupported = true
			}
		}
		if !isSupported {
			return fmt.Errorf("invalid format: %s", statusOpts.output)
		}
		return nil
	},
}

var supportedFormats = []string{"json", "text"}

func init() {
	rootCmd.AddCommand(statusCmd)
	fs := statusCmd.Flags()
	fs.StringVar(&statusOpts.output, "output", "text", fmt.Sprintf("Output format. One of: [%s]", strings.Join(supportedFormats, ", ")))
}
