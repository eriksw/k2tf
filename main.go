package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sl1pm4t/k2tf/pkg/file_io"
	"github.com/sl1pm4t/k2tf/pkg/tfkschema"
	flag "github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/rs/zerolog/log"
)

// Build time variables
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Command line flags
var (
	debug              bool
	input              string
	output             string
	outputDir          string
	includeUnsupported bool
	noColor            bool
	overwriteExisting  bool
	tf12format         bool
	printVersion       bool
)

// convertedResource holds a converted Terraform resource for sorting and output.
type convertedResource struct {
	resourceType string
	resourceName string
	hcl          string
	isCRD        bool
}

func init() {
	// init command line flags
	flag.BoolVarP(&overwriteExisting, "overwrite-existing", "x", false, "allow overwriting existing output file(s)")
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug output")
	flag.StringVarP(&input, "filepath", "f", "-", `file or directory that contains the YAML configuration to convert. Use "-" to read from stdin`)
	flag.StringVarP(&output, "output", "o", "-", `file or directory where Terraform config will be written`)
	flag.StringVarP(&outputDir, "output-dir", "O", "", `output directory for split files (one .tf file per resource)`)
	flag.BoolVarP(&includeUnsupported, "include-unsupported", "I", false, `set to true to include unsupported Attributes / Blocks in the generated TF config`)
	flag.BoolVarP(&tf12format, "tf12format", "F", false, `Use Terraform 0.12 formatter`)
	flag.BoolVarP(&printVersion, "version", "v", false, `Print k2tf version`)

	flag.Parse()

	setupLogOutput()
}

func main() {
	if printVersion {
		fmt.Printf("k2tf version: %s\n", version)
		os.Exit(0)
	}

	log.Debug().
		Str("version", version).
		Str("commit", commit).
		Str("builddate", date).
		Msg("starting k2tf")

	objs := file_io.ReadInput(input)

	log.Debug().Msgf("read %d objects from input", len(objs))

	// Convert all objects to HCL strings, collecting metadata for sorting.
	var resources []convertedResource
	for i, obj := range objs {
		if _, isCRD := obj.(*apiextensionsv1.CustomResourceDefinition); isCRD {
			hcl, err := WriteCRD(obj)
			if err != nil {
				log.Error().Int("obj#", i).Err(err).Msg("error writing CRD object")
				continue
			}
			resources = append(resources, convertedResource{
				resourceType: "kubernetes_manifest",
				resourceName: crdResourceName(obj),
				hcl:          hcl,
				isCRD:        true,
			})
		} else if tfkschema.IsKubernetesKindSupported(obj) {
			f := hclwrite.NewEmptyFile()
			_, err := WriteObject(obj, f.Body())
			if err != nil {
				log.Error().Int("obj#", i).Err(err).Msg("error writing object")
			}
			formatted := formatObject(f.Bytes())
			resources = append(resources, convertedResource{
				resourceType: tfkschema.ToTerraformResourceType(obj),
				resourceName: tfkschema.ToTerraformResourceName(obj),
				hcl:          string(formatted),
				isCRD:        false,
			})
		} else {
			log.Warn().Str("kind", obj.GetObjectKind().GroupVersionKind().Kind).Msg("skipping API object, kind not supported by Terraform provider.")
		}
	}

	// Sort resources by type, then by name.
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].resourceType != resources[j].resourceType {
			return resources[i].resourceType < resources[j].resourceType
		}
		return resources[i].resourceName < resources[j].resourceName
	})

	if outputDir != "" {
		writeSplitOutput(resources)
	} else {
		writeSingleOutput(resources)
	}
}

// writeSingleOutput writes all resources to a single output (file or stdout).
func writeSingleOutput(resources []convertedResource) {
	w, closer := file_io.SetupOutput(output, overwriteExisting)
	defer closer()

	for _, r := range resources {
		fmt.Fprint(w, r.hcl)
		fmt.Fprintln(w)
	}
}

// writeSplitOutput writes each resource to its own file in the output directory.
// Files are named as <resource_type>-<resource_name>.tf.
func writeSplitOutput(resources []convertedResource) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal().Err(err).Msg("could not create output directory")
	}

	for _, r := range resources {
		typeName := strings.TrimPrefix(r.resourceType, "kubernetes_manifest")
		typeName = strings.TrimPrefix(typeName, "kubernetes_")
		typeName = strings.TrimPrefix(typeName, "-")
		typeName = strings.TrimPrefix(typeName, "_")

		var filename string
		if typeName == "" {
			filename = r.resourceName + ".tf"
		} else {
			filename = typeName + "-" + r.resourceName + ".tf"
		}
		path := filepath.Join(outputDir, filename)

		f := openOutputFile(path)
		fmt.Fprint(f, r.hcl)
		f.Close()
	}

	log.Info().Str("dir", outputDir).Int("count", len(resources)).Msg("wrote output files")
}

func openOutputFile(path string) *os.File {
	if _, err := os.Stat(path); err == nil && !overwriteExisting {
		log.Fatal().Str("file", path).Msg("output file already exists (use -x to overwrite)")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal().Err(err).Str("file", path).Msg("could not open output file")
	}
	return f
}

// crdResourceName extracts the terraform resource name for a CRD object.
func crdResourceName(obj interface{}) string {
	if crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition); ok {
		return "custom_resource_definition-" + snakify(crd.Name)
	}
	return ""
}

func formatObject(in []byte) []byte {
	var result []byte
	var err error

	if tf12format {
		result = hclwrite.Format(in)
	} else {
		result, err = printer.Format(in)
		if err != nil {
			log.Error().Err(err).Msg("could not format object")
			return in
		}
	}

	return result
}
