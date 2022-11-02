package letme

import (
	"bufio"
	"fmt"
	"github.com/lockedinspace/letme/pkg"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var configFileCmd = &cobra.Command{
	Use:   "config-file",
	Short: "Create a configuration file where parameters such as MFA arn are stored and used afterwards.",
	Long: `Creates a toml template with all the key-value pairs needed by letme.
The config file is created in '$HOME/.letme/letme-config', letme needs this file
to perform aws calls. Once created, you will need to manually edit that file and fill it with 
your values.
        `,
	Run: func(cmd *cobra.Command, args []string) {

		// grab and define force flag & verify flags
		forceFlag, _ := cmd.Flags().GetBool("force")
		verifyFlag, _ := cmd.Flags().GetBool("verify")

		// define file name and grab user home directory
		fileName := "letme-config"
		homeDir := utils.GetHomeDirectory()

		// if verify flag is passed, verify the letme-config file
		if verifyFlag {
			result := utils.CheckConfigFile(utils.GetHomeDirectory() + "/.letme/letme-config")
			if result {
				os.Exit(0)
			} else {
				fmt.Printf(
					`
letme: config file should have the following structure:
%v
`, utils.TemplateConfigFile())
				os.Exit(1)
			}
		}

		// creates the directory + config file or just the config file if the directory already exists
		// then writes the marshalled values on a toml document (letme-config).
		if _, err := os.Stat(homeDir + "/.letme/"); err != nil {
			err = os.Mkdir(homeDir+"/.letme/", 0700)
			utils.CheckAndReturnError(err)

			configFile, err := os.Create(filepath.Join(homeDir+"/.letme/", filepath.Base(fileName)))
			utils.CheckAndReturnError(err)
			defer configFile.Close()

			writer := bufio.NewWriter(configFile)
			_, err = fmt.Fprintf(writer, "%v", utils.TemplateConfigFile())
			utils.CheckAndReturnError(err)
			writer.Flush()
			fmt.Println("letme: edit the config file at " + homeDir + "/.letme/letme-config with your values.")
		} else if _, err := os.Stat(homeDir + "/.letme/"); err == nil {
			if _, err = os.Stat(homeDir + "/.letme/" + fileName); err == nil && !(forceFlag) {
				fmt.Println("letme: letme-config file already exists at: " + homeDir + "/.letme/" + fileName)
				fmt.Println("letme: to restore the letme-config file, pass the -f, --force flags or delete the letme-config file manually.")
				fmt.Println("letme: discover more flags with -h flag.")
				os.Exit(0)
			}
			configFile, err := os.Create(filepath.Join(homeDir+"/.letme/", filepath.Base(fileName)))
			utils.CheckAndReturnError(err)
			defer configFile.Close()

			writer := bufio.NewWriter(configFile)
			_, err = fmt.Fprintf(writer, "%v", utils.TemplateConfigFile())
			utils.CheckAndReturnError(err)
			writer.Flush()
			fmt.Println("letme: edit the config file at " + homeDir + "/.letme/letme-config with your values.")
		}
	},
}

func init() {

	// define a Region boolean variable
	var Force bool
	var Verify bool
	rootCmd.AddCommand(configFileCmd)

	// create a local force flag
	configFileCmd.Flags().BoolVarP(&Force, "force", "f", false, "bypass safety restrictions and force a command to be run")
	configFileCmd.Flags().BoolVarP(&Verify, "verify", "", false, "verify config file structure and integrity")
}
