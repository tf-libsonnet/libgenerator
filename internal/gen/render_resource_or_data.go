package gen

import (
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

type resourceOrDataSource uint8

const (
	IsUnknown resourceOrDataSource = iota
	IsResource
	IsDataSource
)

const (
	resourceLabelArg   = "resourceLabel"
	dataSourceLabelArg = "dataSrcLabel"
)

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
	// TODO: replace with github.com/tf-libsonnet/core
	locals := []j.LocalType{
		j.Local(j.Import("tf", "github.com/fensak-io/tf-libsonnet/main.libsonnet")),
	}
	rootFields := []j.Type{}

	constructor, err := constructor(resrcOrDataSrc, typ, schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(constructor))

	attrConstructor, err := attrsConstructor(schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(attrConstructor))

	for attr, cfg := range getInputAttributes(schema) {
		rootFields = append(
			rootFields,
			j.Hidden(withAttributeFn(resrcOrDataSrc, typ, attr, cfg)),
		)
	}

	rootObj := j.Object(typ, rootFields...)
	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

func constructor(resrcOrDataSrc resourceOrDataSource, typ string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
	labelArg := resourceLabelArg
	if resrcOrDataSrc == IsDataSource {
		labelArg = dataSourceLabelArg
	}

	args := []j.Type{
		j.Required(j.String(labelArg, "")),
	}
	attrCallArgs := []j.Type{}
	for attr, cfg := range getInputAttributes(schema) {
		// Default all the optional args to null, which is treated the same as omitting it from the arg list.
		var arg j.Type = j.Null(attr)
		if cfg.Required {
			arg = j.Required(j.String(attr, ""))
		}
		args = append(args, arg)

		attrCallArgs = append(attrCallArgs, j.Ref(attr, attr))
	}

	attrs := j.Call("attrs", "self.newAttrs", attrCallArgs)
	fn := "tf.withResource"
	if resrcOrDataSrc == IsDataSource {
		fn = "tf.withData"
	}
	resource := j.Call(
		"",
		fn,
		[]j.Type{
			j.String("type", typ),
			j.Ref("label", resourceLabelArg),
			attrs,
		},
	)

	return j.LargeFunc("new", j.Args(args...), resource), nil
}

func attrsConstructor(schema *tfjson.SchemaBlock) (j.FuncType, error) {
	fields := []j.Type{}
	args := []j.Type{}
	for attr, cfg := range getInputAttributes(schema) {
		fields = append(fields, j.Ref(attr, attr))

		// Default all the optional args to null, which is treated the same as omitting it from the arg list.
		var arg j.Type = j.Null(attr)
		if cfg.Required {
			arg = j.Required(j.String(attr, ""))
		}
		args = append(args, arg)
	}

	// Prune null attributes so they are omitted from the final json.
	// Although this is not strictly necessary to do, it makes the rendered json nice and tidy.
	return j.LargeFunc("newAttrs",
		j.Args(args...),
		j.Call("", "std.prune", []j.Type{j.Object("a", fields...)}),
	), nil
}

func withAttributeFn(
	resrcOrDataSrc resourceOrDataSource,
	typ, attr string,
	cfg *tfjson.SchemaAttribute,
) j.FuncType {
	valueArgName := "value"
	labelArg := resourceLabelArg
	injectName := "resource"
	if resrcOrDataSrc == IsDataSource {
		labelArg = dataSourceLabelArg
		injectName = "data"
	}

	// NOTE: this is a hack to work around the lack of functionality to introduce a reference key merge in the builder
	// library. This takes advantage of a quirk where the builder outputs the literal string name of the object as the key
	// for the merge operation. So using the reference name wrapped in [] as the merge key results in the literal
	// `[REFERENCE]+` without quotes being printed out in the resulting jsonnet, which is what we want.
	// The maintainers of the k8s generator library may change this behavior in the future!
	refMerge := fmt.Sprintf("[%s]", labelArg)

	result := j.Object("",
		j.Merge(j.Object(injectName,
			j.Merge(j.Object(typ,
				j.Merge(j.Object(refMerge,
					j.Ref(attr, valueArgName))))))))
	return j.Func(fmt.Sprintf("with%s", strcase.ToCamel(attr)),
		j.Args(j.Required(j.String(labelArg, "")), j.Required(j.String(valueArgName, ""))),
		result,
	)
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
