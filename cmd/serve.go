package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zoetrope/declarative-port-forwarder/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/util/homedir"
)

var serveOpts struct {
	config     string
	kubeconfig string
	logdir     string
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		//TODO: read from KUBECONFIG environment variable
		//TODO: create log dir if not exist

		var cfg zap.Config
		if rootOpts.debug {
			cfg = zap.NewDevelopmentConfig()
		} else {
			cfg = zap.NewProductionConfig()
		}
		pid := os.Getpid()
		cfg.OutputPaths = []string{
			filepath.Join(serveOpts.logdir, fmt.Sprintf("server-%d.log", pid)),
		}
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, err := cfg.Build()
		if err != nil {
			return err
		}
		zap.ReplaceGlobals(logger)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		s := pkg.NewServer(rootOpts.socket, serveOpts.kubeconfig, serveOpts.config)
		return s.Run()
	},
}

func AddServeFlags(fs *pflag.FlagSet) {
	fs.StringVar(&serveOpts.config, "config", "", "path to the config file")
	var defaultKubeconfig = ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	}
	fs.StringVar(&serveOpts.kubeconfig, "kubeconfig", defaultKubeconfig, "path to the kubeconfig file")
	fs.StringVar(&serveOpts.logdir, "logdir", filepath.Join(os.TempDir(), "dpf"), "")
}

func init() {
	rootCmd.AddCommand(serveCmd)

	fs := serveCmd.Flags()
	AddServeFlags(fs)
}
