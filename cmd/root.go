package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"terraform-provider-iterative/task/common"

	"terraform-provider-iterative/cmd/create"
	"terraform-provider-iterative/cmd/delete"
	"terraform-provider-iterative/cmd/list"
	"terraform-provider-iterative/cmd/read"
	"terraform-provider-iterative/cmd/stop"
)

type Options struct {
	Region   string
	Provider string
	Log      string
	common.Cloud
}

func Execute() {
	cmd := New()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func New() *cobra.Command {
	o := Options{
		Cloud: common.Cloud{
			Timeouts: common.Timeouts{
				Create: 15 * time.Minute,
				Read:   3 * time.Minute,
				Update: 3 * time.Minute,
				Delete: 15 * time.Minute,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "task",
		Short: "Run code in the cloud",
		Long: `Task is a command-line tool that allows
	data scientists to run code in the cloud.`,
	}

	cmd.AddCommand(create.New(&o.Cloud))
	cmd.AddCommand(delete.New(&o.Cloud))
	cmd.AddCommand(list.New(&o.Cloud))
	cmd.AddCommand(read.New(&o.Cloud))
	cmd.AddCommand(stop.New(&o.Cloud))

	cmd.PersistentFlags().StringVar(&o.Provider, "cloud", "", "cloud provider")
	cmd.PersistentFlags().StringVar(&o.Log, "log", "info", "log level")
	cmd.PersistentFlags().StringVar(&o.Region, "region", "us-east", "cloud region")
	cobra.CheckErr(cmd.MarkPersistentFlagRequired("cloud"))

	cobra.OnInitialize(func() {
		switch o.Log {
		case "info":
			logrus.SetLevel(logrus.InfoLevel)
		case "debug":
			logrus.SetLevel(logrus.DebugLevel)
		}

		o.Cloud.Provider = common.Provider(o.Provider)
		o.Cloud.Region = common.Region(o.Region)
	})

	cwd, err := os.Getwd()
	cobra.CheckErr(err)
	viper.AddConfigPath(cwd)
	viper.SetConfigType("hcl")
	viper.SetConfigName("main.tf")
	viper.SetEnvPrefix("task")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logrus.Errorf("error reading configuration from %s: %s", viper.ConfigFileUsed(), err.Error())
		}
	}

	// https://github.com/spf13/viper/issues/1350; should be done using viper.Sub("resource.0...")
	if resources := viper.Get("resource"); resources != nil {
		for _, resource := range resources.([]map[string]interface{}) {
			if tasks, ok := resource["iterative_task"]; ok {
				for _, task := range tasks.([]map[string]interface{}) {
					for _, block := range task {
						for _, options := range block.([]map[string]interface{}) {
							for _, option := range []string{
								"cloud",
								"image",
								"log",
								"machine",
								"name",
								"parallelism",
								"permission_set",
								"region",
								"script",
								"spot",
								"disk_size",
								"timeout",
							} {
								if value, ok := options[option]; ok {
									viper.Set(strings.ReplaceAll(option, "_", "-"), value)
								}
							}
							for _, option := range []string{
								"tags",
								"environment",
							} {
								if value, ok := options[option]; ok {
									for _, nestedBlock := range value.([]map[string]interface{}) {
										viper.Set(option, nestedBlock)
									}
								}
							}
							if value, ok := options["storage"]; ok {
								for _, nestedBlock := range value.([]map[string]interface{}) {
									if value, ok := nestedBlock["output"]; ok {
										viper.Set("output", value)
									}
									if value, ok := nestedBlock["workdir"]; ok {
										viper.Set("workdir", value)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	for _, cmd := range append(cmd.Commands(), cmd) {
		cobra.CheckErr(viper.BindPFlags(cmd.Flags()))
		cobra.CheckErr(viper.BindPFlags(cmd.PersistentFlags()))

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if val := viper.GetString(f.Name); viper.IsSet(f.Name) && val != "" {
				cobra.CheckErr(cmd.Flags().Set(f.Name, val))
			}
		})

		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if val := viper.GetString(f.Name); viper.IsSet(f.Name) && val != "" {
				cobra.CheckErr(cmd.PersistentFlags().Set(f.Name, val))
			}
		})
	}

	return cmd
}
