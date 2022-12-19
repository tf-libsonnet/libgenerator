package gen

import (
	"path/filepath"
	"sort"

	j "github.com/jsonnet-libs/k8s/pkg/builder"
	d "github.com/jsonnet-libs/k8s/pkg/builder/docsonnet"
)

const (
	libRootDirName        = "_gen"
	libResourcesDirName   = "resources"
	libDataSourcesDirName = "data"
)

type indexImports struct {
	providerName string
	resources    []string
	dataSources  []string
}

func renderIndex(idx indexImports) (j.Doc, error) {
	fields := sortedTypeList{}
	for _, r := range idx.resources {
		libsonnet := nameToLibsonnetName(idx.providerName, r)
		fields = append(
			fields,
			j.Import(r, filepath.Join(".", libResourcesDirName, libsonnet)),
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

	// Import the data source index, namespaced under the data key.
	fields = append(
		fields,
		j.Import("data", filepath.Join(".", libDataSourcesDirName, "main.libsonnet")),
	)

	// Generate pkg docs and prepend to the fields list so that it is the first field.
	docstr, err := rootDocString(idx.providerName, "TODO")
	if err != nil {
		return j.Doc{}, err
	}
	doc := d.Pkg(idx.providerName, "", docstr)
	fields = append([]j.Type{doc}, fields...)

	root := j.Object("", fields...)
	return j.Doc{
		Locals: []j.LocalType{importDocsonnet()},
		Root:   root,
	}, nil
}

func renderDataIndex(idx indexImports) j.Doc {
	fields := sortedTypeList{}
	for _, data := range idx.dataSources {
		libsonnet := nameToLibsonnetName(idx.providerName, data)
		fields = append(
			fields,
			j.Import(data, filepath.Join(".", libsonnet)),
		)
	}
	sort.Sort(fields)

	// Generate pkg docs and prepend to the fields list so that it is the first field.
	// TODO
	doc := d.Pkg("data", "", "")
	fields = append([]j.Type{doc}, fields...)

	root := j.Object("", fields...)
	return j.Doc{
		Locals: []j.LocalType{importDocsonnet()},
		Root:   root,
	}
}
