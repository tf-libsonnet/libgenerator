package cmdcfg

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/gen"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

func init() {
	rootCmd.AddCommand(genCmd)
	flags := genCmd.Flags()

	addProviderAndTFVersionFlags(flags)
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

			logger.Info("Retrieving schemas for providers")
			ctx := context.Background()
			schema, err := tfschema.GetSchemas(logger, ctx, tfV, req)
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
