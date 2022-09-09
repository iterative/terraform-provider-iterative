package common

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"terraform-provider-iterative/task/common"
)

// BaseOptions specify base flags for commands that interact with
// cloud deployments.
type BaseOptions struct {
	Region   string
	Provider string
	Log      string
}

// defaultCloud specifies default timeouts.
var defaultCloud = common.Cloud{
	Timeouts: common.Timeouts{
		Create: 15 * time.Minute,
		Read:   3 * time.Minute,
		Update: 3 * time.Minute,
		Delete: 15 * time.Minute,
	},
}

// SetFlags sets base option flags on the provided flagset.
func (o *BaseOptions) SetFlags(f *pflag.FlagSet, cmd *cobra.Command) {
	f.StringVar(&o.Provider, "cloud", "", "cloud provider")
	f.StringVar(&o.Log, "log", "info", "log level")
	f.StringVar(&o.Region, "region", "us-east", "cloud region")
	cmd.MarkFlagRequired("cloud")
}

// GetCloud parses cloud-specific options and returns a cloud structure.
func (o *BaseOptions) GetCloud() *common.Cloud {
	cloud := defaultCloud
	cloud.Provider = common.Provider(o.Provider)
	cloud.Region = common.Region(o.Region)
	return &cloud
}

// ConfigureLogging configures logging and sets the log level.
func (o *BaseOptions) ConfigureLogging() {
	switch o.Log {
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})
}

// Initialize processes the options, the function can be used with `cobra.OnInitialize`.
func (o *BaseOptions) Initialize() {

}
