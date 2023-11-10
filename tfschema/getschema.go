package tfschema

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/google/go-jsonnet"
	version "github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"go.uber.org/zap"
)

const (
	providersTFJsonnetName = "providers.tf.jsonnet"
	providersTFJSONName    = "providers.tf.json"
)

var (
	//go:embed providers.tf.jsonnet
	providersTFJsonnet string
)

// SchemaRequestList represets a request to retrieve schemas for multiple Terraform providers.
type SchemaRequestList []*SchemaRequest

// SchemaRequest represents a request to retrieve schemas for a single Terraform provider.
type SchemaRequest struct {
	// name represents the provider name.
	Name string `json:"name"`

	// src represents the provider src for retrieving the schema. For example, hashicorp/aws or DopplerHQ/doppler.
	Src string `json:"src"`

	// version represents a version constraint to use for retrieving the specific provider version to use when retrieving
	// the schema information.
	Version string `json:"version"`
}

// NewSchemaRequest constructs a new schema request given a canonical provider source string (e.g., aws or
// DopplerHQ/doppler) and version constraint.
func NewSchemaRequest(provider, version string) (*SchemaRequest, error) {
	pAddr, err := tfaddr.ParseProviderSource(provider)
	if err != nil {
		return nil, err
	}

	if pAddr.Namespace == tfaddr.UnknownProviderNamespace {
		pAddr.Namespace = "hashicorp"
	}

	return &SchemaRequest{
		Name:    pAddr.Type,
		Src:     pAddr.String(),
		Version: version,
	}, nil
}

// GetSchemas returns the resource and data source schemas for the requested providers using the Terraform binary and
// the `providers get` command.
//
// This function will look for a Terraform binary that matches the given requested version on the local machine. If it
// cannot find one in the machine PATH, then this will install a new one in a temporary directory that is removed later.
//
// The providers are pulled down using `terraform init` against a basic Terraform module that only lists all the
// requested providers as required_providers. We use Jsonnet to render this basic Terraform module given the schema
// request. The module is rendered into a temporary directory that is cleaned up at the end of the function.
func GetSchemas(
	logger *zap.SugaredLogger,
	ctx context.Context,
	tfVersion *version.Version,
	req SchemaRequestList,
) (out *tfjson.ProviderSchemas, returnErr error) {
	// Ensure Terraform binary is available.
	inst := install.NewInstaller()
	// Use an anon function so we handle the error for inst.Remove
	defer func() {
		if err := inst.Remove(ctx); err != nil {
			logger.Errorf("Error removing installed Terraform files: %s", err)

			// Bubble remove error to the return error if an error hasn't been reported yet.
			if returnErr == nil {
				returnErr = err
			}
		}
	}()

	logger.Debugf("Finding or installing terraform version %s", tfVersion)
	tfPath, err := inst.Ensure(ctx, []src.Source{
		&fs.ExactVersion{
			Product: product.Terraform,
			Version: tfVersion,
		},
		&releases.ExactVersion{
			Product: product.Terraform,
			Version: tfVersion,
		},
	})
	if err != nil {
		return nil, err
	}
	logger.Debugf("Using terraform binary %s", tfPath)

	// Create a temporary directory to use as a workspace
	tmpDir, err := os.MkdirTemp("", "libgenerator-tf-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	logger.Debugf("Using working directory %s", tmpDir)

	// Render the providers.tf.json into the working dir
	renderErr := renderProvidersTFJSON(ctx, tmpDir, req)
	if renderErr != nil {
		return nil, renderErr
	}

	logger.Debug("Rendered providers.tf.json:")
	data, err := os.ReadFile(filepath.Join(tmpDir, providersTFJSONName))
	if err != nil {
		return nil, err
	}
	logger.Debug(string(data))

	// Download the providers and extract the schemas
	tf, err := tfexec.NewTerraform(tmpDir, tfPath)
	if err != nil {
		return nil, err
	}
	logger.Debug("Running terraform init")
	initErr := tf.Init(ctx)
	if initErr != nil {
		return nil, initErr
	}

	logger.Debug("Running terraform providers schema")
	return tf.ProvidersSchema(ctx)
}

// renderProvidersTFJSON runs Jsonnet against the builtin providers.tf.jsonnet code to render a providers.tf.json file
// that contains a required_providers block with all the providers we need for extracting out the schema.
func renderProvidersTFJSON(ctx context.Context, wd string, req SchemaRequestList) error {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}

	vm := jsonnet.MakeVM()
	vm.TLACode("providers", string(reqJSON))
	rendered, err := vm.EvaluateAnonymousSnippet(providersTFJsonnetName, providersTFJsonnet)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(wd, providersTFJSONName), []byte(rendered), 0644)
}
