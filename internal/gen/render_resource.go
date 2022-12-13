package gen

import (
	tfjson "github.com/hashicorp/terraform-json"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

func RenderResource(name string, schema *tfjson.SchemaBlock) (j.Type, error) {
	constructor, err := resourceConstructor(schema)
	if err != nil {
		return nil, err
	}

	fields := []j.Type{constructor}
	return j.Object(name, fields...), nil
}

func resourceConstructor(schema *tfjson.SchemaBlock) (j.FuncType, error) {
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
	return j.LargeFunc("new",
		j.Args(args...),
		j.Call("", "std.prune", []j.Type{j.Object("a", fields...)}),
	), nil
}
