package gen

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/google/go-jsonnet/formatter"
)

func TestRenderProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	schema := loadSchema(g, tfcoremockSchemaF)

	jt, err := renderProvider("tfcoremock", schema.ConfigSchema.Block)
	g.Expect(err).NotTo(HaveOccurred())

	out, err := formatter.Format("", jt.String(), formatter.DefaultOptions())
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf(out)
}
