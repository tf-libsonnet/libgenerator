package cmdcfg

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tf-libsonnet/libgenerator/internal/gen"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
	"github.com/tf-libsonnet/libgenerator/internal/tfschema"
)

const (
	outDirFlagName = "out"
	configFlagName = "config"
)

func init() {
	rootCmd.AddCommand(genCmd)
	flags := genCmd.Flags()

	addProviderAndTFVersionFlags(flags)
	flags.String(
		outDirFlagName,
		"./out",
		strings.TrimSpace(`
Path to the output directory where the libraries should be rendered. Each
provider will be rendered to a subdirectory of the output directory with the
provider name.
`),
	)
	flags.String(
		configFlagName,
		"",
		strings.TrimSpace("Path to a config file containing the list of libraries to render."),
	)

}

var (
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
			configFile, err := cmd.Flags().GetString(configFlagName)
			if err != nil {
				return err
			}

			var genCfg *genConfig
			if configFile == "" {
				genCfg, err = extractConfigFromProvidersInput(cmd)
			} else {
				genCfg, err = parseConfigFile(configFile)
			}
			if err != nil {
				return err
			}

			tfV, err := parseTerraformVersion(cmd)
			if err != nil {
				return err
			}

			outDir, err := cmd.Flags().GetString(outDirFlagName)
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
			schema, err := tfschema.GetSchemas(logger, ctx, tfV, genCfg.requests)
			if err != nil {
				return err
			}

			for _, entry := range genCfg.entries {
				k := entry.Provider.schemaRequest.Src
				pName := entry.Provider.schemaRequest.Name

				libRoot := filepath.Join(outDir, entry.Repo, entry.Subdir)
				if err := os.MkdirAll(libRoot, 0755); err != nil {
					return err
				}

				logger.Infof("Rendering %s library to %s", k, libRoot)
				providerSchema := schema.Schemas[k]
				renderErr := gen.RenderLibrary(logger, libRoot, pName, providerSchema)
				if renderErr != nil {
					return renderErr
				}
			}

			return nil
		},
	}
)

type genConfig struct {
	entries  []configEntry
	requests tfschema.SchemaRequestList
}

type configEntry struct {
	Repo     string          `json:"repo"`
	Subdir   string          `json:"subdir"`
	Provider *providerConfig `json:"provider"`
}

type providerConfig struct {
	Src     string `json:"src"`
	Version string `json:"version"`

	schemaRequest *tfschema.SchemaRequest
}

func parseConfigFile(config string) (*genConfig, error) {
	cfgContents, err := os.ReadFile(config)
	if err != nil {
		return nil, err
	}

	var entries []configEntry
	if err := json.Unmarshal(cfgContents, &entries); err != nil {
		return nil, err
	}

	requests := tfschema.SchemaRequestList{}
	for _, c := range entries {
		req, err := tfschema.NewSchemaRequest(c.Provider.Src, c.Provider.Version)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
		c.Provider.schemaRequest = req
	}
	cfg := &genConfig{
		entries:  entries,
		requests: requests,
	}
	return cfg, nil
}

func extractConfigFromProvidersInput(cmd *cobra.Command) (*genConfig, error) {
	requests, err := parseProvidersInput(cmd)
	if err != nil {
		return nil, err
	}

	var entries []configEntry
	for _, req := range requests {
		entries = append(entries, configEntry{
			Repo:   req.Name,
			Subdir: "",
			Provider: &providerConfig{
				Src:           req.Src,
				Version:       req.Version,
				schemaRequest: req,
			},
		})
	}
	cfg := &genConfig{
		entries:  entries,
		requests: requests,
	}
	return cfg, nil
}
