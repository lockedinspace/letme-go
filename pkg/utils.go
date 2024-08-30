package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/ini.v1"
)

// Expected keys in letme-config file
var ExpectedKeys = map[string]bool{
	"aws_source_profile":        true,
	"aws_source_profile_region": true,
	"dynamodb_table":            true,
	"mfa_arn":                   true,
	"session_name":              true,
	"session_duration":          true,
	"tags":                      true,
}

// Mandatory keys in letme-config file
var MandatoryKeys = []string{
	"aws_source_profile",
	"aws_source_profile_region",
	"dynamodb_table",
}

type Context struct {
	ActiveContext string `ini:"active_context"`
}

type LetmeContext struct {
	AwsSourceProfile       string   `ini:"aws_source_profile"`
	AwsSourceProfileRegion string   `ini:"aws_source_profile_region"`
	AwsDynamoDbTable       string   `ini:"dynamodb_table"`
	AwsMfaArn              string   `ini:"mfa_arn"`
	AwsSessionName         string   `ini:"session_name"`
	AwsSessionDuration     int32    `ini:"session_duration"`
	Tags                   []string `ini:"tags"`
}

type DynamoDbAccountConfig struct {
	Name   string   `dynamodbav:"name"`
	Region []string `dynamodbav:"region"`
	Role   []string `dynamodbav:"role"`
	Tags   []string `dynamodbav:"tags"`
}

type AccountItem struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

type AccountItems struct {
	Items []AccountItem `json:"items"`
}

type ContextItem struct {
	Name   string `json:"name"`
	Active string `json:"active,omitempty""`
}
type ContextsItems struct {
	Items  []ContextItem `json:"items"`
}

type ProfileConfig struct {
	Output string `ini:"output"`
	Region string `ini:"region"`
}

type ProfileCredential struct {
	AccessKey    string `ini:"aws_access_key_id"`
	SecretKey    string `ini:"aws_secret_access_key"`
	SessionToken string `ini:"aws_session_token"`
}

// Verify if the config-file respects the struct LetmeContext
func CheckConfigFile(path string) bool {
	filePath := GetHomeDirectory() + "/.letme/letme-config"

	// Check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		CheckAndReturnError(err)
	}

	config, err := ini.Load(filePath)
	CheckAndReturnError(err)

	sections := config.Sections()

	for _, section := range sections {
		if section.Name() == "DEFAULT" {
			continue
		}
		for _, key := range MandatoryKeys {
			if ok := section.HasKey(key); !ok {
				fmt.Printf("letme: missing mandatory key '%s' in table '%s'. Config file should have the following structure:\n", key, section.Name())
				return false
			}
		}

		for _, key := range section.KeyStrings() {
			if !ExpectedKeys[key] {
				fmt.Printf("letme: invalid key '%s' in table '%s'. Config file should have the following structure:\n", key, section.Name())
				return false
			}
		}
	}

	return true
}

// Check if a command exists
func CommandExists(command string) {
	_, err := exec.LookPath(command)
	CheckAndReturnError(err)
}

// Checks the error, if the error contains a message, stop the execution and show the error to the user
func CheckAndReturnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Marshalls data into a toml file (config-file)
func TemplateConfigFile(stdout bool) {
	template := ini.Empty()

	section, err := template.NewSection("contextName")
	CheckAndReturnError(err)

	data := &LetmeContext{
		AwsSourceProfile:       "default",
		AwsSourceProfileRegion: "eu-west-3",
		AwsDynamoDbTable:       "customers",
		AwsMfaArn:              "arn:aws:iam::4002019901:mfa/user",
		AwsSessionDuration:     3600,
		AwsSessionName:         "user_letme",
	}

	err = section.ReflectFrom(data)
	CheckAndReturnError(err)

	if stdout {
		_, err = template.WriteTo(os.Stdout)
		CheckAndReturnError(err)
		os.Exit(1)
	} else {
		fileName := "letme-config"
		homeDir := GetHomeDirectory()
		configFile, err := os.Create(filepath.Join(homeDir+"/.letme", filepath.Base(fileName)))
		CheckAndReturnError(err)
		defer configFile.Close()
		_, err = template.WriteTo(configFile)
		CheckAndReturnError(err)
	}
}

func mfaArnInput(awsProfile string, awsRegion string) string {
	var mfaArn string
	mfaArnRegex := `^arn:aws:iam::[0-9]{12}:mfa\/[\S]+$`
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsProfile), config.WithRegion(awsRegion))
	CheckAndReturnError(err)

	sesIam := iam.NewFromConfig(cfg)

	currentUser, err := sesIam.GetUser(context.TODO(), &iam.GetUserInput{})
	CheckAndReturnError(err)

	iamResp, err := sesIam.ListMFADevices(context.TODO(), &iam.ListMFADevicesInput{
		UserName: currentUser.User.UserName,
	})
	CheckAndReturnError(err)

	if len(iamResp.MFADevices) == 0 {
		fmt.Println("letme: no MFA devices configured.")
		return ""
	}

	var mfaDevices []string
	for _, device := range iamResp.MFADevices {
		mfaDevices = append(mfaDevices, *device.SerialNumber)
	}

	mfaArnExists := false
	for {
		fmt.Print("→ AWS MFA Device arn (optional): ")
		fmt.Scanln(&mfaArn)

		if len(mfaArn) == 0 {
			return ""
		}

		re := regexp.MustCompile(mfaArnRegex)
		switch re.MatchString(mfaArn) {
		case true:
			for _, arn := range mfaDevices {
				if arn == mfaArn {
					mfaArnExists = true
					break
				}
			}
		case false:
			fmt.Println("letme: not a valid MFA device arn. Run 'aws iam list-mfa-devices --query 'MFADevices[*].SerialNumber --profile " + awsProfile)
			continue
		}
		if !mfaArnExists {
			fmt.Println("letme: MFA Device not found. Run 'aws iam list-mfa-devices --query 'MFADevices[*].SerialNumber --profile " + awsProfile)
			continue
		}
		break
	}
	return mfaArn
}

func sourceProfileRegionInput() string {
	var awsRegion string
	fmt.Print("→ AWS Source Profile Region: ")
	fmt.Scanln(&awsRegion)
	return awsRegion
}

func sessionDurationInput() int32 {
	var sessionDuration int32
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("→ Token Session Duration in seconds (optional): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if len(input) == 0 {
			input = "3600"
		}

		duration, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("letme: expected integer not string.")
			continue
		} else if duration < 900 || duration > 43200 {
			fmt.Println("letme: token session duration cannot be lower than 15 minutes or higher than 12 hours.")
			continue
		} else {
			sessionDuration = int32(duration)
			break
		}
	}
	// fmt.Println(sessionDuration)
	return sessionDuration
}

func sessionNameInput() string {
	var sessionName string
	fmt.Print("→ Session Name (optional): ")
	fmt.Scanln(&sessionName)

	if len(sessionName) == 0 {
		return "letme_session"
	}

	return sessionName
}

func sourceProfileInput() string {
	config := AwsConfigFileReadV2()
	credentials := AwsCredsFileReadV2()
	var awsProfile string

	for {
		fmt.Print("→ AWS Source Profile Name: ")
		fmt.Scanln(&awsProfile)
		configProfileExists := false
		credentialsProfileExists := false

		if len(awsProfile) == 0 {
			fmt.Println("letme: AWS Profile Name field is required ")
			continue
		}

		if config.HasSection("profile "+awsProfile) || config.HasSection(awsProfile) {
			configProfileExists = true
		}

		if credentials.HasSection(awsProfile) {
			credentialsProfileExists = true
		}

		if !configProfileExists {
			fmt.Println("letme: profile name does not exist in your AWS config file. Specify a valid profile.")
			continue
		}

		if !credentialsProfileExists {
			fmt.Println("letme: profile name does not exist in your AWS credentials file. Specify a valid profile.")
			continue
		}
		break
	}
	return awsProfile
}

func dynamoDbTableInput(awsProfile string, awsRegion string) string {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsProfile), config.WithRegion(awsRegion))
	CheckAndReturnError(err)

	sesAwsDynamoDb := dynamodb.NewFromConfig(cfg)

	resp, err := sesAwsDynamoDb.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	CheckAndReturnError(err)
	var dynamoDbTableName string

	for {
		fmt.Print("→ AWS DynamoDB Table Name: ")
		fmt.Scanln(&dynamoDbTableName)

		if len(dynamoDbTableName) == 0 {
			fmt.Println("letme: DynamoDB Table Name field is required.")
			continue
		}

		tableExists := false
		for _, table := range resp.TableNames {
			if table == dynamoDbTableName {
				tableExists = true
			}
		}

		if !tableExists {
			fmt.Println("letme: DynamoDB Table not found.")
			continue
		}

		break
	}
	return dynamoDbTableName
}

func letmeTagsInput() []string {
	var tagInput string
	fmt.Print("→ Tags (optional): ")
	fmt.Scanln(&tagInput)

	if len(tagInput) == 0 {
		return []string{}
	}

	tags := strings.Split(tagInput, ",")

	return tags
}

func NewContext(context string, mode int) {
	if mode == 0 {
		fmt.Println("letme: creating context '" + context + "'. Optional fields can be left empty.")
	} else if mode == 1 {
		fmt.Println("letme: updating context '" + context + "'. Optional fields can be left empty.")
	}
	var letmeContext LetmeContext

	letmeContext.AwsSourceProfile = sourceProfileInput()
	letmeContext.AwsSourceProfileRegion = sourceProfileRegionInput()
	letmeContext.AwsDynamoDbTable = dynamoDbTableInput(letmeContext.AwsSourceProfile, letmeContext.AwsSourceProfileRegion)
	letmeContext.AwsMfaArn = mfaArnInput(letmeContext.AwsSourceProfile, letmeContext.AwsSourceProfileRegion)
	letmeContext.AwsSessionDuration = sessionDurationInput()
	letmeContext.AwsSessionName = sessionNameInput()
	letmeContext.Tags = letmeTagsInput()

	letmeConfig := LetmeConfigRead()

	section := letmeConfig.Section(context)

	if err := section.ReflectFrom(&letmeContext); err != nil {
		CheckAndReturnError(err)
	}

	if len(letmeContext.AwsMfaArn) == 0 {
		section.DeleteKey("mfa_arn")
	}

	if len(letmeContext.Tags) == 0 {
		section.DeleteKey("tags")
	}
	letmeConfig.SaveTo(GetHomeDirectory() + "/.letme/letme-config")
}

// Gets the user $HOME directory
func GetHomeDirectory() string {
	homeDir, err := os.UserHomeDir()
	CheckAndReturnError(err)
	return homeDir
}

// Checks if the .letme-cache file exists, this file is not supported starting from versions 0.2.0 and above
func CacheFileExists() bool {
	if _, err := os.Stat(GetHomeDirectory() + "/.letme/.letme-cache"); err == nil {
		return true
	} else {
		return false
	}
}

// Marshalls data into a string used for the aws config file but with the v1 output protocol
func AwsConfigFileCredentialsProcessV1(accountName string, region string) {
	credentials := AwsCredsFileReadV2()
	config := AwsConfigFileReadV2()

	accountInFile := CheckAccountLocally(accountName)
	switch {
	case accountInFile["credentials"]:
		credentials.DeleteSection(accountName)
		err := credentials.SaveTo(GetHomeDirectory() + "/.aws/credentials")
		fmt.Println("letme: removed profile '" + accountName + "' entry from credentials file.")
		CheckAndReturnError(err)
		fallthrough
	case accountInFile["config"] && !config.Section("profile "+accountName).HasKey("credential_process"):
		_, err := config.Section("profile "+accountName).NewKey("credential_process", "letme obtain "+accountName+" --v1")
		CheckAndReturnError(err)
		err = config.SaveTo(GetHomeDirectory() + "/.aws/config")
		CheckAndReturnError(err)
	default:
		section, errSection := config.NewSection("profile " + accountName)
		CheckAndReturnError(errSection)
		_, errCredentialProcess := section.NewKey("credential_process", "letme obtain "+accountName+" --v1")
		CheckAndReturnError(errCredentialProcess)
		_, errRegion := section.NewKey("region", region)
		CheckAndReturnError(errRegion)
		_, errOutput := section.NewKey("output", "json")
		CheckAndReturnError(errOutput)
		err := config.SaveTo(GetHomeDirectory() + "/.aws/config")
		CheckAndReturnError(err)
	}

	fmt.Println("letme: configured credential process V1 for account " + accountName)
	fmt.Println("letme: use the argument '--profile " + accountName + "' to interact with the account.")
	os.Exit(0)
}

// Check if an account is present on the local aws credentials/config files
func CheckAccountLocally(account string) map[string]bool {
	credentials := AwsCredsFileReadV2()
	config := AwsConfigFileReadV2()

	accountInFile := make(map[string]bool)

	accountInFile["credentials"] = false
	accountInFile["config"] = false

	if credentials.HasSection(account) {
		accountInFile["credentials"] = true
	}

	if config.HasSection("profile " + account) {
		accountInFile["config"] = true
	}

	return accountInFile
}

// Struct which states the credential process output for the v1 protocol
type CredentialsProcess struct {
	Version         int
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// Return aws credentials following the credentials_process standard
// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
func CredentialsProcessOutput(accessKeyID string, secretAccessKey string, sessionToken string, expirationTime time.Time) string {
	group := CredentialsProcess{
		Version:         1,
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Expiration:      expirationTime,
	}
	b, err := json.Marshal(group)
	CheckAndReturnError(err)
	return string(b)
}

type Dataset struct {
	Name          string `json:"name"`
	LastRequest   int64  `json:"lastRequest"`
	Expiry        int64  `json:"expiry"`
	AuthMethod    string `json:"authMethod"`
	V1Credentials string `json:"v1Credentials,omitempty"`
}
type Account struct {
	Account Dataset `json:"account"`
}

// Create a file which stores the last time when credentials where requested. Then query if the account exists,
// if not, it will create its first entry.
func DatabaseFile(accountName string, sessionDuration int32, v1Credentials string, authMethod string) {
	databaseFileWriter, err := os.OpenFile(GetHomeDirectory()+"/.letme/.letme-db", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	CheckAndReturnError(err)
	databaseFileReader, err := os.ReadFile(GetHomeDirectory() + "/.letme/.letme-db")
	CheckAndReturnError(err)
	fi, err := os.Stat(GetHomeDirectory() + "/.letme/.letme-db")
	CheckAndReturnError(err)
	var idents []Account
	if fi.Size() > 0 {
		//check if the json is valid, but ensure that the file has content
		if !json.Valid([]byte(databaseFileReader)) && fi.Size() > 0 {
			fmt.Printf("letme: " + GetHomeDirectory() + "/.letme/.letme-db" + " is not JSON valid. Remove the file and try again.\n")
			os.Exit(1)
		}
		err = json.Unmarshal(databaseFileReader, &idents)
		CheckAndReturnError(err)
		err = os.Truncate(GetHomeDirectory()+"/.letme/.letme-db", 0)
		CheckAndReturnError(err)
		for i := range idents {
			//when file is populated and client exist, just update fields
			if idents[i].Account.Name == accountName {
				idents[i].Account.LastRequest = time.Now().Unix()
				idents[i].Account.Expiry = time.Now().Add(time.Second * time.Duration(sessionDuration)).Unix()
				idents[i].Account.V1Credentials = v1Credentials
				idents[i].Account.AuthMethod = authMethod
				b, err := json.MarshalIndent(idents, "", "  ")
				CheckAndReturnError(err)

				if _, err = databaseFileWriter.WriteString(string(b)); err != nil {
					CheckAndReturnError(err)
					defer databaseFileWriter.Close()
				}
				return
			}
		}
		//when file is populated but client does not exist
		idents = append(idents, Account{Dataset{accountName, time.Now().Unix(), time.Now().Add(time.Second * time.Duration(sessionDuration)).Unix(), authMethod, v1Credentials}})
		b, err := json.MarshalIndent(idents, "", "  ")
		CheckAndReturnError(err)

		if _, err = databaseFileWriter.WriteString(string(b)); err != nil {
			CheckAndReturnError(err)
			defer databaseFileWriter.Close()
		}
		//when file does not exist neither the client
	} else if fi.Size() == 0 {
		idents = append(idents, Account{Dataset{accountName, time.Now().Unix(), time.Now().Add(time.Second * time.Duration(sessionDuration)).Unix(), authMethod, v1Credentials}})
		b, err := json.MarshalIndent(idents, "", "  ")
		CheckAndReturnError(err)

		if _, err = databaseFileWriter.WriteString(string(b)); err != nil {
			CheckAndReturnError(err)
			defer databaseFileWriter.Close()
		}
	}
}

// Compare the current local time with the expiry field in the .letme-db file. If current time has not yet surpassed
// expiry time, return true. Else, return false indicating new credentials need to be requested.
func CheckAccountAvailability(accountName string) bool {
	if _, err := os.Stat(GetHomeDirectory() + "/.letme/.letme-db"); err == nil {
		databaseFileReader, err := os.ReadFile(GetHomeDirectory() + "/.letme/.letme-db")
		CheckAndReturnError(err)
		fi, err := os.Stat(GetHomeDirectory() + "/.letme/.letme-db")
		CheckAndReturnError(err)
		if !json.Valid([]byte(databaseFileReader)) && fi.Size() > 0 {
			fmt.Printf("letme: " + GetHomeDirectory() + "/.letme/.letme-db" + " is not JSON valid. Remove the file and try again.\n")
			os.Exit(1)
		}
		var idents []Account
		json.Unmarshal(databaseFileReader, &idents) //should really check with _, err
		for i := range idents {
			if idents[i].Account.Name == accountName {
				t1 := time.Now().Unix()
				t2 := idents[i].Account.Expiry
				t3 := t2 - t1
				if t3 <= 0 {
					return false
				} else {
					return true
				}
			}
		}
	} else {
		_, err := os.OpenFile(GetHomeDirectory()+"/.letme/.letme-db", os.O_CREATE, 0600)
		CheckAndReturnError(err)
	}
	return false
}

// Check if the account to retrieve stored credentials exist, if true, return the credentials to stdout
func ReturnAccountCredentials(accountName string) map[string]string {
	databaseFileReader, err := os.ReadFile(GetHomeDirectory() + "/.letme/.letme-db")
	CheckAndReturnError(err)
	var idents []Account
	var result string
	m := make(map[string]string)
	err = json.Unmarshal(databaseFileReader, &idents)
	CheckAndReturnError(err)
	for i := range idents {
		if idents[i].Account.Name == accountName {
			result = idents[i].Account.V1Credentials
			data := CredentialsProcess{}
			json.Unmarshal([]byte(result), &data)
			m["AccessKeyId"] = data.AccessKeyId
			m["SecretAccessKey"] = data.SecretAccessKey
			m["SessionToken"] = data.SessionToken
		}
	}
	return m
}

// Remove an account from the database file
func RemoveAccountFromDatabaseFile(accountName string) {
	jsonData, err := os.ReadFile(GetHomeDirectory() + "/.letme/.letme-db")
	CheckAndReturnError(err)
	//unmarshal JSON data into a slice of maps
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		CheckAndReturnError(err)
	}

	//iterate over each object in the slice
	for i, obj := range data {
		//check if the "name" field of the "account" object is "adaral"
		if name, ok := obj["account"].(map[string]interface{})["name"].(string); ok && name == accountName {
			//remove the object from the slice
			data = append(data[:i], data[i+1:]...)
			break //break after removing to avoid index out of range error
		}
	}

	updatedJsonData, err := json.MarshalIndent(data, "", "  ")
	CheckAndReturnError(err)

	//write the prettified JSON data to the file /test.json
	if err := os.WriteFile(GetHomeDirectory()+"/.letme/.letme-db", updatedJsonData, 0600); err != nil {
		CheckAndReturnError(err)
	}
}

func AwsCredsFileReadV2() *ini.File {
	awsCredentialsFile, err := ini.Load(GetHomeDirectory() + "/.aws/credentials")
	CheckAndReturnError(err)
	return awsCredentialsFile
}

func AwsConfigFileReadV2() *ini.File {
	awsCredentialsFile, err := ini.Load(GetHomeDirectory() + "/.aws/config")
	CheckAndReturnError(err)
	return awsCredentialsFile
}

func LetmeConfigCreate() {
	filePath := GetHomeDirectory() + "/.letme/letme-config"
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		letmeConfigFileCreate, err := os.Create(filePath)
		CheckAndReturnError(err)
		defer letmeConfigFileCreate.Close()
	} else {
		CheckAndReturnError(err)
	}
}

func LetmeConfigRead() *ini.File {

	filePath := GetHomeDirectory() + "/.letme/letme-config"
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		letmeConfigFileCreate, err := os.Create(filePath)
		CheckAndReturnError(err)
		defer letmeConfigFileCreate.Close()
	}

	if err != nil && !os.IsNotExist(err) {
		CheckAndReturnError(err)
	}

	letmeConfigFile, err := ini.Load(filePath)
	CheckAndReturnError(err)

	return letmeConfigFile
}

func LoadAwsCredentials(profileName string, profileCredential ProfileCredential) {
	credentialsFile := AwsCredsFileReadV2()

	credentialsSection := credentialsFile.Section(profileName)
	credentialsSection.Comment = "letme managed"

	if err := credentialsSection.ReflectFrom(&profileCredential); err != nil {
		CheckAndReturnError(err)
	}

	if err := credentialsFile.SaveTo(GetHomeDirectory() + "/.aws/credentials"); err != nil {
		CheckAndReturnError(err)
	}
}

func LoadAwsConfig(profileName string, profileConfig ProfileConfig) {
	configFile := AwsConfigFileReadV2()

	configSection := configFile.Section("profile " + profileName)
	configSection.Comment = "letme managed"
	if err := configSection.ReflectFrom(&profileConfig); err != nil {
		CheckAndReturnError(err)
	}
	if err := configFile.SaveTo(GetHomeDirectory() + "/.aws/config"); err != nil {
		CheckAndReturnError(err)
	}
}

// Create the .letme-usersettings file which holds the current context and more
func UpdateContext(context string) {
	filePath := GetHomeDirectory() + "/.letme/.letme-usersettings"

	//check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		//file exists, read the current content
		content, err := ini.Load(filePath)

		if err != nil {
			CheckAndReturnError(err)
		}

		//reads "context" section and updates "active_context" field
		contextSection := content.Section("context")
		err = contextSection.ReflectFrom(&Context{
			ActiveContext: context,
		})

		if err != nil {
			CheckAndReturnError(err)
		}

		//saves the updated data onto the file
		content.SaveTo(filePath)

	} else if os.IsNotExist(err) {
		userSettings, err := os.Create(filePath)
		CheckAndReturnError(err)
		defer userSettings.Close()
		content := ini.Empty()
		contextSection := content.Section("context")
		err = contextSection.ReflectFrom(&Context{
			ActiveContext: context,
		})
		CheckAndReturnError(err)
		_, err = content.WriteTo(userSettings)
		CheckAndReturnError(err)
		//an unexpected error occurred
	} else {
		CheckAndReturnError(err)
	}
}

func GetCurrentContext() string {
	filePath := GetHomeDirectory() + "/.letme/.letme-usersettings"

	//check if the file exists if not exists returns "general"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "general"
	}
	//read the content of the file
	content, err := ini.Load(filePath)
	content.BlockMode = false

	if err != nil {
		CheckAndReturnError(err)
	}

	//maps "context" section to settings variable and returns the active context
	settings := new(Context)
	section, err := content.GetSection("context")
	CheckAndReturnError(err)
	if err := section.MapTo(&settings); err != nil {
		CheckAndReturnError(err)
	}

	return settings.ActiveContext
}

func GetAvalaibleContexts() []string {
	filePath := GetHomeDirectory() + "/.letme/letme-config"
	content, err := ini.Load(filePath)
	// content.BlockMode = false
	if err != nil {
		CheckAndReturnError(err)
	}

	sections := content.SectionStrings()
	sortedSections := make([]string, 0, len(sections)-1)
	for _, section := range sections {
		if section == "DEFAULT" {
			continue
		}
		sortedSections = append(sortedSections, section)
	}
	sort.Strings(sortedSections)
	return sortedSections
}

func GetAccount(awsDynamoDbTable string, cfg aws.Config, profileName string) *DynamoDbAccountConfig {
	sesAwsDynamoDb := dynamodb.NewFromConfig(cfg)

	account := new(DynamoDbAccountConfig)

	resp, err := sesAwsDynamoDb.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(awsDynamoDbTable),
		Key:       map[string]dynamodbTypes.AttributeValue{"name": &dynamodbTypes.AttributeValueMemberS{Value: profileName}},
	})
	CheckAndReturnError(err)
	err = attributevalue.UnmarshalMap(resp.Item, &account)
	CheckAndReturnError(err)

	return account
}

func GetTableData(awsDynamoDbTable string, tags []string, cfg aws.Config) (resp []DynamoDbAccountConfig) {
	sesAwsDynamoDb := dynamodb.NewFromConfig(cfg)
	var accountList []DynamoDbAccountConfig

	switch {
	case len(tags) > 0:
		var filter []string
		expressionAttributeValues := make(map[string]dynamodbTypes.AttributeValue, len(tags))

		for _, tag := range tags {
			expressionAttributeValues[":"+tag] = &dynamodbTypes.AttributeValueMemberS{Value: tag}
			expression := fmt.Sprintf("contains(#tags, :%s)", tag)
			filter = append(filter, expression)
		}

		filterExpression := strings.Join(filter, " AND ")

		// fmt.Println(expressionAttributeValues, filterExpression)

		resp, err := sesAwsDynamoDb.Scan(context.TODO(), &dynamodb.ScanInput{
			TableName: aws.String(awsDynamoDbTable),
			ExpressionAttributeNames: map[string]string{
				"#tags": "tags",
			},
			ExpressionAttributeValues: expressionAttributeValues,
			FilterExpression:          aws.String(filterExpression),
		})
		CheckAndReturnError(err)
		for _, item := range resp.Items {
			var account DynamoDbAccountConfig
			err = attributevalue.UnmarshalMap(item, &account)
			CheckAndReturnError(err)
			accountList = append(accountList, account)
		}
	default:
		resp, err := sesAwsDynamoDb.Scan(context.TODO(), &dynamodb.ScanInput{
			TableName: aws.String(awsDynamoDbTable),
		})
		CheckAndReturnError(err)

		for _, item := range resp.Items {
			var account DynamoDbAccountConfig
			err = attributevalue.UnmarshalMap(item, &account)
			CheckAndReturnError(err)
			accountList = append(accountList, account)
		}
	}

	return accountList
}

func ListTextOutput(accountList []DynamoDbAccountConfig) {
	sorted := make([]string, 0, len(accountList))
	var nameLengths []int
	var w *tabwriter.Writer

	for _, account := range accountList {
		sorted = append(sorted, account.Name+"\t"+account.Region[0])
		nameLengths = append(nameLengths, len(account.Name))
	}
	sort.Ints(nameLengths)
	sort.Strings(sorted)
	w = tabwriter.NewWriter(os.Stdout, nameLengths[len(nameLengths)-1]+5, 200, 1, ' ', 0)

	fmt.Fprintln(w, "NAME:\tMAIN REGION:")
	fmt.Fprintln(w, "-----\t------------")
	for _, id := range sorted {
		fmt.Fprintln(w, id)
		w.Flush()
	}
}

func ListJsonOutput(accountList []DynamoDbAccountConfig) {
	// sorted := make([]string, 0, len(accountList))
	var accountItems AccountItems

	for _, account := range accountList {
		accountItems.Items = append(accountItems.Items, AccountItem{Name: account.Name, Region: account.Region[0]})
		// sorted = append(sorted, account.Name+"\t"+account.Region[0])
	}

	sort.Slice(accountItems.Items, func(i, j int) bool {
		return accountItems.Items[i].Name < accountItems.Items[j].Name
	})

	jsonData, err := json.MarshalIndent(accountItems, "", " ")
	CheckAndReturnError(err)
	fmt.Println(string(jsonData))
}

func ContextJsonOutput(contexts []string, currentContext string) {
	var contextsItems ContextsItems
	for _, context := range contexts {
		if context == currentContext {
			contextsItems.Items = append(contextsItems.Items, ContextItem{Name: context, Active: "true"})
		} else {
			contextsItems.Items = append(contextsItems.Items, ContextItem{Name: context})
		}
	}

	sort.Slice(contextsItems.Items, func(i, j int) bool {
		return contextsItems.Items[i].Name < contextsItems.Items[j].Name
	})
	jsonData, err := json.MarshalIndent(contextsItems, "", " ")
	CheckAndReturnError(err)
	fmt.Println(string(jsonData))
}

func GetContextData(context string) *LetmeContext {
	filePath := GetHomeDirectory() + "/.letme/letme-config"

	//check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		CheckAndReturnError(err)
	}

	//check if the file is empty
	if info, err := os.Stat(filePath); err == nil && info.Size() == 0 {
		fmt.Println("letme: no contexts found. Run 'letme config new-context ${contextName}' to create one.")
		os.Exit(1)
	}

	config, err := ini.Load(filePath)
	CheckAndReturnError(err)

	contextSection, err := config.GetSection(context)
	CheckAndReturnError(err)

	letmeContext := new(LetmeContext)
	if err := contextSection.MapTo(&letmeContext); err != nil {
		CheckAndReturnError(err)
	}

	return letmeContext
}

func AssumeRole(letmeContext *LetmeContext, cfg aws.Config, inlineTokenMfa string, account *DynamoDbAccountConfig, renew bool, localCredentialProcessFlagV1 bool, authMethod string) (ProfileCredential, ProfileConfig) {
	//if credentials not expired
	if CheckAccountAvailability(account.Name) && !localCredentialProcessFlagV1 && !renew {
		cachedCredentials := ReturnAccountCredentials(account.Name)
		profileCredential := ProfileCredential{
			AccessKey:    cachedCredentials["AccessKeyId"],
			SecretKey:    cachedCredentials["SecretAccessKey"],
			SessionToken: cachedCredentials["SessionToken"],
		}

		profileConfig := ProfileConfig{
			Output: "json",
			Region: account.Region[0],
		}

		fmt.Println("letme: using cached credentials. Append argument '--renew' to obtain new credentials.")
		return profileCredential, profileConfig
	}

	sesAwsSts := sts.NewFromConfig(cfg)
	var input *sts.AssumeRoleInput

	switch {
	case len(letmeContext.AwsMfaArn) > 0 && len(inlineTokenMfa) > 0:
		input = &sts.AssumeRoleInput{
			RoleArn:         &account.Role[0],
			RoleSessionName: &letmeContext.AwsSessionName,
			SerialNumber:    &letmeContext.AwsMfaArn,
			TokenCode:       &inlineTokenMfa,
			DurationSeconds: &letmeContext.AwsSessionDuration,
		}
	case len(letmeContext.AwsMfaArn) > 0 && len(inlineTokenMfa) <= 0:
		fmt.Printf("Enter MFA one time pass code: ")
		var tokenMfa string
		fmt.Scanln(&tokenMfa)
		input = &sts.AssumeRoleInput{
			RoleArn:         &account.Role[0],
			RoleSessionName: &letmeContext.AwsSessionName,
			SerialNumber:    &letmeContext.AwsMfaArn,
			TokenCode:       &tokenMfa,
			DurationSeconds: &letmeContext.AwsSessionDuration,
		}
	default:
		input = &sts.AssumeRoleInput{
			RoleArn:         &account.Role[0],
			RoleSessionName: &letmeContext.AwsSessionName,
			DurationSeconds: &letmeContext.AwsSessionDuration,
		}
	}

	resp, err := sesAwsSts.AssumeRole(context.TODO(), input)
	CheckAndReturnError(err)

	profileCredential := ProfileCredential{
		AccessKey:    *resp.Credentials.AccessKeyId,
		SecretKey:    *resp.Credentials.SecretAccessKey,
		SessionToken: *resp.Credentials.SessionToken,
	}

	profileConfig := ProfileConfig{
		Output: "json",
		Region: account.Region[0],
	}
	switch {
	case localCredentialProcessFlagV1:
		fmt.Printf(CredentialsProcessOutput(profileCredential.AccessKey, profileCredential.SecretKey, profileCredential.SessionToken, *resp.Credentials.Expiration))
		os.Exit(0)
	case renew || !CheckAccountAvailability(account.Name):
		DatabaseFile(account.Name, letmeContext.AwsSessionDuration, CredentialsProcessOutput(profileCredential.AccessKey, profileCredential.SecretKey, profileCredential.SessionToken, *resp.Credentials.Expiration), authMethod) //only when we really authenticate against aws
	}

	return profileCredential, profileConfig
}

func AssumeRoleChained(letmeContext *LetmeContext, cfg aws.Config, inlineTokenMfa string, account *DynamoDbAccountConfig, renew bool, localCredentialProcessFlagV1 bool, authMethod string) (ProfileCredential, ProfileConfig) {
	//if credentials not expired
	if CheckAccountAvailability(account.Name) && !localCredentialProcessFlagV1 {
		cachedCredentials := ReturnAccountCredentials(account.Name)
		profileCredential := ProfileCredential{
			AccessKey:    cachedCredentials["AccessKeyId"],
			SecretKey:    cachedCredentials["SecretAccessKey"],
			SessionToken: cachedCredentials["SessionToken"],
		}

		profileConfig := ProfileConfig{
			Output: "json",
			Region: account.Region[0],
		}

		fmt.Println("letme: using cached credentials. Append argument '--renew' to obtain new credentials.")
		return profileCredential, profileConfig
	}

	sesAwsSts := sts.NewFromConfig(cfg)
	var input *sts.AssumeRoleInput
	var output *sts.AssumeRoleOutput
	var err error

	fmt.Println("More than one role detected. Total hops:", len(account.Role))
	for i := range account.Role {
		fmt.Printf("[%v/%v]\n", i+1, len(account.Role))
		switch {
		//first hop with --inline-mfa flag
		case i == 0 && len(letmeContext.AwsMfaArn) > 0 && len(inlineTokenMfa) > 0:
			input = &sts.AssumeRoleInput{
				RoleArn:         &account.Role[i],
				RoleSessionName: &letmeContext.AwsSessionName,
				SerialNumber:    &letmeContext.AwsMfaArn,
				TokenCode:       &inlineTokenMfa,
				DurationSeconds: &letmeContext.AwsSessionDuration,
			}
			output, err = sesAwsSts.AssumeRole(context.TODO(), input)
			CheckAndReturnError(err)
		//first hop with interactive MFA token
		case i == 0 && len(letmeContext.AwsMfaArn) > 0 && len(inlineTokenMfa) <= 0:
			var tokenMfa string
			fmt.Printf("Enter MFA one time pass code: ")
			fmt.Scanln(&tokenMfa)
			input = &sts.AssumeRoleInput{
				RoleArn:         &account.Role[i],
				RoleSessionName: &letmeContext.AwsSessionName,
				SerialNumber:    &letmeContext.AwsMfaArn,
				TokenCode:       &tokenMfa,
				DurationSeconds: &letmeContext.AwsSessionDuration,
			}
			output, err = sesAwsSts.AssumeRole(context.TODO(), input)
			CheckAndReturnError(err)
		//first hop without MFA
		case i == 0:
			input = &sts.AssumeRoleInput{
				RoleArn:         &account.Role[i],
				RoleSessionName: &letmeContext.AwsSessionName,
				DurationSeconds: &letmeContext.AwsSessionDuration,
			}
			output, err = sesAwsSts.AssumeRole(context.TODO(), input)
			CheckAndReturnError(err)
		//chained AssumneRoles with credentials from previous iterations
		default:
			cfg, err := config.LoadDefaultConfig(context.TODO(),
				config.WithRegion(account.Region[0]),
				config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID: *output.Credentials.AccessKeyId, SecretAccessKey: *output.Credentials.SecretAccessKey, SessionToken: *output.Credentials.SessionToken,
					},
				}))
			CheckAndReturnError(err)
			sesChainedSts := sts.NewFromConfig(cfg)
			input = &sts.AssumeRoleInput{
				RoleArn:         &account.Role[i],
				RoleSessionName: &letmeContext.AwsSessionName,
				DurationSeconds: &letmeContext.AwsSessionDuration,
			}
			output, err = sesChainedSts.AssumeRole(context.TODO(), input)
			CheckAndReturnError(err)
		}
	}

	profileCredential := ProfileCredential{
		AccessKey:    *output.Credentials.AccessKeyId,
		SecretKey:    *output.Credentials.SecretAccessKey,
		SessionToken: *output.Credentials.SessionToken,
	}

	profileConfig := ProfileConfig{
		Output: "json",
		Region: account.Region[0],
	}

	switch {
	case localCredentialProcessFlagV1:
		fmt.Printf(CredentialsProcessOutput(profileCredential.AccessKey, profileCredential.SecretKey, profileCredential.SessionToken, *output.Credentials.Expiration))
		os.Exit(0)
	case renew || !CheckAccountAvailability(account.Name):
		DatabaseFile(account.Name, letmeContext.AwsSessionDuration, CredentialsProcessOutput(profileCredential.AccessKey, profileCredential.SecretKey, profileCredential.SessionToken, *output.Credentials.Expiration), authMethod) //only when we really authenticate against aws
	}
	return profileCredential, profileConfig
}

// Check if letme-config file exists and if its valid
func ConfigFileHealth() {
	if _, err := os.Stat(GetHomeDirectory() + "/.letme/letme-config"); err == nil {
	} else {
		fmt.Println("letme: no contexts found. Run 'letme config new-context ${contextName}' to create one.")
		os.Exit(1)
	}
	result := CheckConfigFile(GetHomeDirectory() + "/.letme/letme-config")
	if result {
	} else {
		fmt.Println("letme: run 'letme config view-template' to obtain a template for your config file.")
		os.Exit(1)
	}
}
func ActiveAccounts() (string, error) {
	databaseFileReader, err := os.ReadFile(GetHomeDirectory() + "/.letme/.letme-db")
	CheckAndReturnError(err)

	var accounts []Account
	err = json.Unmarshal(databaseFileReader, &accounts)
	CheckAndReturnError(err)

	currentTime := time.Now().Unix()

	activeAccounts := make(map[string]map[string]int64)

	for _, account := range accounts {
		if account.Account.LastRequest <= currentTime && currentTime <= account.Account.Expiry {
			activeAccounts[account.Account.Name] = map[string]int64{
				"expiry": account.Account.Expiry,
			}
		}
	}

	activeAccountsJSON, err := json.MarshalIndent(activeAccounts, "", "  ")
	if err != nil {
		return "", err
	}

	return string(activeAccountsJSON), nil
}