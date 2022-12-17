package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	IsProvider
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
	metaParamName            = "_meta"
	unknown                  = "__UNKNOWN__"
)

func (resrcOrDataSrc resourceOrDataSource) String() string {
	switch resrcOrDataSrc {
	case IsResource:
		return "resource"
	case IsDataSource:
		return "data source"
	case IsProvider:
		return "provider"
	case IsNestedBlock:
		return "sub block"
	}
	return unknown
}

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

type sortedTypeList []j.Type

// START sort interface
func (a sortedTypeList) Len() int {
	return len(a)
}
func (a sortedTypeList) Less(x, y int) bool {
	xTyp := a[x]
	yTyp := a[y]

	_, xIsRequired := xTyp.(j.RequiredArgType)
	_, yIsRequired := yTyp.(j.RequiredArgType)
	if xIsRequired && !yIsRequired {
		return true
	}
	if yIsRequired && !xIsRequired {
		return false
	}

	// To ensure the docsonnet attrs are sorted with the functions, we trim the #, but making sure that the # version of
	// the function will always sort first.
	xTypName := strings.TrimPrefix(xTyp.Name(), "#")
	yTypName := strings.TrimPrefix(yTyp.Name(), "#")
	if xTypName == yTypName {
		return xTyp.Name() < yTyp.Name()
	}
	return xTypName < yTypName
}

func (a sortedTypeList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// END sort interface

// paramList represents a list of parameters for a constructor function.
type paramList struct {
	// params is the list of parameters that the constructor should accept. This includes all attributes and nested
	// blocks for a given resource/data source/provider schema.
	params sortedTypeList

	// tfFieldSetters is the list of object field references that sets the resulting terraform object fields based on the
	// parameters in the params list.
	tfFieldSetters sortedTypeList

	// attrsCallArgs is the list of parameter references that can be used to forward to another constructor that takes in
	// the params.
	attrsCallArgs sortedTypeList
}

// constructorParamList returns the list of Jsonnet Type objects that should be accepted as a parameter for the
// constructor. This will iterate all attributes and nested blocks to generate a corresponding parameter name for each
// one, marking required attributes as required while default nulling all optionals.
//
// The resulting list will be sorted into two sections, with the required parameters iterated first. Within each
// section, the parameters are sorted by name.
//
// This also returns the list of field references for setting the object fields by the resulting parameter. Refer to
// paramList struct for more info.
func constructorParamList(schema *tfjson.SchemaBlock) paramList {
	params := sortedTypeList{}
	fields := sortedTypeList{}
	attrsCallArgs := sortedTypeList{}

	// Add params for the attributes
	for attr, cfg := range getInputAttributes(schema) {
		// Default all the optional params to null, which is treated the same as omitting it from the param list.
		var param j.Type = j.Null(attr)
		if cfg.attr.Required {
			param = j.Required(param)
		}
		params = append(params, param)

		fields = append(fields, j.Ref(cfg.tfName, attr))
		attrsCallArgs = append(attrsCallArgs, j.Ref(attr, attr))
	}

	// Add params for the nested blocks
	for block, cfg := range getNestedBlocks(schema) {
		// Nested blocks can not be labeled as required so always assume optional.
		params = append(params, j.Null(block))

		fields = append(fields, j.Ref(cfg.tfName, block))
		attrsCallArgs = append(attrsCallArgs, j.Ref(block, block))
	}

	sort.Sort(params)
	sort.Sort(fields)
	sort.Sort(attrsCallArgs)

	return paramList{
		params:         params,
		tfFieldSetters: fields,
		attrsCallArgs:  attrsCallArgs,
	}
}

// importCore returns the import call for importing the core library.
func importCore() j.Type {
	return j.Import("tf", "github.com/tf-libsonnet/core/main.libsonnet")
}

// improtDocsonnet returns the import call for importing the docsonnet library.
func importDocsonnet() j.Type {
	return j.Import("d", "github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet")
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

// providerNameToLibsonnetName returns the libsonnet filename for the given provider. Returns the name as
// provider_PROVIDER.libsonnet.
func providerNameToLibsonnetName(name string) string {
	return fmt.Sprintf("provider_%s.libsonnet", name)
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
