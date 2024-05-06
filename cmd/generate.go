package cmd

import (
	"openkeeper/config"
	"openkeeper/openkeeper"

	"github.com/spf13/cobra"
)

/*
   openkeeper generate --oas spec.yaml --toml rules.toml
*/

const (
	FlagConfig = "config"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate ruleset",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cmd.Flags().GetString(FlagConfig)
		if err != nil {
			return err
		}
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		return openkeeper.Process(cfg)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().SortFlags = false
	generateCmd.Flags().String(FlagConfig, "", "Set configuration file location")
}
