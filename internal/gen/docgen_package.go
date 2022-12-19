package gen

import (
	"bytes"
	"html/template"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
	tfjson "github.com/hashicorp/terraform-json"
)

var (
	//go:embed doctmpls/root_docstring.md.tmpl
	rootDocStringTmplContents string
	rootDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(rootDocStringTmplContents),
	)

	//go:embed doctmpls/object_docstring.md.tmpl
	objectDocStringTmplContents string
	objectDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(objectDocStringTmplContents),
	)
)

type rootDocStringData struct {
	ProviderName   string
	ProviderDocURL string
}

type objectDocStringData struct {
	ProviderName         string
	ObjectName           string
	Description          string
	ResourceOrDataSource string
}

func rootDocString(
	providerName, providerDocURL string,
) (string, error) {
	data := rootDocStringData{
		ProviderName:   providerName,
		ProviderDocURL: providerDocURL,
	}

	var out bytes.Buffer
	err := rootDocStringTmpl.Execute(&out, data)
	return out.String(), err
}

func objectDocString(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (string, error) {
	data := objectDocStringData{
		ProviderName:         providerName,
		ObjectName:           nameWithoutProvider(providerName, typ),
		ResourceOrDataSource: resrcOrDataSrc.String(),
		Description:          schema.Description,
	}

	var out bytes.Buffer
	err := objectDocStringTmpl.Execute(&out, data)
	return out.String(), err
}
