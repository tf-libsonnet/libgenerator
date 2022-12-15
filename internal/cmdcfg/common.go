package cmdcfg

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

const (
	providersFlagName = "provider"
	tfVersionFlagName = "tfversion"
)

func addProviderAndTFVersionFlags(flags *pflag.FlagSet) {
	flags.StringSlice(
		providersFlagName,
		[]string{},
		strings.TrimSpace(`
Provider to generate libsonnet libraries from. This should be two key-value
pairs with the keys src and version, separated by an ampersand. E.g.,
--provider 'src=aws&version=4.46.0'. Pass in multiple times for sourcing from
multiple providers.
`),
	)
	flags.String(
		tfVersionFlagName,
		"1.3.6",
		strings.TrimSpace(`
The version of Terraform to use when retrieving providers and their schema. If
there is no compatible terraform version installed on the operator machine,
libgenerator will download one from releases.hashicorp.com.
`),
	)
}

// parseProvidersInput parses the --provider arg list.
func parseProvidersInput(cmd *cobra.Command) (tfschema.SchemaRequestList, error) {
	providersInput, err := cmd.Flags().GetStringSlice(providersFlagName)
	if err != nil {
		return nil, err
	}

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

// parseTerraformVersion parses the --tfversion flag.
func parseTerraformVersion(cmd *cobra.Command) (*version.Version, error) {
	tfVersion, err := cmd.Flags().GetString(tfVersionFlagName)
	if err != nil {
		return nil, err
	}

	return version.NewVersion(tfVersion)
}
