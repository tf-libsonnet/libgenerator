package gen

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDocStringResourceConsructor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	complexResource := schema.ResourceSchemas["tfcoremock_complex_resource"]
	out, err := constructorDocString(
		"tfcoremock", "tfcoremock_complex_resource",
		IsResource, complexResource.Block,
	)
	g.Expect(err).NotTo(HaveOccurred())
	t.Logf(out)
}

func TestDocStringResourceAttrsConstructor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	complexResource := schema.ResourceSchemas["tfcoremock_complex_resource"]
	out, err := attrsConstructorDocString(
		"tfcoremock", "tfcoremock_complex_resource",
		IsResource, "newAttrs", "", complexResource.Block,
	)
	g.Expect(err).NotTo(HaveOccurred())
	t.Logf(out)
}
