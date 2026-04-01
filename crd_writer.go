package main

// WriteCRD converts a CustomResourceDefinition object to a kubernetes_manifest
// Terraform resource block. The approach is adapted from tfk8s
// (https://github.com/jrhouston/tfk8s): serialize the object to JSON,
// convert to a cty.Value, then format as HCL.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"k8s.io/apimachinery/pkg/runtime"

	cty "github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// WriteCRD converts a Kubernetes runtime.Object (expected to be a CRD) into
// a Terraform kubernetes_manifest resource block string.
func WriteCRD(obj runtime.Object) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("could not marshal object to JSON: %w", err)
	}

	t, err := ctyjson.ImpliedType(b)
	if err != nil {
		return "", fmt.Errorf("could not determine cty type: %w", err)
	}

	doc, err := ctyjson.Unmarshal(b, t)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal JSON to cty: %w", err)
	}

	m := doc.AsValueMap()

	// Strip the status field — it's server-side state, not part of the manifest.
	delete(m, "status")

	// Strip empty/zero-value metadata fields injected by the Go types.
	if meta, ok := m["metadata"]; ok {
		mm := meta.AsValueMap()
		delete(mm, "creationTimestamp")
		// Remove empty resourceVersion if present
		if rv, ok := mm["resourceVersion"]; ok && rv.AsString() == "" {
			delete(mm, "resourceVersion")
		}
		// Remove zero generation
		if gen, ok := mm["generation"]; ok {
			bf := gen.AsBigFloat()
			if f, _ := bf.Float64(); f == 0 {
				delete(mm, "generation")
			}
		}
		// Remove empty uid
		if uid, ok := mm["uid"]; ok && uid.AsString() == "" {
			delete(mm, "uid")
		}
		m["metadata"] = ctyObjectVal(mm)
	}

	doc = ctyObjectVal(m)

	kind := m["kind"].AsString()
	metadata := m["metadata"].AsValueMap()
	name := metadata["name"].AsString()

	resourceName := camelToSnake(kind) + "-" + snakify(name)

	s := formatValue(doc, 0)
	s = escapeShellVars(s)

	var hcl strings.Builder
	hcl.WriteString(fmt.Sprintf("resource %q %q {\n", "kubernetes_manifest", resourceName))
	hcl.WriteString(fmt.Sprintf("  manifest = %s\n", strings.ReplaceAll(s, "\n", "\n  ")))
	hcl.WriteString("}\n")

	// Format with hclwrite for consistent alignment of = signs.
	formatted := hclwrite.Format([]byte(hcl.String()))

	return string(formatted), nil
}

// ctyObjectVal builds a cty.ObjectVal from a map, handling the case where
// the map might be empty.
func ctyObjectVal(m map[string]cty.Value) cty.Value {
	if len(m) == 0 {
		return cty.EmptyObjectVal
	}
	return cty.ObjectVal(m)
}
