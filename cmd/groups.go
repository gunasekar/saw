package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/TylerBrock/saw/blade"
	"github.com/TylerBrock/saw/config"
	"github.com/spf13/cobra"
)

// TODO: colorize based on logGroup prefix (/aws/lambda, /aws/kinesisfirehose, etc...)
var groupsConfig config.Configuration

var groupsCommand = &cobra.Command{
	Use:   "groups",
	Short: "List log groups",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		b, err := blade.NewBlade(&groupsConfig, &awsConfig, nil)
		if err != nil {
			fmt.Println("Error creating blade:", err)
			os.Exit(1)
		}

		logGroups := b.GetLogGroups(ctx)
		for _, group := range logGroups {
			fmt.Println(*group.LogGroupName)
		}
	},
}

func init() {
	groupsCommand.Flags().StringVar(&groupsConfig.Prefix, "prefix", "", "log group prefix filter")
}
