package gen

import (
	"fmt"
	"sort"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
	d "github.com/jsonnet-libs/k8s/pkg/builder/docsonnet"
)

// renderResourceOrDataSource will render the libsonnet code for constructing a resource or data source definition for
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
//     block will have its own `new` functions for constructing the nested block object.
//   - Nested blocks will recursively nest subblocks if the nested blocks have its own nested blocks.
func renderResourceOrDataSource(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (*j.Doc, error) {
	locals := []j.LocalType{
		importCore(),
		importDocsonnet(),
	}
	rootFields := sortedTypeList{}

	constructorDocs, err := constructorDocs(providerName, typ, resrcOrDataSrc, schema)
	if err != nil {
		return nil, err
	}
	constructor, err := constructor(providerName, typ, resrcOrDataSrc, schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, *constructorDocs, j.Hidden(*constructor))

	attrConstructorDocs, err := attrsConstructorDocs(
		providerName, typ, resrcOrDataSrc, newAttrsFnName, "", schema,
	)
	if err != nil {
		return nil, err
	}
	attrConstructor, err := attrsConstructor(
		newAttrsFnName, providerName, typ, resrcOrDataSrc, schema,
	)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, *attrConstructorDocs, j.Hidden(*attrConstructor))

	// Add modifier functions for each attribute
	for _, cfg := range getInputAttributes(schema) {
		bareWithFnDoc, err := withFnDocs(
			providerName, typ, resrcOrDataSrc, cfg.tfName, getAttrType(cfg.attr), IsNotCollection,
			false,
		)
		if err != nil {
			return nil, err
		}
		bareWithFn, err := withAttributeOrBlockFn(
			resrcOrDataSrc, providerName, typ, cfg.tfName, false, IsNotCollection,
		)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, *bareWithFn, j.Hidden(*bareWithFnDoc))

		if cfg.attr.AttributeNestedType != nil {
			collTyp := getCollectionType(cfg.attr.AttributeNestedType.NestingMode)
			mixinWithFnDoc, err := withFnDocs(
				providerName, typ, resrcOrDataSrc, cfg.tfName, getAttrType(cfg.attr), collTyp,
				true,
			)
			if err != nil {
				return nil, err
			}
			mixinWithFn, err := withAttributeOrBlockFn(
				resrcOrDataSrc, providerName, typ, cfg.tfName, true, collTyp,
			)
			if err != nil {
				return nil, err
			}
			rootFields = append(rootFields, *mixinWithFnDoc, j.Hidden(*mixinWithFn))
		}
	}

	// Add modifier functions for each block
	for block, cfg := range getNestedBlocks(schema) {
		collTyp := getCollectionType(cfg.block.NestingMode)

		bareWithFnDoc, err := withFnDocs(
			providerName, typ, resrcOrDataSrc, cfg.tfName, getBlockType(cfg.block.NestingMode), collTyp,
			false,
		)
		if err != nil {
			return nil, err
		}
		bareWithFn, err := withAttributeOrBlockFn(
			resrcOrDataSrc, providerName, typ, block, false, collTyp,
		)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, *bareWithFn, j.Hidden(*bareWithFnDoc))

		mixinWithFnDoc, err := withFnDocs(
			providerName, typ, resrcOrDataSrc, cfg.tfName, getBlockType(cfg.block.NestingMode), collTyp,
			true,
		)
		if err != nil {
			return nil, err
		}
		mixinWithFn, err := withAttributeOrBlockFn(
			resrcOrDataSrc, providerName, typ, cfg.tfName, true, collTyp,
		)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, *mixinWithFn, j.Hidden(*mixinWithFnDoc))

		objectName := nameWithoutProvider(providerName, typ)
		providerNameForNested := fmt.Sprintf(
			"%s.%s",
			providerName, objectName,
		)
		blockObj, err := nestedBlockObject(providerNameForNested, cfg.tfName, cfg)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(blockObj))
	}

	sort.Sort(rootFields)

	// Inject the package docs at the top
	docstr, err := objectDocString(providerName, typ, resrcOrDataSrc, schema)
	if err != nil {
		return nil, err
	}
	docs := d.Pkg(
		nameWithoutProvider(providerName, typ),
		"",
		docstr,
	)
	rootFields = append([]j.Type{docs}, rootFields...)

	rootObj := j.Object(typ, rootFields...)
	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

// constructor returns the function implementation to construct a new resource or data source into the root terraform
// document. This will also return the docsonnet compatible docstring.
func constructor(
	providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (*j.FuncType, error) {
	params := constructorParamList(schema)

	// Prepend the label param after it has been sorted so that it is always the first function parameter.
	labelParamName := resrcOrDataSrc.labelArg()
	labelParam := j.Required(j.String(labelParamName, ""))
	params.params = append(sortedTypeList{labelParam}, params.params...)

	// Append the `_meta` param after it has been sorted so that it is always the last function parameter.
	metaParam := j.Object(metaParamName)
	params.params = append(params.params, metaParam)

	attrs := j.Call("attrs", "self."+newAttrsFnName, params.attrsCallArgs)
	fnCall := "tf.withResource"
	if resrcOrDataSrc == IsDataSource {
		fnCall = "tf.withData"
	}
	resource := j.Call(
		"",
		fnCall,
		[]j.Type{
			j.String("type", typ),
			j.Ref("label", labelParamName),
			attrs,
			j.Ref(metaParamName, metaParamName),
		},
	)

	// Construct function
	fn := j.LargeFunc(
		constructorFnName,
		j.Args(params.params...),
		resource,
	)
	return &fn, nil
}

// attrsConstructor returns the function implementation to construct a new mixin object to set attributes on a resource
// or data source in the root terraform document. This will also return the docsonnet compatible docstring.
func attrsConstructor(
	fnName, providerName, typ string,
	resrcOrDataSrc resourceOrDataSource,
	schema *tfjson.SchemaBlock,
) (*j.FuncType, error) {
	params := constructorParamList(schema)

	// Prune null attributes so they are omitted from the final json.
	// Although this is not strictly necessary to do, it makes the rendered terraform json (NOT jsonnet code!) nice and
	// tidy.
	fn := j.LargeFunc(
		fnName,
		j.Args(params.params...),
		j.Call(
			"",
			"std.prune",
			[]j.Type{j.Object("a", params.tfFieldSetters...)},
		),
	)
	return &fn, nil
}

func withAttributeOrBlockFn(
	resrcOrDataSrc resourceOrDataSource,
	providerName, typ, attrTFName string,
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

	fnName := fmt.Sprintf("with%s", strcase.ToCamel(attrTFName))
	var attrRef j.Type = j.Ref(attrTFName, valueArgName)

	if isMixin {
		fnName = fnName + "Mixin"
		switch collTyp {
		case IsMap:
			attrRef = j.Merge(attrRef)
		case IsListOrSet:
			// For lists or sets, we want to conditionally convert the arg to a list so that it can be appended.
			conditional := j.IfThenElse(attrTFName,
				j.Call("", "std.isArray", []j.Type{j.Ref("v", valueArgName)}),
				attrRef,
				j.List("", attrRef),
			)
			attrRef = j.Merge(conditional)
		default:
			return nil, fmt.Errorf("Mixin function for attribute %s with collection type %s is not supported", attrTFName, collTyp)
		}
	}

	result := j.Object("",
		j.Merge(j.Object(resrcOrDataSrc.injectAttrName(),
			j.Merge(j.Object(typ,
				j.Merge(j.Object(refMerge,
					attrRef)))))))
	fn := j.Func(fnName,
		j.Args(
			j.Required(j.String(resrcOrDataSrc.labelArg(), "")),
			j.Required(j.String(valueArgName, "")),
		),
		result,
	)
	return &fn, nil
}

// nestedBlockObject renders the object with functions for constructing and modifying nested blocks on the resource or
// data source.
// For now, this is just the constructors. In the future, we may add mixin objects, but these are currently not
// implemented due to the complexity involved in setting up the merge operators correctly across the nested levels.
// nestedName tracks the number of nesting that has occurred, and is used for constructing the relative links in
// the docsonnet docs. This should represent the level at the current object, and should include the nested block name.
func nestedBlockObject(providerName, nestedName string, cfg *block) (j.Type, error) {
	errRet := j.Null(cfg.tfName)
	objFields := sortedTypeList{}

	constructorDocs, err := attrsConstructorDocs(
		providerName, cfg.tfName, IsNestedBlock, constructorFnName, nestedName, cfg.block.Block,
	)
	if err != nil {
		return errRet, err
	}
	constructor, err := attrsConstructor(
		constructorFnName, providerName, cfg.tfName, IsNestedBlock, cfg.block.Block,
	)
	if err != nil {
		return errRet, err
	}
	objFields = append(objFields, *constructorDocs, j.Hidden(*constructor))

	// Add nested objects for deep nested blocks as well.
	for _, nestedCfg := range getNestedBlocks(cfg.block.Block) {
		providerNameForNested := fmt.Sprintf("%s.%s", providerName, cfg.tfName)
		deepNestedBlockObj, err := nestedBlockObject(
			providerNameForNested,
			nestedName+cfg.tfName,
			nestedCfg,
		)
		if err != nil {
			return errRet, err
		}
		objFields = append(objFields, j.Hidden(deepNestedBlockObj))
	}

	sort.Sort(objFields)

	obj := j.Object(cfg.tfName, objFields...)
	return obj, nil
}
