package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Execute() {
	cmd := New()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Run code in the cloud",
		Long: `Task is a command-line tool that allows
	data scientists to run code in the cloud.`,
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDestroyCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newStopCmd())

	cobra.CheckErr(parseConfigFile)

	return cmd
}

func parseConfigFile(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

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

	for _, subcmd := range append(cmd.Commands(), cmd) {
		for _, flagSet := range []*pflag.FlagSet{
			subcmd.PersistentFlags(),
			subcmd.Flags(),
		} {
			cobra.CheckErr(viper.BindPFlags(flagSet))
			flagSet.VisitAll(func(f *pflag.Flag) {
				if viper.IsSet(f.Name) {
					switch val := viper.Get(f.Name).(type) {
					case map[string]interface{}:
						for k, v := range val {
							cobra.CheckErr(flagSet.Set(f.Name, fmt.Sprintf("%s=%s", k, v)))
						}
					default:
						cobra.CheckErr(flagSet.Set(f.Name, viper.GetString(f.Name)))
					}
				}
			})
		}
	}
	return nil
}
