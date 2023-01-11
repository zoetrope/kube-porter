package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoetrope/declarative-port-forwarder/pkg"
)

// logfileCmd represents the logfile command
var logfileCmd = &cobra.Command{
	Use:   "logfile",
	Short: "Print the name of log file for declarative-port-forwarder",
	Long: `Print the name of log file for declarative-port-forwarder
You can view log by passing the result of this command as follows:

$ cat $(declarative-port-forwarder logfile)

$ tail -f $(declarative-port-forwarder logfile)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := pkg.NewClient(rootOpts.socket)
		err := c.Ready()
		if err != nil {
			fmt.Fprintln(os.Stderr, "declarative-port-forwarder is not yet running")
			return err
		}
		logfile, err := c.Get("/logfile")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get logfile: %v\n", err)
			return err
		}
		fmt.Fprintf(os.Stdout, logfile)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logfileCmd)
}
