package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"terraform-provider-iterative/iterative"
	"terraform-provider-iterative/iterative/utils"
	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

func main() {
	defer utils.WaitForAnalyticsAndHandlePanics()

	if identifier := os.Getenv("TPI_TASK_IDENTIFIER"); identifier != "" {
		provider := os.Getenv("TPI_TASK_CLOUD_PROVIDER")
		region := os.Getenv("TPI_TASK_CLOUD_REGION")
		log.Printf("[INFO] Stopping task %s...\n", identifier)
		if err := stop(context.TODO(), provider, region, common.Identifier(identifier)); err != nil {
			log.Printf("[INFO] Failed to stop task: %s\n", err.Error())
		} else {
			log.Printf("[INFO] Done!\n")
		}
		return
	}

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return iterative.Provider()
		},
	})
}

func stop(ctx context.Context, provider, region string, identifier common.Identifier) error {
	c := common.Cloud{
		Provider: common.Provider(provider),
		Region:   common.Region(region),
		Timeouts: common.Timeouts{
			Create: 10 * time.Minute,
			Read:   10 * time.Minute,
			Update: 10 * time.Minute,
			Delete: 10 * time.Minute,
		},
	}

	t, err := task.New(ctx, c, identifier, common.Task{})
	if err != nil {
		return err
	}

	return t.Stop(ctx)
}
