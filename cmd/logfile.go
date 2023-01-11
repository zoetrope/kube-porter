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
> tail -f $(declarative-port-forwarder logfile)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := pkg.NewClient(rootOpts.socket)
		err := c.Ready()
		if err != nil {
			fmt.Fprintln(os.Stderr, "declarative-port-forwarder is not yet running")
			return err
		}
		c.Get("/logfile")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logfileCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logfileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logfileCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
