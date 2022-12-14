package gen

import (
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

const (
	resourceLabelArg = "resourceLabel"
)

// RenderResource will render the libsonnet code for constructing a resource definition for the given Terraform resource
// block. The generated libsonnet code follows the following canonical pattern:
//
//   - `newAttrs`: A function to construct an object that can be passed in as attrs for the resource, with every required
//     arg as a function arg, and optional args as null. The attrs are meant to be passed into the
//     tf.withResource function.
//   - `new`: A function to construct a mixin to inject the instantiated resource into a root Terraform JSON object.
//     This takes in the same arguments as `newAttrs`.
//   - A `with{ATTRIBUTE_NAME}` function for every attribute, which will generate a mixin to update the given resource
//     block in the document. Note that this flavor of the function will require the resource name so that it knows
//     which resource to update.
func RenderResource(resrcType string, schema *tfjson.SchemaBlock) (*j.Doc, error) {
	// TODO: replace with github.com/tf-libsonnet/core
	locals := []j.LocalType{
		j.Local(j.Import("tf", "github.com/fensak-io/tf-libsonnet/main.libsonnet")),
	}
	rootFields := []j.Type{}

	constructor, err := resrcConstructor(resrcType, schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(constructor))

	attrConstructor, err := resrcAttrsConstructor(schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(attrConstructor))

	for attr, cfg := range schema.Attributes {
		attrFn, err := withAttributeFn(resrcType, attr, cfg)
		if err != nil {
			return nil, err
		}
		rootFields = append(rootFields, j.Hidden(attrFn))
	}

	rootObj := j.Object(resrcType, rootFields...)
	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

func resrcConstructor(resrcType string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
	args := []j.Type{
		j.Required(j.String(resourceLabelArg, "")),
	}
	attrCallArgs := []j.Type{}
	for attr, cfg := range schema.Attributes {
		// Default all the optional args to null, which is treated the same as omitting it from the arg list.
		var arg j.Type = j.Null(attr)
		if cfg.Required {
			arg = j.Required(j.String(attr, ""))
		}
		args = append(args, arg)

		attrCallArgs = append(attrCallArgs, j.Ref(attr, attr))
	}

	attrs := j.Call("attrs", "self.newAttrs", attrCallArgs)
	resource := j.Call(
		"",
		"tf.withResource",
		[]j.Type{
			j.String("type", resrcType),
			j.Ref("label", resourceLabelArg),
			attrs,
		},
	)

	return j.LargeFunc("new", j.Args(args...), resource), nil
}

func resrcAttrsConstructor(schema *tfjson.SchemaBlock) (j.FuncType, error) {
	fields := []j.Type{}
	args := []j.Type{}
	for attr, cfg := range schema.Attributes {
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
	resrcType, attr string,
	cfg *tfjson.SchemaAttribute,
) (j.FuncType, error) {
	valueArgName := "value"
	result := j.Object("",
		j.Merge(j.Object("resource",
			j.Merge(j.Object(resrcType,
				j.Merge(j.Object(fmt.Sprintf("[%s]", resourceLabelArg),
					j.Ref(attr, valueArgName))))))))
	return j.Func(fmt.Sprintf("with%s", strcase.ToCamel(attr)),
		j.Args(j.Required(j.String(resourceLabelArg, "")), j.Required(j.String(valueArgName, ""))),
		result,
	), nil
}
