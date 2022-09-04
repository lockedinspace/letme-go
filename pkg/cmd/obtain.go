package letme

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/lockedinspace/letme-go/pkg"
	"github.com/spf13/cobra"
	"os"
	"regexp"
	"bufio"
	

)

var obtainCmd = &cobra.Command{
	Use: "obtain",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(utils.GetHomeDirectory() + "/.letme/letme-config"); err == nil {
		} else {
			fmt.Println("letme: Could not locate any config file. Please run 'letme config-file' to create one.")
			os.Exit(1)
		}		
	},
	Short: "Obtain aws credentials",
	Long: `Through the AWS Security Token Service, obtain temporal credentials
once the user successfully authenticates itself. Credentials will last 3600 seconds
and can be used with the argument '--profile example1' within the aws cli binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		profile := utils.ConfigFileResultString("Aws_source_profile")
		region := utils.ConfigFileResultString("Aws_source_profile_region")
		sesAws, err := session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewSharedCredentials("", profile),
		})
		utils.CheckAndReturnError(err)
		_, err = sesAws.Config.Credentials.Get()
		utils.CheckAndReturnError(err)
		if utils.CacheFileExists() {
			//fmt.Println(strings.Split(utils.CacheFileRead(), ","))
			accountExists, err := regexp.MatchString("\\b" + args[0] + "\\b", utils.CacheFileRead())
			utils.CheckAndReturnError(err)
			if accountExists {
				file, err := os.Open(utils.GetHomeDirectory() + "/.letme/.letme-cache")
				utils.CheckAndReturnError(err)
				defer file.Close()
				scanner := bufio.NewScanner(file)
				//var fullLineR string
				for scanner.Scan() {
					a := scanner.Text()
					fullLine, err := regexp.MatchString("\\b" + args[0] + "\\b", a) 
					utils.CheckAndReturnError(err)
					if fullLine {
						//fullLineR = a 
					}
				}
				//fmt.Printf(fullLineR)
				testvar := utils.ParseCacheFile(args[0])
				fmt.Println(testvar.Role)
				/* retrieveCacheFields := strings.Split(fullLineR, ",")
				fmt.Println(retrieveCacheFields) */
			} else {
				fmt.Printf("letme: account '" + args[0] + "' not found on your cache file. Try running 'letme init' to create a new updated cache file\n")
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(obtainCmd)
}
