package gen

import (
	"bytes"
	"html/template"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
)

var (
	//go:embed doctmpls/provider_docstring.md.tmpl
	providerDocStringTmplContents string
	providerDocStringTmpl         = template.Must(
		template.New("docstring").Funcs(sprig.FuncMap()).Parse(providerDocStringTmplContents),
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
