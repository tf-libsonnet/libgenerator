package gen

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
	d "github.com/jsonnet-libs/k8s/pkg/builder/docsonnet"
)

var (
	//go:embed doctmpls/constructor_docstring.md.tmpl
	constructorDocStringTmplContents string
	constructorDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(constructorDocStringTmplContents),
	)

	//go:embed doctmpls/newattrs_docstring.md.tmpl
	attrsConstructorDocStringTmplContents string
	attrsConstructorDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(attrsConstructorDocStringTmplContents),
	)

	//go:embed doctmpls/withfn_docstring.md.tmpl
	withFnDocStringTmplContents string
	withFnDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(withFnDocStringTmplContents),
	)
)

type constructorDocStringData struct {
	ProviderName string
	ObjectName   string

	ResourceOrDataSource string
	LabelParam           string
	CoreFnRef            string

	FnName    string
	FnPrefix  string
	RefPrefix string
	Params    []constructorDocStringParam

	ConstructorRef string
}

type constructorDocStringParam struct {
	Name        string
	Description string
	Typ         string
	IsOptional  bool
	IsBlock     bool

	ParamConstructorRef string // only set on blocks
}

type withFnDocStringData struct {
	AttrOrBlockName string
	ObjectName      string
	Typ             string

	FnPrefix string
	FnName   string

	ResourceOrDataSource string
	LabelParam           string

	IsArray bool
	IsMap   bool
	IsMixin bool
}

func constructorDocs(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (*j.Type, error) {
	docstr, err := constructorDocString(providerName, typ, resrcOrDataSrc, schema)
	if err != nil {
		return nil, err
	}
	doc := d.Func(
		constructorFnName,
		docstr,
		// TODO: set args
		nil,
	)
	return &doc, nil
}

func constructorDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (string, error) {
	data := getConstructorDocStringData(providerName, typ, resrcOrDataSrc, constructorFnName, "", schema)

	var out bytes.Buffer
	err := constructorDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func attrsConstructorDocs(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	fnName,
	nestedName string,
	schema *tfjson.SchemaBlock,
) (*j.Type, error) {
	docstr, err := attrsConstructorDocString(providerName, typ, resrcOrDataSrc, fnName, nestedName, schema)
	if err != nil {
		return nil, err
	}
	doc := d.Func(
		fnName,
		docstr,
		// TODO: set args
		nil,
	)
	return &doc, nil
}

func attrsConstructorDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	fnName,
	nestedName string,
	schema *tfjson.SchemaBlock,
) (string, error) {
	data := getConstructorDocStringData(providerName, typ, resrcOrDataSrc, fnName, nestedName, schema)

	var out bytes.Buffer
	err := attrsConstructorDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func withFnDocs(
	providerName, objectName string,
	resrcOrDataSrc resourceOrDataSource,
	attrOrBlockName string,
	typ string,
	collTyp collectionType,
	isMixin bool,
) (*j.Type, error) {
	fnName := fmt.Sprintf("with%s", strcase.ToCamel(attrOrBlockName))
	if isMixin {
		fnName = fnName + "Mixin"
	}

	docstr, err := withFnDocString(
		providerName, nameWithoutProvider(providerName, typ), resrcOrDataSrc,
		attrOrBlockName, fnName, typ, collTyp,
	)
	if err != nil {
		return nil, err
	}
	doc := d.Func(
		fnName,
		docstr,
		// TODO
		nil,
	)
	return &doc, nil
}

// TODO: consolidate params list
func withFnDocString(
	providerName, objectName string,
	resrcOrDataSrc resourceOrDataSource,
	attrOrBlockName string,
	fnName string,
	typ string,
	collTyp collectionType,
) (string, error) {
	data := getWithFnDocStringData(
		providerName, objectName, resrcOrDataSrc, attrOrBlockName, fnName, typ,
		collTyp == IsListOrSet, collTyp == IsMap,
	)

	var out bytes.Buffer
	err := withFnDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func getConstructorDocStringData(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	fnName,
	nestedName string,
	schema *tfjson.SchemaBlock,
) constructorDocStringData {
	objectName := nameWithoutProvider(providerName, typ)

	data := constructorDocStringData{
		ProviderName:         providerName,
		ObjectName:           objectName,
		ResourceOrDataSource: resrcOrDataSrc.String(),
		LabelParam:           resrcOrDataSrc.labelArg(),
		FnName:               fnName,
		CoreFnRef:            getCoreFnRef(resrcOrDataSrc),
		FnPrefix:             fmt.Sprintf("%s.%s", providerName, objectName),
		RefPrefix:            fmt.Sprintf("%s_%s", providerName, objectName),
		ConstructorRef:       "#fn-new",
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
		data.Params = append(data.Params, constructorDocStringParam{
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
		data.Params = append(data.Params, constructorDocStringParam{
			Name:        block,
			Description: cfg.block.Block.Description,
			Typ:         getBlockType(cfg.block.NestingMode),
			IsOptional:  true,
			IsBlock:     true,
			ParamConstructorRef: fmt.Sprintf(
				"#fn-%s%snew",
				strings.ToLower(nestedName),
				strings.ToLower(block),
			),
		})
	}
	return data
}

func getWithFnDocStringData(
	providerName, objectName string,
	resrcOrDataSrc resourceOrDataSource,
	attrOrBlockName string,
	fnName string,
	typ string,
	isArray, isMap bool,
) withFnDocStringData {
	isMixin := strings.HasSuffix(fnName, "Mixin")

	data := withFnDocStringData{
		AttrOrBlockName: attrOrBlockName,
		ObjectName:      objectName,
		Typ:             typ,
		FnPrefix: fmt.Sprintf(
			"%s.%s",
			providerName, objectName,
		),
		ResourceOrDataSource: resrcOrDataSrc.String(),
		LabelParam:           resrcOrDataSrc.labelArg(),
		FnName:               fnName,
		IsArray:              isArray,
		IsMap:                isMap,
		IsMixin:              isMixin,
	}
	return data
}
