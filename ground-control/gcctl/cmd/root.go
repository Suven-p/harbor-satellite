package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/config"
)

var appConfig *config.Config
var appConfigStore *config.FileStore

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gcctl",
	Short: "Command-line interface for Harbor Satellite Ground Control",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var configPath string
		if str, _ := cmd.Flags().GetString("config"); str != "" {
			configPath = str
		}
		configPath, err := config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to determine default config path: %w", err)
		}

		appConfig = &config.Config{}
		appConfigStore, err = config.NewFileStore(configPath)
		if err != nil {
			return fmt.Errorf("failed to create config store: %w", err)
		}
		err = appConfigStore.Load(appConfig)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "Path to configuration file.")
}
