package letme

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	utils "github.com/lockedinspace/letme/pkg"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use: "list",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		utils.ConfigFileHealth()
	},
	Short: "List accounts.",
	Long:  `List all the AWS accounts and their main region.`,
	Run: func(cmd *cobra.Command, args []string) {
		// get the current context
		currentContext := utils.GetCurrentContext()
		letmeContext := utils.GetContextData(currentContext)
		filterTags, err := cmd.Flags().GetStringArray("filter")
		utils.CheckAndReturnError(err)
		output, err := cmd.Flags().GetString("output")
		utils.CheckAndReturnError(err)
		active, _ := cmd.Flags().GetBool("active")
		if len(filterTags) != 0 {
			letmeContext.Tags = filterTags
		}
		if active {
			activeAccountsJSON, err := utils.ActiveAccounts()
			utils.CheckAndReturnError(err)
	
			fmt.Println(activeAccountsJSON)
			os.Exit(0)
		}
		
		// fmt.Println(letmeContext.Tags)
		// os.Exit(0)

		// create a new aws session
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(letmeContext.AwsSourceProfile), config.WithRegion(letmeContext.AwsSourceProfileRegion))
		utils.CheckAndReturnError(err)
		tableData := utils.GetTableData(letmeContext.AwsDynamoDbTable, letmeContext.Tags, cfg)

		if len(tableData) == 0 {
			fmt.Println("letme: no items found that matched your filters on DynamoDB Table '" + letmeContext.AwsDynamoDbTable + "'.")
			os.Exit(1)
		}

		switch output {
		case "text":
			fmt.Println("Listing accounts using '" + currentContext + "' context:\n")
			utils.ListTextOutput(tableData)
		case "json":
			utils.ListJsonOutput(tableData)
		}
	},
}

func init() {
	var active bool
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringArray("filter", []string{}, "a comma delimited list to filter output based on tags")
	listCmd.Flags().StringP("output", "o", "text", "output results in specific format (text|json)")
	listCmd.Flags().BoolVarP(&active, "active", "", false, "list accounts which credentials are still valid")

}
