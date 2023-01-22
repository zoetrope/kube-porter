package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kube-porter/pkg"
)

// logfileCmd represents the logfile command
var logfileCmd = &cobra.Command{
	Use:   "logfile",
	Short: "Print the name of log file for kube-porter",
	Long: `Print the name of log file for kube-porter
You can view log by passing the result of this command as follows:

$ cat $(kube-porter logfile)

$ tail -f $(kube-porter logfile)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := pkg.NewClient(rootOpts.socket)
		err := c.Ready()
		if err != nil {
			fmt.Fprintln(os.Stderr, "kube-porter is not yet running")
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
