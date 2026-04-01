package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sl1pm4t/k2tf/pkg/testutils"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

var update bool

func init() {
	v := os.Getenv("UPDATE_GOLDEN")
	if strings.ToLower(v) == "true" {
		update = true
	}
}

func testLoadFile(t *testing.T, fileparts ...string) string {
	filename := filepath.Join(fileparts...)
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to load test file, %s: %v", filename, err)
	}

	return string(content)
}

func TestWriteObject(t *testing.T) {
	tests := []struct {
		name            string
		resourceType    string
		wantedWarnCount int
	}{
		{
			"basicDeployment",
			"kubernetes_deployment_v1",
			0,
		},
		{
			"configMap",
			"kubernetes_config_map_v1",
			0,
		},
		{
			"cronJob",
			"kubernetes_cron_job_v1",
			0,
		},
		{
			"daemonset",
			"kubernetes_daemonset_v1",
			0,
		},
		{
			"deployment",
			"kubernetes_deployment_v1",
			0,
		},
		{
			"deployment2Containers",
			"kubernetes_deployment_v1",
			0,
		},
		{
			"endpoints",
			"kubernetes_endpoints_v1",
			0,
		},
		{
			"ingress",
			"kubernetes_ingress_v1",
			0,
		},
		{
			"ingress_v1",
			"kubernetes_ingress_v1",
			0,
		},
		{
			"job",
			"kubernetes_job_v1",
			0,
		},
		{
			"namespace",
			"kubernetes_namespace_v1",
			0,
		},
		{
			"namespace_w_spec",
			"kubernetes_namespace_v1",
			1,
		},
		{
			"networkPolicy",
			"kubernetes_network_policy_v1",
			0,
		},
		{
			"podDisruptionBudget",
			"kubernetes_pod_disruption_budget_v1",
			0,
		},
		{
			"podNodeExporter",
			"kubernetes_pod_v1",
			0,
		},
		{
			"role",
			"kubernetes_role_v1",
			0,
		},
		{
			"roleBinding",
			"kubernetes_role_binding_v1",
			0,
		},
		{
			"service",
			"kubernetes_service_v1",
			0,
		},
		{
			"statefulSet",
			"kubernetes_stateful_set_v1",
			0,
		},
		{
			"issue-48",
			"kubernetes_replication_controller_v1",
			0,
		},
		{
			"certificateSigningRequest",
			"kubernetes_certificate_signing_request_v1",
			0,
		},
		{
			"clusterRole",
			"kubernetes_cluster_role_v1",
			0,
		},
		{
			"issue-28",
			"kubernetes_daemonset_v1",
			0,
		},
		{
			"storageClass",
			"kubernetes_storage_class_v1",
			0,
		},
		{
			"replicationController",
			"kubernetes_replication_controller_v1",
			0,
		},
		{
			"secretStringData",
			"kubernetes_secret_v1",
			0,
		},
		{
			"cronjob_v1",
			"kubernetes_cronjob_v1",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Generate HCL from test data
			obj := testutils.TestParseYAML(t, testLoadFile(t, "test-fixtures", tt.name+".yaml"))
			hclFile := hclwrite.NewEmptyFile()
			warnCount, err := WriteObject(obj, hclFile.Body())
			if err != nil {
				t.Fatal(err)
			}

			// Read our golden file (or optionally write if env var is set)
			goldenFile := filepath.Join("test-fixtures", tt.name+".tf.golden")
			if update {
				os.WriteFile(goldenFile, hclFile.Bytes(), 0644)
			}
			expected := testLoadFile(t, goldenFile)

			// Validate configs are equal
			assert.Equal(t, expected, string(hclFile.Bytes()), "should be equal")

			// Validate warning count
			assert.Equal(t, tt.wantedWarnCount, warnCount, "conversion warning count should match")
		})
	}
}
