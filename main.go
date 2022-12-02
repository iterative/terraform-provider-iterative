package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/iterative/terraform-provider-iterative/iterative"
	"github.com/iterative/terraform-provider-iterative/iterative/utils"
)

func main() {
	defer utils.WaitForAnalyticsAndHandlePanics()
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return iterative.Provider()
		},
	})
}
