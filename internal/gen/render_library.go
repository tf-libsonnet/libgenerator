package gen

import (
	"path/filepath"

	tfjson "github.com/hashicorp/terraform-json"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
	"go.uber.org/zap"
)

// RenderLibrary renders a full provider schema as a libsonnet library. The libsonnet library has the following
// folderstructure:
//
// - `main.libsonnet`: The root index file containing all the definitions.
// - `_gen`: Folder containing all the autogenerated files.
// - `_gen/main.libsonnet`: The index file containing all the autogenerated definitions.
// - `_gen/resource_RESOURCE.libsonnet`: A resource object file containing definitions for constructing the given
// resource block.
// - `_gen/data_DATASRC.libsonnet`: A data source object file containing definitions for constructing the given
// data source block.
func RenderLibrary(
	logger *zap.SugaredLogger,
	outDir string,
	providerName string,
	schema *tfjson.ProviderSchema,
) error {
	libraryFPath := filepath.Join(outDir, libRootDirName)
	resourcesFPath := filepath.Join(libraryFPath, libResourcesDirName)
	dataSourcesFPath := filepath.Join(libraryFPath, libDataSourcesDirName)
	idx := indexImports{
		providerName: providerName,
	}

	logger.Info("Rendering provider config generator")
	doc, err := renderProvider(providerName, schema.ConfigSchema.Block)
	if err != nil {
		return err
	}

	providerFPath := filepath.Join(
		libraryFPath,
		providerNameToLibsonnetName(providerName),
	)
	if err := writeDocToFile(logger, doc, providerFPath); err != nil {
		return err
	}

	// Render the resource libsonnet files
	for resrcName, resrcSchema := range schema.ResourceSchemas {
		logger.Infof("Rendering %s", resrcName)

		idx.resources = append(
			idx.resources,
			nameWithoutProvider(providerName, resrcName),
		)

		doc, err := renderResourceOrDataSource(
			providerName, resrcName, IsResource, resrcSchema.Block,
		)
		if err != nil {
			return err
		}

		resrcFPath := filepath.Join(
			resourcesFPath,
			nameToLibsonnetName(providerName, resrcName),
		)
		if err := writeDocToFile(logger, doc, resrcFPath); err != nil {
			return err
		}
	}

	// Render the data source libsonnet files
	for datasrcName, datasrcSchema := range schema.DataSourceSchemas {
		logger.Infof("Rendering %s", datasrcName)

		idx.dataSources = append(
			idx.dataSources,
			nameWithoutProvider(providerName, datasrcName),
		)

		doc, err := renderResourceOrDataSource(
			providerName, datasrcName, IsDataSource, datasrcSchema.Block,
		)
		if err != nil {
			return err
		}

		datasrcFPath := filepath.Join(
			dataSourcesFPath,
			nameToLibsonnetName(providerName, datasrcName),
		)
		if err := writeDocToFile(logger, doc, datasrcFPath); err != nil {
			return err
		}
	}

	// Render the _gen index file
	logger.Info("Rendering index files")
	dataIdx := renderDataIndex(idx)
	dataIdxFPath := filepath.Join(dataSourcesFPath, mainLibsonnetName)
	if err := writeDocToFile(logger, &dataIdx, dataIdxFPath); err != nil {
		return err
	}

	genIdx, err := renderIndex(idx)
	if err != nil {
		return err
	}
	genIdxFPath := filepath.Join(libraryFPath, mainLibsonnetName)
	if err := writeDocToFile(logger, &genIdx, genIdxFPath); err != nil {
		return err
	}

	// Render the main index file
	mainImp := j.Import("", filepath.Join(".", "_gen", mainLibsonnetName))
	mainIdx := j.Doc{Root: mainImp}
	mainIdxFPath := filepath.Join(outDir, mainLibsonnetName)
	if err := writeDocToFile(logger, &mainIdx, mainIdxFPath); err != nil {
		return err
	}

	return nil
}
