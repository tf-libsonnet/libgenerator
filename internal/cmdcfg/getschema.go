package cmdcfg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

func init() {
	rootCmd.AddCommand(getschemaCmd)
	flags := getschemaCmd.Flags()

	flags.StringSliceVar(
		&providersInput,
		"provider",
		[]string{},
		strings.TrimSpace(`
Provider to retrieve schema from. This should be two key-value pairs with the
keys src and version, separated by an ampersand. E.g.,
--provider 'src=aws&version=4.46.0'. Pass in multiple times for sourcing from
multiple providers.
`),
	)
	flags.StringVar(
		&tfVersion,
		"tfversion",
		"1.3.6",
		strings.TrimSpace(`
The version of Terraform to use when retrieving providers and their schema. If
there is no compatible terraform version installed on the operator machine,
libgenerator will download one from releases.hashicorp.com.
`),
	)
}

var (
	providersInput []string
	tfVersion      string

	getschemaCmd = &cobra.Command{
		Use:   "getschema",
		Short: "Get the schema from Terraform providers",
		Long:  `getschema gets the resource and data source schemas from any given Terraform provider.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := parseProvidersInput()
			if err != nil {
				return err
			}

			logC, err := parseLoggerArgs()
			if err != nil {
				return err
			}
			logger := logging.GetSugaredLogger(logC)

			v, err := version.NewVersion(tfVersion)
			if err != nil {
				return err
			}

			ctx := context.Background()
			schema, err := tfschema.GetSchemas(logger, ctx, v, req)
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

// parseProvidersInput parses the --provider arg list.
func parseProvidersInput() (tfschema.SchemaRequestList, error) {
	out := make(tfschema.SchemaRequestList, 0, len(providersInput))

	for _, pin := range providersInput {
		// Expect an ampersand separated kv list, which is the same as query parameter encoding.
		pinKV, err := url.ParseQuery(pin)
		if err != nil {
			return nil, err
		}
		src := pinKV.Get("src")
		if src == "" {
			return nil, fmt.Errorf("src key is required for --provider")
		}
		version := pinKV.Get("version")
		if version == "" {
			return nil, fmt.Errorf("version key is required for --provider")
		}
		req, err := tfschema.NewSchemaRequest(src, version)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}

	return out, nil
}
