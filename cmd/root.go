package cmd

import (
	"os"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/sirupsen/logrus"
	
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
	cfgFile string
	region string
	provider string
	log string
)

var cloud = common.Cloud{
	Timeouts: common.Timeouts{
		Create: 15*time.Minute,
		Read:   3*time.Minute,
		Update: 3*time.Minute,
		Delete: 15*time.Minute,
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cwd, err := os.Getwd()
	cobra.CheckErr(err)
	viper.AddConfigPath(cwd)
	viper.SetConfigType("yaml")
	viper.SetConfigName("task.yaml")
	viper.SetEnvPrefix("task")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Reading configuration from", viper.ConfigFileUsed())
	}

	for _, newFunc := range []func(cloud *common.Cloud) *cobra.Command {
		create.New,
		delete.New,
		list.New,
		read.New,
	}{
		cmd := newFunc(&cloud)
		rootCmd.AddCommand(cmd)

		viper.BindPFlags(cmd.Flags())
		viper.BindPFlags(cmd.PersistentFlags())

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
				cmd.Flags().Set(f.Name, viper.GetString(f.Name))
			}
		})

		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
				cmd.PersistentFlags().Set(f.Name, viper.GetString(f.Name))
			}
		})
	}
	
	cobra.OnInitialize(func(){
		switch log {
		case "info":
			logrus.SetLevel(logrus.InfoLevel)
		case "debug":
			logrus.SetLevel(logrus.DebugLevel)
		}
		
		cloud.Provider = common.Provider(provider)
		cloud.Region = common.Region(region)
	})

	rootCmd.PersistentFlags().StringVar(&log, "log", "info", "log level")
	rootCmd.PersistentFlags().StringVar(&region, "region", "us-east", "cloud region")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "", "cloud provider")
	rootCmd.MarkPersistentFlagRequired("provider")

	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			rootCmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})

	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			rootCmd.PersistentFlags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}