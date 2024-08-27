package config

import (
	"fmt"
	"os"
	"encoding/json"
	utils "github.com/lockedinspace/letme/pkg"
	"github.com/spf13/cobra"
)

var GetContexts = &cobra.Command{
	Use: "get-contexts",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		utils.LetmeConfigCreate()
		utils.ConfigFileHealth()
	},
	Short: "Get active and available contexts.",
	Long:  `List all configured contexts in your letme-config file marking the active context with '*'`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		output, err := cmd.Flags().GetString("output")
		utils.CheckAndReturnError(err)
		active, _ := cmd.Flags().GetBool("active-values")
		if active {
			config := utils.GetContextData(utils.GetCurrentContext())
			jsonData, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(jsonData))
			os.Exit(0)
		}
		switch output {
		case "text":
			contexts := utils.GetAvalaibleContexts()
			fmt.Println("Active context marked with '*': ")
			for _, context := range contexts {
				currentContext := utils.GetCurrentContext()
				if context == currentContext {
					fmt.Println("* " + context)
				} else {
					fmt.Println("  " + context)
				}
			}
		case "json":
			contexts := utils.GetAvalaibleContexts()
			currentContext := utils.GetCurrentContext()
			utils.ContextJsonOutput(contexts, currentContext)
		}
		
	},
}

func init() {
	var active bool
	ConfigCmd.AddCommand(GetContexts)
	GetContexts.Flags().StringP("output", "o", "text", "output results in specific format (text|json)")
	GetContexts.Flags().BoolVarP(&active, "active-values", "", false, "show active context configuration values")


}
