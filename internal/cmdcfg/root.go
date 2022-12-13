package cmdcfg

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
)

func init() {
	rootPFlags := rootCmd.PersistentFlags()

	// Configuration options for the logger
	rootPFlags.String("loglevel", "info", "Logging level. Valid options: debug, info, warn, error, panic, fatal")
	rootPFlags.String("logencoding", "console", "Log message encoding. Valid options: json, console")
	rootPFlags.Bool("log-no-color", false, "Log messages without color.")
	rootPFlags.Bool("log-with-trace", false, "Log stack traces on error.")
}

var (
	// This is set on build through ldflags.
	Version string

	rootCmd = &cobra.Command{
		Use:   "libgenerator",
		Short: "generate Jsonnet libraries from Terraform schema",
		Long: `libgenerator generates Jsonnet libraries from Terraform schema.

Learn more at https://github.com/tf-libsonnet/libgenerator.`,
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("libgenerator")
			return nil
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseLoggerArgs() (*logging.Config, error) {
	pflags := rootCmd.PersistentFlags()

	lvl, err := pflags.GetString("loglevel")
	if err != nil {
		return nil, err
	}

	le, err := pflags.GetString("logencoding")
	if err != nil {
		return nil, err
	}

	lnc, err := pflags.GetBool("log-no-color")
	if err != nil {
		return nil, err
	}

	lwt, err := pflags.GetBool("log-with-trace")
	if err != nil {
		return nil, err
	}

	cfg, err := logging.NewConfig(lvl)
	if err != nil {
		return nil, err
	}
	cfg.Encoding = le
	cfg.NoColor = lnc
	cfg.WithStackTrace = lwt
	return cfg, nil
}
