package gen

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/google/go-jsonnet/formatter"
	tfjson "github.com/hashicorp/terraform-json"
)

const (
	tfcoremockSchemaF = "fixtures/tfcoremock_schema.json"
)

func TestRenderResource(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	simpleResource := schema.ResourceSchemas["tfcoremock_simple_resource"]

	jt, err := RenderResource("tfcoremock_simple_resource", simpleResource.Block)
	g.Expect(err).NotTo(HaveOccurred())

	out, err := formatter.Format("", jt.String(), formatter.DefaultOptions())
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf(out)
}

func loadSchema(g *WithT, fixturePath string) *tfjson.ProviderSchema {
	data, err := os.ReadFile(tfcoremockSchemaF)
	g.Expect(err).NotTo(HaveOccurred())

	var schema tfjson.ProviderSchema
	jsonErr := json.Unmarshal(data, &schema)
	g.Expect(jsonErr).NotTo(HaveOccurred())

	return &schema
}
