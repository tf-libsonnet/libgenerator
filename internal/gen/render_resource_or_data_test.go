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

func TestRenderResourceComplex(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	complexResource := schema.ResourceSchemas["tfcoremock_complex_resource"]

	jt, err := renderResourceOrDataSource(
		IsResource, "tfcoremock_complex_resource", complexResource.Block,
	)
	g.Expect(err).NotTo(HaveOccurred())

	out, err := formatter.Format("", jt.String(), formatter.DefaultOptions())
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf(out)
}

func TestRenderResourceSimple(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	simpleResource := schema.ResourceSchemas["tfcoremock_simple_resource"]

	jt, err := renderResourceOrDataSource(
		IsResource, "tfcoremock_simple_resource", simpleResource.Block,
	)
	g.Expect(err).NotTo(HaveOccurred())

	out, err := formatter.Format("", jt.String(), formatter.DefaultOptions())
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf(out)
}

func TestRenderDataSourceSimple(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	simpleResource := schema.DataSourceSchemas["tfcoremock_simple_resource"]

	jt, err := renderResourceOrDataSource(
		IsDataSource, "tfcoremock_simple_resource", simpleResource.Block,
	)
	g.Expect(err).NotTo(HaveOccurred())

	out, err := formatter.Format("", jt.String(), formatter.DefaultOptions())
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf(out)
}

func TestRenderDataSourceComplex(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)
	complexResource := schema.DataSourceSchemas["tfcoremock_complex_resource"]

	jt, err := renderResourceOrDataSource(
		IsDataSource, "tfcoremock_complex_resource", complexResource.Block,
	)
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
