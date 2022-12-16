package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet/formatter"
	tfjson "github.com/hashicorp/terraform-json"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
	"go.uber.org/zap"
)

type collectionType uint8

const (
	IsListOrSet collectionType = iota
	IsMap
	IsNotCollection
)

func (ct collectionType) String() string {
	switch ct {
	case IsListOrSet:
		return "IsListOrSet"
	case IsMap:
		return "IsMap"
	case IsNotCollection:
		return "IsNotCollection"
	}
	return unknown
}

func getCollectionType(nestingMode tfjson.SchemaNestingMode) collectionType {
	switch nestingMode {
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		return IsListOrSet
	// TODO: nested object type should be special cased since it is a typed map
	case tfjson.SchemaNestingModeMap, tfjson.SchemaNestingModeSingle, tfjson.SchemaNestingModeGroup:
		return IsMap
	}
	panic(fmt.Errorf("Unsupported nesting mode: %s", nestingMode))
}

type resourceOrDataSource uint8

const (
	IsUnknown resourceOrDataSource = iota
	IsResource
	IsDataSource
	IsNestedBlock
)

const (
	mainLibsonnetName        = "main.libsonnet"
	constructorFnName        = "new"
	newAttrsFnName           = "newAttrs"
	resourceLabelArg         = "resourceLabel"
	resourceInjectAttrName   = "resource"
	dataSourceLabelArg       = "dataSrcLabel"
	dataSourceInjectAttrName = "data"
	unknown                  = "__UNKNOWN__"
)

func (resrcOrDataSrc resourceOrDataSource) labelArg() string {
	switch resrcOrDataSrc {
	case IsResource:
		return resourceLabelArg
	case IsDataSource:
		return dataSourceLabelArg
	}
	return unknown
}

func (resrcOrDataSrc resourceOrDataSource) injectAttrName() string {
	switch resrcOrDataSrc {
	case IsResource:
		return resourceInjectAttrName
	case IsDataSource:
		return dataSourceInjectAttrName
	}
	return unknown
}

type attribute struct {
	tfName string
	attr   *tfjson.SchemaAttribute
}

// getInputAttributes filters the schema attributes to only include those that are used as inputs. This skips:
// - the magic id field present on all Terraform blocks.
// - attributes that are read-only.
func getInputAttributes(schema *tfjson.SchemaBlock) map[string]*attribute {
	out := map[string]*attribute{}
	for name, cfg := range schema.Attributes {
		if name == "id" {
			continue
		}
		if cfg.Computed && !cfg.Optional {
			continue
		}

		out[sanitizeForRef(name)] = &attribute{
			tfName: name,
			attr:   cfg,
		}
	}
	return out
}

type block struct {
	tfName string
	block  *tfjson.SchemaBlockType
}

func getNestedBlocks(schema *tfjson.SchemaBlock) map[string]*block {
	out := map[string]*block{}
	for name, cfg := range schema.NestedBlocks {
		out[sanitizeForRef(name)] = &block{
			tfName: name,
			block:  cfg,
		}
	}
	return out
}

// sanitizeForRef sanitizes attribute names that use reserved Jsonnet words like local and import so that they don't
// cause syntax errors. If the name is a reserved Jsonnet word, this will return the name with an _ suffix.
func sanitizeForRef(name string) string {
	reserved := []string{
		"assert", "else", "error", "false", "for", "function", "if",
		"import", "importstr", "in", "local", "null", "tailstrict",
		"then", "self", "super", "true",
	}
	for _, w := range reserved {
		if name == w {
			return name + "_"
		}
	}
	return name
}

// writeDocToFile writes the given jsonnet document to a file. Note that this
// runs the document through the jsonnet-fmt prior to saving to disk.
func writeDocToFile(logger *zap.SugaredLogger, doc *j.Doc, fpath string) error {
	docStr := doc.String()
	docFmted, err := formatter.Format("", docStr, formatter.DefaultOptions())
	if err != nil {
		logger.Errorf("Error formatting %s", fpath)
		logger.Debugf("Contents:\n%s", docStr)
		return err
	}

	fdir := filepath.Dir(fpath)
	if err := os.MkdirAll(fdir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fpath, []byte(docFmted), 0644)
}

// resourceNameToLibsonnetName returns the libsonnet filename given a resource
// name. This expects a name of the form PROVIDER_RESOURCE, and returns
// the name as resource_RESOURCE.libsonnet.
// If the name already omits the provider, then provider must be set as empty
// string.
func resourceNameToLibsonnetName(provider, name string) string {
	nameWOProvider := name
	if provider != "" {
		nameWOProvider = nameWithoutProvider(provider, name)
	}
	return fmt.Sprintf("resource_%s.libsonnet", nameWOProvider)
}

// dataSourceNameToLibsonnetName returns the libsonnet filename given a data
// source name. This expects a name of the form PROVIDER_DATASRC, and returns
// the name as data_DATASRC.libsonnet.
// If the name already omits the provider, then provider must be set as empty
// string.
func dataSourceNameToLibsonnetName(provider, name string) string {
	nameWOProvider := name
	if provider != "" {
		nameWOProvider = nameWithoutProvider(provider, name)
	}
	return fmt.Sprintf("data_%s.libsonnet", nameWOProvider)
}

func nameWithoutProvider(provider, name string) string {
	return strings.TrimPrefix(name, provider+"_")
}

func ProviderNameFromAddr(addr string) (string, error) {
	providerAddr, err := tfaddr.ParseProviderSource(addr)
	if err != nil {
		return "", err
	}
	return providerAddr.Type, nil
}
