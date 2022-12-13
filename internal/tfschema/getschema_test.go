package tfschema

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	version "github.com/hashicorp/go-version"
	"github.com/tf-libsonnet/libgenerator/internal/logging"
)

type providerReq struct {
	p string
	v string
}

func TestGetSchemasOneHashicorp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runGetSchemas(
		g,
		[]providerReq{{"null", "~>3.0"}},
		[]string{"registry.terraform.io/hashicorp/null"},
	)
}

func TestGetSchemasOneVendor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runGetSchemas(
		g,
		[]providerReq{{"DopplerHQ/doppler", "~>1.0"}},
		[]string{"registry.terraform.io/dopplerhq/doppler"},
	)
}

func TestGetSchemasMultiple(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runGetSchemas(
		g,
		[]providerReq{
			{"null", "~>3.0"},
			{"DopplerHQ/doppler", "~>1.0"},
		},
		[]string{
			"registry.terraform.io/hashicorp/null",
			"registry.terraform.io/dopplerhq/doppler",
		},
	)
}

func runGetSchemas(g *WithT, providerReqs []providerReq, expectedKeys []string) {
	reqL := SchemaRequestList{}
	for _, pr := range providerReqs {
		req, err := NewSchemaRequest(pr.p, pr.v)
		g.Expect(err).NotTo(HaveOccurred())
		reqL = append(reqL, req)
	}

	logger := logging.GetSugaredLoggerForTest()
	ctx := context.Background()
	tfV := version.Must(version.NewVersion("1.3.6"))
	schemas, err := GetSchemas(logger, ctx, tfV, reqL)
	g.Expect(err).NotTo(HaveOccurred())

	for _, k := range expectedKeys {
		g.Expect(schemas.Schemas).To(HaveKey(k))
	}
}
