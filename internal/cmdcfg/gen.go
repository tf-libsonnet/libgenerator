package cmdcfg

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/gen"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

func init() {
	rootCmd.AddCommand(genCmd)
	flags := genCmd.Flags()

	flags.StringSliceVar(
		&providersInput,
		"provider",
		[]string{},
		strings.TrimSpace(`
Provider to generate libsonnet libraries from. This should be two key-value
pairs with the keys src and version, separated by an ampersand. E.g.,
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
	flags.StringVar(
		&outDir,
		"out",
		".",
		strings.TrimSpace(`
Path to the output directory where the libraries should be rendered. Each
provider will be rendered to a subdirectory of the output directory with the
provider name.
`),
	)

}

var (
	outDir string

	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "Generate libsonnet libraries from Terraform providers",
		Long: `gen generates libsonnet libraries for any given Terraform provider.

This command will:
- Retrieve the schema for resources and data sources from the provider.
- Generate corresponding libsonnet files from the schema.
- Write the libsonnet files to a subfolder named after the libraryName.
`,
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

			logger.Info("Retrieving schemas for providers")
			ctx := context.Background()
			schema, err := tfschema.GetSchemas(logger, ctx, v, req)
			if err != nil {
				return err
			}

			for k, cfg := range schema.Schemas {
				pName, err := gen.ProviderNameFromAddr(k)
				if err != nil {
					return err
				}
				pOut := filepath.Join(outDir, pName)
				if err := os.MkdirAll(pOut, 0755); err != nil {
					return err
				}

				logger.Infof("Rendering %s library to %s", k, pOut)
				renderErr := gen.RenderLibrary(logger, pOut, pName, cfg)
				if renderErr != nil {
					return renderErr
				}
			}

			return nil
		},
	}
)
