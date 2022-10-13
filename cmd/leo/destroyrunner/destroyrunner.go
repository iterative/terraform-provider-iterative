package destroyrunner

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-testing-interface"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/iterative/aws"
	"terraform-provider-iterative/iterative/azure"
	"terraform-provider-iterative/iterative/gcp"
	"terraform-provider-iterative/iterative/kubernetes"

	"terraform-provider-iterative/task/common"
)

type Options struct {
	Name string
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:    "destroy-runner <identifier>",
		Short:  "Destroy a CML runner",
		Long:   ``,
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().StringVar(&o.Name, "name", "", "needed for Google Cloud runners") // FIXME: it shouldn't

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	r := map[string]interface{}{
		"region": string(cloud.Region),
		"name":   o.Name,
	}
	s := map[string]*schema.Schema{
		"region": {Type: schema.TypeString},
		"name":   {Type: schema.TypeString},
	}

	d := schema.TestResourceDataRaw(&testing.RuntimeT{}, s, r)
	d.SetId(args[0])

	switch cloud.Provider {
	case common.ProviderAWS:
		return aws.ResourceMachineDelete(cmd.Context(), d, nil)
	case common.ProviderGCP:
		return gcp.ResourceMachineDelete(cmd.Context(), d, nil)
	case common.ProviderAZ:
		return azure.ResourceMachineDelete(cmd.Context(), d, nil)
	case common.ProviderK8S:
		return kubernetes.ResourceMachineDelete(cmd.Context(), d, nil)
	}

	return fmt.Errorf("unknown provider: %#v", cloud.Provider)
}
