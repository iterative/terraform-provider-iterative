package cmd

import (
	"fmt"
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
)

var rootCmd = &cobra.Command{
	Use:   "task",
	Short: "Run code in the cloud",
	Long: `Task is a command-line tool that allows
data scientists to run code in the cloud.`,
}

var (
	cfgFile  string
	region   string
	provider string
	log      string
)

var cloud = common.Cloud{
	Timeouts: common.Timeouts{
		Create: 15 * time.Minute,
		Read:   3 * time.Minute,
		Update: 3 * time.Minute,
		Delete: 15 * time.Minute,
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(create.New(&cloud))
	rootCmd.AddCommand(delete.New(&cloud))
	rootCmd.AddCommand(list.New(&cloud))
	rootCmd.AddCommand(read.New(&cloud))

	rootCmd.MarkPersistentFlagRequired("cloud")
	rootCmd.PersistentFlags().StringVar(&provider, "cloud", "", "cloud provider")
	rootCmd.PersistentFlags().StringVar(&log, "log", "info", "log level")
	rootCmd.PersistentFlags().StringVar(&region, "region", "us-east", "cloud region")

	cwd, err := os.Getwd()
	cobra.CheckErr(err)
	viper.AddConfigPath(cwd)
	viper.SetConfigType("hcl")
	viper.SetConfigName("main.tf")
	viper.SetEnvPrefix("task")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Reading configuration from", viper.ConfigFileUsed())
	}

	// https://github.com/spf13/viper/issues/1350; should be done using viper.Sub("resource.0...")
	if resources := viper.Get("resource"); resources != nil {
		for _, resource := range resources.([]map[string]interface{}) {
			if tasks, ok := resource["iterative_task"]; ok {
				for _, task := range tasks.([]map[string]interface{}) {
					for _, block := range task {
						for _, options := range block.([]map[string]interface{}) {
							for _, option := range []string{
								"image",
								"machine",
								"name",
								"parallelism",
								"permission_set",
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

	for _, cmd := range append(rootCmd.Commands(), rootCmd) {
		viper.BindPFlags(cmd.Flags())
		viper.BindPFlags(cmd.PersistentFlags())

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if val := viper.GetString(f.Name); viper.IsSet(f.Name) && val != "" {
				cmd.Flags().Set(f.Name, val)
			}
		})

		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if val := viper.GetString(f.Name); viper.IsSet(f.Name) && val != "" {
				cmd.PersistentFlags().Set(f.Name, val)
			}
		})
	}

	cobra.OnInitialize(func() {
		switch log {
		case "info":
			logrus.SetLevel(logrus.InfoLevel)
		case "debug":
			logrus.SetLevel(logrus.DebugLevel)
		}

		cloud.Provider = common.Provider(provider)
		cloud.Region = common.Region(region)
	})
}
