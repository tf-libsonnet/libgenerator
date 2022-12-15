package gen

import (
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
)

type collectionType uint8

const (
	IsListOrSet collectionType = iota
	IsMap
	IsNotCollection
)

func (ct collectionType) String() string {
	switch ct {
	case IsListOrSet:
		return "IsListOrSet"
	case IsMap:
		return "IsMap"
	case IsNotCollection:
		return "IsNotCollection"
	}
	return unknown
}

type resourceOrDataSource uint8

const (
	IsUnknown resourceOrDataSource = iota
	IsResource
	IsDataSource
	IsNestedBlock
)

const (
	constructorFnName        = "new"
	newAttrsFnName           = "newAttrs"
	resourceLabelArg         = "resourceLabel"
	resourceInjectAttrName   = "resource"
	dataSourceLabelArg       = "dataSrcLabel"
	dataSourceInjectAttrName = "data"
	unknown                  = "__UNKNOWN__"
)

func (resrcOrDataSrc resourceOrDataSource) labelArg() string {
	switch resrcOrDataSrc {
	case IsResource:
		return resourceLabelArg
	case IsDataSource:
		return dataSourceLabelArg
	}
	return unknown
}

func (resrcOrDataSrc resourceOrDataSource) injectAttrName() string {
	switch resrcOrDataSrc {
	case IsResource:
		return resourceInjectAttrName
	case IsDataSource:
		return dataSourceInjectAttrName
	}
	return unknown
}

// getInputAttributes filters the schema attributes to only include those that are used as inputs. This skips:
// - the magic id field present on all Terraform blocks.
// - attributes that are read-only.
func getInputAttributes(schema *tfjson.SchemaBlock) map[string]*tfjson.SchemaAttribute {
	out := map[string]*tfjson.SchemaAttribute{}
	for name, cfg := range schema.Attributes {
		if name == "id" {
			continue
		}
		if cfg.Computed && !cfg.Optional {
			continue
		}
		out[name] = cfg
	}
	return out
}

func getCollectionType(nestingMode tfjson.SchemaNestingMode) collectionType {
	switch nestingMode {
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		return IsListOrSet
	// TODO: nested object type should be special cased since it is a typed map
	case tfjson.SchemaNestingModeMap, tfjson.SchemaNestingModeSingle, tfjson.SchemaNestingModeGroup:
		return IsMap
	}
	panic(fmt.Errorf("Unsupported nesting mode: %s", nestingMode))
}
