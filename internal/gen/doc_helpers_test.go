package gen

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDocStringResource(t *testing.T) {
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
