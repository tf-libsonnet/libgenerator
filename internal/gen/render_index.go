package gen

import (
	"path/filepath"
	"sort"

	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

type indexImports struct {
	providerName string
	resources    []string
	dataSources  []string
}

func renderIndex(idx indexImports) j.Doc {
	fields := sortedTypeList{}
	for _, r := range idx.resources {
		libsonnet := resourceNameToLibsonnetName(idx.providerName, r)
		fields = append(
			fields,
			j.Import(r, filepath.Join(".", libsonnet)),
		)
	}
	sort.Sort(fields)

	// Prepend the provider field after the resources are added and sorted so that it is always the first item in the
	// object.
	providerLibsonnet := providerNameToLibsonnetName(idx.providerName)
	fields = append(
		sortedTypeList{j.Import("provider", filepath.Join(".", providerLibsonnet))},
		fields...,
	)

	dataFields := sortedTypeList{}
	for _, d := range idx.dataSources {
		libsonnet := dataSourceNameToLibsonnetName(idx.providerName, d)
		dataFields = append(
			dataFields,
			j.Import(d, filepath.Join(".", libsonnet)),
		)
	}
	sort.Sort(dataFields)

	// Data sources are namespaced with the data keyword.
	fields = append(fields, j.Object("data", dataFields...))

	root := j.Object("", fields...)
	return j.Doc{
		Root: root,
	}
}
