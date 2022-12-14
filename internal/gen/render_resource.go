package gen

import (
	tfjson "github.com/hashicorp/terraform-json"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

const (
	resourceLabelArg = "resourceLabel"
)

func RenderResource(name string, schema *tfjson.SchemaBlock) (*j.Doc, error) {
	// TODO: replace with github.com/tf-libsonnet/core
	locals := []j.LocalType{
		j.Local(j.Import("tf", "github.com/fensak-io/tf-libsonnet/main.libsonnet")),
	}

	constructor, err := resrcConstructor(name, schema)
	if err != nil {
		return nil, err
	}
	attrConstructor, err := resrcAttrsConstructor(schema)
	if err != nil {
		return nil, err
	}

	fields := []j.Type{
		j.Hidden(constructor),
		j.Hidden(attrConstructor),
	}
	rootObj := j.Object(name, fields...)

	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

func resrcConstructor(name string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
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
			j.Ref("type", name),
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
