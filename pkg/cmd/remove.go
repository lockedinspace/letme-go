package letme

import (
	"errors"
	"fmt"
	"os"

	utils "github.com/lockedinspace/letme/pkg"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Locally remove the credentials.",
	Long: `Remove the credentials entry in your AWS files.
This will not remove anything on the DynamoDB side. Use it for
cleanup purposes and sanitizing your '$HOME/.aws/credentials'
and '$HOME/.aws/config' files.
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// read both awscredentials and config files
		credentials := utils.AwsCredsFileReadV2()
		config := utils.AwsConfigFileReadV2()

		// open both files and check if there's any error opening them, if not, delete entries based on what's existing
		_, errCred := os.OpenFile(utils.GetHomeDirectory()+"/.aws/credentials", os.O_RDWR|os.O_APPEND, 0600)
		_, errConf := os.OpenFile(utils.GetHomeDirectory()+"/.aws/config", os.O_RDWR|os.O_APPEND, 0600)

		if !(errors.Is(errCred, os.ErrNotExist)) && !(errors.Is(errConf, os.ErrNotExist)) {
			accountInFile := utils.CheckAccountLocally(args[0])
			switch {
			case accountInFile["credentials"]:
				credentialSection, err := credentials.GetSection(args[0])
				utils.CheckAndReturnError(err)
				if credentialSection.Comment != "; letme managed" {
					err := fmt.Errorf("letme: account " + args[0] + " is not managed by letme, cannot be deleted.")
					utils.CheckAndReturnError(err)
				}
				credentials.DeleteSection(args[0])
				if err := credentials.SaveTo(utils.GetHomeDirectory() + "/.aws/credentials"); err != nil {
					utils.CheckAndReturnError(err)
				}
				fmt.Println("letme: removed profile '" + args[0] + "' entry from credentials file.")
				fallthrough
			case accountInFile["config"]:
				configSection, err := config.GetSection("profile " + args[0])
				utils.CheckAndReturnError(err)
				if configSection.Comment != "; letme managed" {
					err := fmt.Errorf("letme: account " + args[0] + " is not managed by letme, cannot be deleted.")
					utils.CheckAndReturnError(err)
				}
				config.DeleteSection("profile " + args[0])
				if err := config.SaveTo(utils.GetHomeDirectory() + "/.aws/config"); err != nil {
					utils.CheckAndReturnError(err)
				}
				fmt.Println("letme: removed profile '" + args[0] + "' entry from config file.")
			default:
				fmt.Println("letme: unable to remove profile '" + args[0] + "', not found on your local aws files.")
				os.Exit(1)
			}
			utils.RemoveAccountFromDatabaseFile(args[0])

		}
	},
}

func init() {
	RootCmd.AddCommand(removeCmd)
}
