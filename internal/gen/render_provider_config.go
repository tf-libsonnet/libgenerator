package gen

import (
	"sort"

	tfjson "github.com/hashicorp/terraform-json"
	j "github.com/jsonnet-libs/k8s/pkg/builder"
)

// renderProvider will render the libsonnet code for constructing a provider block for the given provider. The generated
// libsonnet code will only consist of the constructors (including for nested blocks), as implementing the mixin
// functions for the provider is difficult due to providers being a list instead of a map.
func renderProvider(name string, schema *tfjson.SchemaBlock) (*j.Doc, error) {
	locals := []j.LocalType{
		j.Local(importCore()),
		j.Local(importDocsonnet()),
	}
	rootFields := sortedTypeList{}

	constructor, err := providerConstructor(name, schema)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(constructor))

	attrsConstructor, attrsConstructorDocs, err := attrsConstructor(
		newAttrsFnName, "", name, IsProvider, schema,
	)
	if err != nil {
		return nil, err
	}
	rootFields = append(rootFields, j.Hidden(*attrsConstructor), j.Hidden(*attrsConstructorDocs))

	// Render constructor for nested blocks
	nestedFields := sortedTypeList{}
	for _, cfg := range getNestedBlocks(schema) {
		blockObj, err := nestedBlockObject(name, cfg)
		if err != nil {
			return nil, err
		}
		nestedFields = append(nestedFields, j.Hidden(blockObj))
	}
	sort.Sort(nestedFields)
	rootFields = append(rootFields, nestedFields...)

	rootObj := j.Object("provider", rootFields...)
	return &j.Doc{Locals: locals, Root: rootObj}, nil
}

func providerConstructor(name string, schema *tfjson.SchemaBlock) (j.FuncType, error) {
	providerCallArgs := []j.Type{j.String("name", name)}

	params := constructorParamList(schema)

	// Add the provider specific args:
	// - alias for setting an alias on the provider block
	// - src and version for injecting in required_providers in the resulting document.
	providerParams := []string{"alias", "src", "version"}
	for _, p := range providerParams {
		params.params = append(params.params, j.Null(p))
		providerCallArgs = append(providerCallArgs, j.Ref(p, p))
	}

	attrs := j.Call("attrs", "self."+newAttrsFnName, params.attrsCallArgs)
	providerCallArgs = append(providerCallArgs, attrs)

	fnName := "tf.withProvider"
	resource := j.Call(
		"",
		fnName,
		providerCallArgs,
	)

	fn := j.LargeFunc(
		constructorFnName,
		j.Args(params.params...),
		resource,
	)
	return fn, nil
}
