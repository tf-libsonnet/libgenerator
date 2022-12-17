package gen

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	_ "embed"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	"github.com/zclconf/go-cty/cty"
)

var (
	//go:embed constructor_docstring.md.tmpl
	docStringTmplContents string

	docStringTmpl = template.Must(template.New("docstring").Parse(docStringTmplContents))
)

type docStringData struct {
	ProviderName         string
	ObjectName           string
	ResourceOrDataSource string
	FnPrefix             string
	RefPrefix            string
	Params               []docStringParam
}

type docStringParam struct {
	Name        string
	Description string
	Typ         string
	IsOptional  bool
	IsBlock     bool

	ConstructorRef string // only set on blocks
}

func constructorDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (string, error) {
	objectName := nameWithoutProvider(providerName, typ)

	data := docStringData{
		ProviderName:         providerName,
		ObjectName:           objectName,
		ResourceOrDataSource: resrcOrDataSrc.String(),
		FnPrefix:             fmt.Sprintf("%s.%s", providerName, objectName),
		RefPrefix:            fmt.Sprintf("%s_%s", providerName, objectName),
	}
	if resrcOrDataSrc == IsDataSource {
		data.FnPrefix = fmt.Sprintf("%s.data.%s", providerName, objectName)
		data.RefPrefix = fmt.Sprintf("data_%s_%s", providerName, objectName)
	}

	attrMap := getInputAttributes(schema)
	attrs := []string{}
	for attr := range attrMap {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)
	for _, attr := range attrs {
		cfg := attrMap[attr]
		data.Params = append(data.Params, docStringParam{
			Name:        attr,
			Description: cfg.attr.Description,
			Typ:         getAttrType(cfg.attr),
			IsOptional:  cfg.attr.Optional,
		})
	}

	blockMap := getNestedBlocks(schema)
	blocks := []string{}
	for block := range blockMap {
		blocks = append(blocks, block)
	}
	sort.Strings(blocks)
	for _, block := range blocks {
		cfg := blockMap[block]
		data.Params = append(data.Params, docStringParam{
			Name:        block,
			Description: cfg.block.Block.Description,
			Typ:         getBlockType(cfg.block.NestingMode),
			IsOptional:  true,
			IsBlock:     true,
			ConstructorRef: fmt.Sprintf(
				"#fn-%s%snew",
				strings.ToLower(strcase.ToCamel(objectName)),
				strings.ToLower(strcase.ToCamel(block)),
			),
		})
	}

	var out bytes.Buffer
	err := docStringTmpl.Execute(&out, data)
	return out.String(), err
}

func getAttrType(attr *tfjson.SchemaAttribute) string {
	if attr.AttributeNestedType != nil {
		return getBlockType(attr.AttributeNestedType.NestingMode)
	}

	switch {
	case attr.AttributeType.IsObjectType(), attr.AttributeType.IsMapType():
		return "obj"
	case attr.AttributeType.IsCollectionType():
		return "list"
	case attr.AttributeType == cty.Number:
		return "number"
	case attr.AttributeType == cty.String:
		return "string"
	case attr.AttributeType == cty.Bool:
		return "bool"
	}
	return "any"
}

func getBlockType(nestingMode tfjson.SchemaNestingMode) string {
	switch nestingMode {
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		return "list[obj]"
	case tfjson.SchemaNestingModeMap:
		return "map[str, obj]"
	}
	return "obj"
}
