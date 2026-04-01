package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sl1pm4t/k2tf/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCRD(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"crd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := testutils.TestParseYAML(t, testLoadFile(t, "test-fixtures", tt.name+".yaml"))

			hcl, err := WriteCRD(obj)
			require.NoError(t, err)

			goldenFile := filepath.Join("test-fixtures", tt.name+".tf.golden")
			if strings.ToLower(os.Getenv("UPDATE_GOLDEN")) == "true" {
				os.WriteFile(goldenFile, []byte(hcl), 0644)
			}
			expected := testLoadFile(t, goldenFile)

			assert.Equal(t, expected, hcl, "CRD HCL output should match golden file")
		})
	}
}
