package iterative

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sebdah/goldie/v2"
	"os"
	"testing"
)

func generateEnvironmentTestData(t *testing.T) map[string]string {
	return map[string]string{
		"AWS_SECRET_ACCESS_KEY":    "1 value with \"quotes\" and spaces",
		"AWS_ACCESS_KEY_ID":        "2 value with \"quotes\" and spaces",
		"AZURE_CLIENT_ID":          "3 value with \"quotes\" and spaces",
		"AZURE_CLIENT_SECRET":      "4 value with \"quotes\" and spaces",
		"AZURE_SUBSCRIPTION_ID":    "5 value with \"quotes\" and spaces",
		"AZURE_TENANT_ID":          "6 value with \"quotes\" and spaces",
		"KUBERNETES_CONFIGURATION": "7 value with \"quotes\" and spaces",
	}
}

func generateSchemaTestData(cloud string, t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceRunner().Schema, map[string]interface{}{
		"cloud":             cloud,
		"region":            "9 value with \"quotes\" and spaces",
		"name":              "10 value with \"quotes\" and spaces",
		"idle_timeout":      11,
		"instance_hdd_size": 12,
		"token":             "13 value with \"quotes\" and spaces",
		"repo":              "14 value with \"quotes\" and spaces",
		"driver":            "15 value with \"quotes\" and spaces",
		"labels":            "16 value with \"quotes\" and spaces",
		"instance_gpu":      "17 value with \"quotes\" and spaces",
	})
}

func generateProvisionerCode(t *testing.T, cloud string) (string, error) {
	// Note for future tests: this code modifies process environment variables.
	for key, value := range generateEnvironmentTestData(t) {
		os.Setenv(key, value)
	}
	return provisionerCode(generateSchemaTestData(cloud, t))
}

func TestProvisionerCodeAWS(t *testing.T) {
	if val, err := generateProvisionerCode(t, "aws"); err == nil {
		goldie.New(t).Assert(t, "provisioner_code_cloud_aws", []byte(val))
	} else {
		t.Fail()
	}
}

func TestProvisionerCodeAzure(t *testing.T) {
	if val, err := generateProvisionerCode(t, "azure"); err == nil {
		goldie.New(t).Assert(t, "provisioner_code_cloud_azure", []byte(val))
	} else {
		t.Fail()
	}
}

func TestProvisionerCodeKubernetes(t *testing.T) {
	if val, err := generateProvisionerCode(t, "kubernetes"); err == nil {
		goldie.New(t).Assert(t, "provisioner_code_cloud_kubernetes", []byte(val))
	} else {
		t.Fail()
	}
}

func TestProvisionerCodeInvalid(t *testing.T) {
	if val, err := generateProvisionerCode(t, "invalid"); err == nil {
		goldie.New(t).Assert(t, "provisioner_code_cloud_invalid", []byte(val))
	} else {
		t.Fail()
	}
}
