package config

import (
	utils "github.com/lockedinspace/letme/pkg"
	letme "github.com/lockedinspace/letme/pkg/cmd"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use: "config",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		utils.LetmeConfigCreate()
		utils.ConfigFileHealth()
	},
	Short: "Configure letme",
	Long:  `Personalize your letme experience, manage contexts and more.`,
	// Args: cobra.MinimumNFlags(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	letme.RootCmd.AddCommand(ConfigCmd)
}
