package utils

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

// struct which represents the config-file toml keys
type GeneralParams struct {
	Aws_source_profile        string
	Aws_source_profile_region string `toml:"aws_source_profile_region,omitempty"`
	Dynamodb_table            string
	Mfa_arn                   string `toml:"mfa_arn,omitempty"`
	Session_name              string
}

// struct which represents the cache file toml keys
type CacheFields struct {
	Id     int      `toml:"id"`
	Name   string   `toml:"name"`
	Role   []string `toml:"role"`
	Region []string `toml:"region"`
}

// verify config-file integrity
func CheckConfigFile(path string) bool {
	type config struct {
		General struct {
			Aws_source_profile        string
			Aws_source_profile_region string `toml:"aws_source_profile_region,omitempty"`
			Dynamodb_table            string
			Mfa_arn                   string `toml:"mfa_arn,omitempty"`
			Session_name              string
		}
	}
	var conf config
	md, err := toml.DecodeFile(path, &conf)
	CheckAndReturnError(err)
	if len(md.Undecoded()) > 0 {
		fmt.Printf("letme: config file is corrupted. Following values might be misspelled:\n")
		fmt.Printf("* %v \n", md.Undecoded())
		return false
	} else {
		return true
	}
}

// check if a command exists on the host machine
func CommandExists(command string) {
	_, err := exec.LookPath(command)
	CheckAndReturnError(err)
}

// checks the error, if the error contains a message, stop the execution and show the error to the user
func CheckAndReturnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// marshalls data into a toml file (config-file)
func TemplateConfigFile() string {
	var (
		buf = new(bytes.Buffer)
	)
	err := toml.NewEncoder(buf).Encode(map[string]interface{}{
		"general": map[string]string{
			"aws_source_profile":        "",
			"aws_source_profile_region": "",
			"dynamodb_table":            "",
			"mfa_arn":                   "",
			"session_name":              "",
		},
	})
	CheckAndReturnError(err)
	return buf.String()
}

// marshalls data into a toml file (.letme-cache)
func TemplateCacheFile(accountName string, accountID int, accountRole []string, accountRegion []string) string {
	var (
		buf = new(bytes.Buffer)
	)
	err := toml.NewEncoder(buf).Encode(map[string]interface{}{
		accountName: map[string]interface{}{
			"id":     accountID,
			"name":   accountName,
			"role":   accountRole,
			"region": accountRegion,
		},
	})
	CheckAndReturnError(err)
	return buf.String()
}

// gets user's $HOME directory
func GetHomeDirectory() string {
	homeDir, err := os.UserHomeDir()
	CheckAndReturnError(err)
	return homeDir
}

// parses letme-config file and returns one field at a time
func ConfigFileResultString(field string) string {
	type structUnmarshal = GeneralParams
	type general map[string]structUnmarshal
	var generalConfig general
	_, err := toml.DecodeFile(GetHomeDirectory()+"/.letme/letme-config", &generalConfig)
	CheckAndReturnError(err)
	var exportedField string
	for _, name := range []string{"general"} {
		a := generalConfig[name]
		r := reflect.ValueOf(a)
		f := reflect.Indirect(r).FieldByName(field)
		exportedField = string(f.String())

	}
	return exportedField
}

// checks if a cache file exists
func CacheFileExists() bool {
	if _, err := os.Stat(GetHomeDirectory() + "/.letme/.letme-cache"); err == nil {
		return true
	} else {
		return false
	}
}

// reads the cache file
func CacheFileRead() string {
	readCacheFile, err := ioutil.ReadFile(GetHomeDirectory() + "/.letme/.letme-cache")
	CheckAndReturnError(err)
	s := string(readCacheFile)
	return s
}

// reads the aws credentials file
func AwsCredsFileRead() string {
	readCacheFile, err := ioutil.ReadFile(GetHomeDirectory() + "/.aws/credentials")
	CheckAndReturnError(err)
	s := string(readCacheFile)
	return s
}

// reads the aws config file
func AwsConfigFileRead() string {
	readCacheFile, err := ioutil.ReadFile(GetHomeDirectory() + "/.aws/config")
	CheckAndReturnError(err)
	s := string(readCacheFile)
	return s
}

// maps data from the cache file into a struct
func ParseCacheFile(account string) CacheFields {
	type o = CacheFields
	type general map[string]o
	var generalConfig general
	homeDir := GetHomeDirectory()
	configFilePath := homeDir + "/.letme/.letme-cache"
	_, err := toml.DecodeFile(configFilePath, &generalConfig)
	CheckAndReturnError(err)
	s := generalConfig[account]
	return s
}

// marshalls data into a string
func AwsCredentialsFile(accountName string, accessKeyID string, secretAccessKey string, sessionToken string) string {
	return fmt.Sprintf(
		`#s-%v
#managed by letme
[%v]
aws_access_key_id = %v
aws_secret_access_key = %v
aws_session_token = %v
#e-%v
`, accountName, accountName, accessKeyID, secretAccessKey, sessionToken, accountName)
}

// marshalls data into a string
func AwsConfigFile(accountName string, region string) string {
	return fmt.Sprintf(
		`#s-%v
#managed by letme
[profile %v]
region = %v
output = json
#e-%v
`, accountName, accountName, region, accountName)
}

// removes from a file all text in between two strings
func AwsReplaceBlock(file string, accountName string) string {
	str := "#s-" + accountName
	etr := "#e-" + accountName
	empty := ""
	if strings.Contains(file, str) && strings.Contains(file, etr) {
		startIndex := strings.Index(file, str)
		stopIndex := strings.Index(file, etr) + len(etr)
		res := file[:startIndex] + file[stopIndex:]
		res = strings.ReplaceAll(res, "\n\n", "\n")
		return res
	}
	return empty
}
