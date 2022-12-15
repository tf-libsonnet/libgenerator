package gen

import (
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
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

// RenderResourceOrDataSource will render the libsonnet code for constructing a resource or data source definition for
// the given Terraform block. The generated libsonnet code has the following canonical pattern:
//
//   - `newAttrs`: A function to construct an object that can be passed in as attrs for the resource or data source,
//     with every required arg as a function arg, and optional args as null. The attrs are meant to be passed into the
//     tf.withResource or tf.withData function.
//   - `new`: A function to construct a mixin to inject the instantiated resource or data source into a root Terraform
//     JSON object. This takes in the same arguments as `newAttrs`.
//   - A `with{ATTRIBUTE_NAME}` function for every attribute, which will generate a mixin to update the given resource
//     or data source block in the document. Note that this flavor of the function will require the name so that it
//     knows which resource or data source to update.
//   - Each nested block will be an object attributed by the block name in the resulting jsonnet document. The nested
//     block will have it's own `new` and `with{ATTRIBUTE_NAME}` functions.
func RenderResourceOrDataSource(
	resrcOrDataSrc resourceOrDataSource,
	typ string,
	schema *tfjson.SchemaBlock,
) (*j.Doc, error) {
	var locals []j.LocalType
	var rootFields []j.Type

	if resrcOrDataSrc == IsNestedBlock {
		// Nested block constructor should be an attrs constructor
		constructor, err := attrsConstructor(constructorFnName, schema)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(constructor))
	} else {
		// TODO: replace with github.com/tf-libsonnet/core
		locals = append(
			locals,
			j.Local(j.Import("tf", "github.com/fensak-io/tf-libsonnet/main.libsonnet")),
		)

		constructor, err := constructor(resrcOrDataSrc, typ, schema)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(constructor))

		attrConstructor, err := attrsConstructor(newAttrsFnName, schema)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(attrConstructor))
	}

	// Add modifier functions for each attribute
	for attr, cfg := range getInputAttributes(schema) {
		bareWithFn, err := withAttributeOrBlockFn(resrcOrDataSrc, typ, attr, false, IsNotCollection)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(*bareWithFn))

		if cfg.AttributeNestedType != nil {
			collTyp := getCollectionType(cfg.AttributeNestedType.NestingMode)
			mixinWithFn, err := withAttributeOrBlockFn(resrcOrDataSrc, typ, attr, true, collTyp)
			if err != nil {
				return nil, err
			}
			rootFields = append(rootFields, j.Hidden(*mixinWithFn))
		}
	}

	// Add modifier functions for each block
	for block, cfg := range schema.NestedBlocks {
		bareWithFn, err := withAttributeOrBlockFn(resrcOrDataSrc, typ, block, false, IsNotCollection)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(*bareWithFn))

		collTyp := getCollectionType(cfg.NestingMode)
		mixinWithFn, err := withAttributeOrBlockFn(resrcOrDataSrc, typ, block, true, collTyp)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(*mixinWithFn))

		// TODO: implement nested block constructor
	}

	rootObj := j.Object(typ, rootFields...)
	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

func constructor(resrcOrDataSrc resourceOrDataSource, typ string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
	args := []j.Type{
		j.Required(j.String(resrcOrDataSrc.labelArg(), "")),
	}
	attrCallArgs := []j.Type{}

	// Add args for the attributes
	for attr, cfg := range getInputAttributes(schema) {
		// Default all the optional args to null, which is treated the same as omitting it from the arg list.
		var arg j.Type = j.Null(attr)
		if cfg.Required {
			arg = j.Required(j.String(attr, ""))
		}
		args = append(args, arg)

		attrCallArgs = append(attrCallArgs, j.Ref(attr, attr))
	}

	// Add args for the nested blocks
	for block := range schema.NestedBlocks {
		// Nested blocks can not be labeled as required so always assume optional.
		args = append(args, j.Null(block))
		attrCallArgs = append(attrCallArgs, j.Ref(block, block))
	}

	attrs := j.Call("attrs", "self."+newAttrsFnName, attrCallArgs)
	fn := "tf.withResource"
	if resrcOrDataSrc == IsDataSource {
		fn = "tf.withData"
	}
	resource := j.Call(
		"",
		fn,
		[]j.Type{
			j.String("type", typ),
			j.Ref("label", resrcOrDataSrc.labelArg()),
			attrs,
		},
	)

	return j.LargeFunc(constructorFnName, j.Args(args...), resource), nil
}

func attrsConstructor(fnName string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
	fields := []j.Type{}
	args := []j.Type{}

	// Add args for the attributes
	for attr, cfg := range getInputAttributes(schema) {
		fields = append(fields, j.Ref(attr, attr))

		// Default all the optional args to null, which is treated the same as omitting it from the arg list.
		var arg j.Type = j.Null(attr)
		if cfg.Required {
			arg = j.Required(j.String(attr, ""))
		}
		args = append(args, arg)
	}

	// Add args for the nested blocks
	for block := range schema.NestedBlocks {
		fields = append(fields, j.Ref(block, block))

		// Nested blocks can not be labeled as required so always assume optional.
		args = append(args, j.Null(block))
	}

	// Prune null attributes so they are omitted from the final json.
	// Although this is not strictly necessary to do, it makes the rendered json nice and tidy.
	return j.LargeFunc(fnName,
		j.Args(args...),
		j.Call("", "std.prune", []j.Type{j.Object("a", fields...)}),
	), nil
}

// TODO: handle nested blocks correctly
func withAttributeOrBlockFn(
	resrcOrDataSrc resourceOrDataSource,
	typ, attr string,
	isMixin bool,
	collTyp collectionType,
) (*j.FuncType, error) {
	valueArgName := "value"

	// NOTE: this is a hack to work around the lack of functionality to introduce a reference key merge in the builder
	// library. This takes advantage of a quirk where the builder outputs the literal string name of the object as the key
	// for the merge operation. So using the reference name wrapped in [] as the merge key results in the literal
	// `[REFERENCE]+` without quotes being printed out in the resulting jsonnet, which is what we want.
	// The maintainers of the k8s generator library may change this behavior in the future!
	refMerge := fmt.Sprintf("[%s]", resrcOrDataSrc.labelArg())

	fnName := fmt.Sprintf("with%s", strcase.ToCamel(attr))
	var attrRef j.Type = j.Ref(attr, valueArgName)

	if isMixin {
		fnName = fnName + "Mixin"
		switch collTyp {
		case IsMap:
			attrRef = j.Merge(attrRef)
		case IsListOrSet:
			// For lists or sets, we want to conditionally convert the arg to a list so that it can be appended.
			conditional := j.IfThenElse(attr,
				j.Call("", "std.isArray", []j.Type{j.Ref("v", valueArgName)}),
				attrRef,
				j.List("", attrRef),
			)
			attrRef = j.Merge(conditional)
		default:
			return nil, fmt.Errorf("Mixin function for attribute %s with collection type %s is not supported", attr, collTyp)
		}
	}

	result := j.Object("",
		j.Merge(j.Object(resrcOrDataSrc.injectAttrName(),
			j.Merge(j.Object(typ,
				j.Merge(j.Object(refMerge,
					attrRef)))))))
	fn := j.Func(fnName,
		j.Args(j.Required(j.String(resrcOrDataSrc.labelArg(), "")), j.Required(j.String(valueArgName, ""))),
		result,
	)
	return &fn, nil
}

// nestedBlockObject renders the object with functions for constructing and modifying nested blocks on the resource or
// data source.
func nestedBlockObject(block string, cfg *tfjson.SchemaBlockType) (j.Type, error) {
	doc, err := RenderResourceOrDataSource(IsNestedBlock, block, cfg.Block)
	if err != nil {
		return j.Null(block), err
	}
	return doc.Root, nil
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