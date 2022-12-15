package cmdcfg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

func init() {
	rootCmd.AddCommand(getschemaCmd)
	flags := getschemaCmd.Flags()

	addProviderAndTFVersionFlags(flags)
}

var (
	getschemaCmd = &cobra.Command{
		Use:   "getschema",
		Short: "Get the schema from Terraform providers",
		Long:  `getschema gets the resource and data source schemas from any given Terraform provider.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := parseProvidersInput(cmd)
			if err != nil {
				return err
			}

			tfV, err := parseTerraformVersion(cmd)
			if err != nil {
				return err
			}

			logC, err := parseLoggerArgs()
			if err != nil {
				return err
			}
			logger := logging.GetSugaredLogger(logC)

			ctx := context.Background()
			schema, err := tfschema.GetSchemas(logger, ctx, tfV, req)
			if err != nil {
				return err
			}

			out, err := json.Marshal(schema)
			if err != nil {
				return err
			}
			fmt.Println(string(out))

			return nil
		},
	}
)
