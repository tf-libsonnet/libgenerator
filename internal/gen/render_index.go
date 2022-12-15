package gen

import (
	"path/filepath"

	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

type indexImports struct {
	resources   []string
	dataSources []string
}

func renderIndex(idx indexImports) j.Doc {
	fields := make([]j.Type, 0, len(idx.resources)+1)
	for _, r := range idx.resources {
		libsonnet := resourceNameToLibsonnetName("", r)
		fields = append(
			fields,
			j.Import(r, filepath.Join(".", libsonnet)),
		)
	}

	dataFields := make([]j.Type, 0, len(idx.dataSources))
	for _, d := range idx.dataSources {
		libsonnet := dataSourceNameToLibsonnetName("", d)
		dataFields = append(
			dataFields,
			j.Import(d, filepath.Join(".", libsonnet)),
		)
	}
	// Data sources are namespaced with the data keyword.
	fields = append(fields, j.Object("data", dataFields...))

	root := j.Object("", fields...)
	return j.Doc{
		Root: root,
	}
}
