package gen

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	. "github.com/onsi/gomega"

	"github.com/google/go-jsonnet"

	"github.com/tf-libsonnet/libgenerator/internal/logging"
)

const (
	jbManifestFile                       = "fixtures/jsonnetfile.json"
	jbLockFile                           = "fixtures/jsonnetfile.lock.json"
	renderLibraryTestCasesDir            = "fixtures/tfcoremock_usage"
	renderLibraryTestCasesExpectedOutDir = "fixtures/tfcoremock_usage_expected_out"
)

func TestRenderLibrary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	logger := logging.GetSugaredLoggerForTest()

	tmpWorkDir, err := os.MkdirTemp("", "test-render-library-*")
	g.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tmpWorkDir)
	createJsonnetWorkDir(g, tmpWorkDir)

	libDir := filepath.Join(tmpWorkDir, "tfcoremock")
	schema := loadSchema(g, tfcoremockSchemaF)
	g.Expect(RenderLibrary(logger, libDir, "tfcoremock", schema)).To(Succeed())

	testCases, err := os.ReadDir(renderLibraryTestCasesDir)
	g.Expect(err).NotTo(HaveOccurred())

	t.Run("group", func(t *testing.T) {
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.Name(), func(t *testing.T) {
				t.Parallel()
				g := NewGomegaWithT(t)

				tcf := filepath.Join(renderLibraryTestCasesDir, tc.Name())
				evalJsonnetAndRunTerraform(g, tmpWorkDir, tcf)
			})
		}
	})
}

func createJsonnetWorkDir(g *WithT, workDir string) {
	mfc, err := os.ReadFile(jbManifestFile)
	g.Expect(err).NotTo(HaveOccurred())
	mfPath := filepath.Join(workDir, filepath.Base(jbManifestFile))
	g.Expect(os.WriteFile(mfPath, mfc, 0644)).To(Succeed())

	lockC, err := os.ReadFile(jbLockFile)
	g.Expect(err).NotTo(HaveOccurred())
	lockPath := filepath.Join(workDir, filepath.Base(jbLockFile))
	g.Expect(os.WriteFile(lockPath, lockC, 0644)).To(Succeed())

	cmd := exec.Command("jb", "install")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	g.Expect(cmd.Run()).To(Succeed())
}

func runTestJsonnetFile(jsf, workDir string) (string, error) {
	vm := jsonnet.MakeVM()

	vendorDir := filepath.Join(workDir, "vendor")
	imp := &jsonnet.FileImporter{JPaths: []string{workDir, vendorDir}}
	vm.Importer(imp)

	return vm.EvaluateFile(jsf)
}

func evalJsonnetAndRunTerraform(
	g *WithT,
	tmpWorkDir, jsonnetFile string,
) {
	jsfBase := filepath.Base(jsonnetFile)

	// Copy jsonnet file from fixture dir to tmp working dir and run it through VM to render it.
	contents, err := os.ReadFile(jsonnetFile)
	g.Expect(err).NotTo(HaveOccurred())

	workingJsonnetFile := filepath.Join(tmpWorkDir, jsfBase)
	g.Expect(os.WriteFile(workingJsonnetFile, contents, 0644)).To(Succeed())

	rendered, err := runTestJsonnetFile(workingJsonnetFile, tmpWorkDir)
	g.Expect(err).NotTo(HaveOccurred())

	// Create a temporary working directory for terraform and copy the rendered json to it.
	tfWorkDir, err := os.MkdirTemp("", "render-library-terraform-*")
	g.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tfWorkDir)

	tmpMainTF := filepath.Join(tfWorkDir, "main.tf.json")
	g.Expect(os.WriteFile(tmpMainTF, []byte(rendered), 0644)).To(Succeed())

	// Load Terraform and apply the rendered tf file
	tfPath, err := exec.LookPath("terraform")
	g.Expect(err).NotTo(HaveOccurred())
	tf, err := tfexec.NewTerraform(tfWorkDir, tfPath)
	g.Expect(err).NotTo(HaveOccurred())
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	ctx := context.Background()
	g.Expect(tf.Init(ctx)).To(Succeed())
	g.Expect(tf.Apply(ctx)).To(Succeed())

	// Load the expected output and compare it against the output from Terraform, if an expected out file exists.
	expectedOutFP := filepath.Join(renderLibraryTestCasesExpectedOutDir, jsfBase+".tfoutputs.json")

	_, statErr := os.Stat(expectedOutFP)
	if statErr != nil {
		return
	}

	expectedOutJSON, err := os.ReadFile(expectedOutFP)
	g.Expect(err).NotTo(HaveOccurred())
	var expectedOutMap map[string]interface{}
	g.Expect(
		json.Unmarshal(expectedOutJSON, &expectedOutMap),
	).To(Succeed())

	outputs, err := tf.Output(ctx)
	g.Expect(err).NotTo(HaveOccurred())
	for k, v := range expectedOutMap {
		expVJSON, err := json.Marshal(v)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(outputs).To(HaveKey(k))
		vJSON, err := json.Marshal(outputs[k].Value)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(vJSON).To(Equal(expVJSON))
	}
}
