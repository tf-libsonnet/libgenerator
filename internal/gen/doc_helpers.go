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
	//go:embed doctmpls/constructor_docstring.md.tmpl
	constructorDocStringTmplContents string
	constructorDocStringTmpl         = template.Must(
		template.New("docstring").Parse(constructorDocStringTmplContents),
	)

	//go:embed doctmpls/newattrs_docstring.md.tmpl
	attrsConstructorDocStringTmplContents string
	attrsConstructorDocStringTmpl         = template.Must(
		template.New("docstring").Parse(attrsConstructorDocStringTmplContents),
	)
)

type docStringData struct {
	ProviderName string
	ObjectName   string

	ResourceOrDataSource string
	CoreFnRef            string

	FnPrefix  string
	RefPrefix string
	Params    []docStringParam

	ConstructorRef string
}

type docStringParam struct {
	Name        string
	Description string
	Typ         string
	IsOptional  bool
	IsBlock     bool

	ParamConstructorRef string // only set on blocks
}

func constructorDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (string, error) {
	data := getDocStringData(providerName, typ, resrcOrDataSrc, schema)

	var out bytes.Buffer
	err := constructorDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func attrsConstructorDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (string, error) {
	data := getDocStringData(providerName, typ, resrcOrDataSrc, schema)

	var out bytes.Buffer
	err := attrsConstructorDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func getDocStringData(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) docStringData {
	objectName := nameWithoutProvider(providerName, typ)

	data := docStringData{
		ProviderName:         providerName,
		ObjectName:           objectName,
		ResourceOrDataSource: resrcOrDataSrc.String(),
		CoreFnRef:            getCoreFnRef(resrcOrDataSrc),
		FnPrefix:             fmt.Sprintf("%s.%s", providerName, objectName),
		RefPrefix:            fmt.Sprintf("%s_%s", providerName, objectName),
		ConstructorRef: fmt.Sprintf(
			"#fn-%snew",
			strings.ToLower(strcase.ToCamel(objectName)),
		),
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
			ParamConstructorRef: fmt.Sprintf(
				"#fn-%s%snew",
				strings.ToLower(strcase.ToCamel(objectName)),
				strings.ToLower(strcase.ToCamel(block)),
			),
		})
	}
	return data
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

func getCoreFnRef(resrcOrDataSrc resourceOrDataSource) string {
	switch resrcOrDataSrc {
	case IsResource:
		return "[tf.withResource](https://github.com/tf-libsonnet/core/tree/main/docs#fn-withresource)"
	case IsDataSource:
		return "[tf.withData](https://github.com/tf-libsonnet/core/tree/main/docs#fn-withdata)"
	case IsProvider:
		return "[tf.withProvider](https://github.com/tf-libsonnet/core/tree/main/docs#fn-withprovider)"
	}
	return ""
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
