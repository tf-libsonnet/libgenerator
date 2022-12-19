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
	//go:embed doctmpls/provider_docstring.md.tmpl
	providerDocStringTmplContents string
	providerDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(providerDocStringTmplContents),
	)

	//go:embed doctmpls/provider_constructor_docstring.md.tmpl
	providerConstructorDocStringTmplContents string
	providerConstructorDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(providerConstructorDocStringTmplContents),
	)

	//go:embed doctmpls/provider_newattrs_docstring.md.tmpl
	providerNewAttrsDocStringTmplContents string
	providerNewAttrsDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(providerNewAttrsDocStringTmplContents),
	)
)

type providerDocStringData struct {
	ProviderName string
	Description  string
}

func providerDocString(
	providerName, description string,
) (string, error) {
	data := providerDocStringData{
		ProviderName: providerName,
		Description:  description,
	}

	var out bytes.Buffer
	err := providerDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func providerConstructorDocs(providerName string, schema *tfjson.SchemaBlock) (*j.Type, error) {
	data := getProviderConstructorDocStringData(providerName, schema)
	var out bytes.Buffer
	if err := providerConstructorDocStringTmpl.Execute(&out, data); err != nil {
		return nil, err
	}

	docs := d.Func(
		constructorFnName,
		out.String(),
		// TODO
		nil,
	)
	return &docs, nil
}

func providerNewAttrsDocs(providerName string, schema *tfjson.SchemaBlock) (*j.Type, error) {
	data := getProviderConstructorDocStringData(providerName, schema)
	var out bytes.Buffer
	if err := providerNewAttrsDocStringTmpl.Execute(&out, data); err != nil {
		return nil, err
	}

	docs := d.Func(
		newAttrsFnName,
		out.String(),
		// TODO
		nil,
	)
	return &docs, nil
}

func getProviderConstructorDocStringData(providerName string, schema *tfjson.SchemaBlock) constructorDocStringData {
	data := constructorDocStringData{
		ProviderName: providerName,
		CoreFnRef:    getCoreFnRef(IsProvider),
		FnPrefix:     fmt.Sprintf("%s.provider", providerName),
		ConstructorRef: fmt.Sprintf(
			"#fn-%snew",
			strings.ToLower(strcase.ToCamel(providerName)),
		),
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
				strings.ToLower(strcase.ToCamel(providerName)),
				strings.ToLower(strcase.ToCamel(block)),
			),
		})
	}
	return data
}
