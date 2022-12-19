package gen

import (
	_ "embed"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

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
