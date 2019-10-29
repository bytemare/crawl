package crawl

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

var log = logrus.New()

func configFile() string {
	return "configs/config.yml"
}

// Config holds the crawler's secondary running parameters
// Environment variables overwrite configuration file values.
type Config struct {
	Requests struct {
		Timeout time.Duration `yaml:"timeout" envconfig:"CRAWLER_REQ_TIMEOUT"`
		Retries uint          `yaml:"retries" envconfig:"CRAWLER_REQ_RETRIES"`
	} `yaml:"requests"`
	Logging struct {
		Level       uint   `yaml:"level" envconfig:"CRAWLER_LOG_LEVEL"`
		Output      string `yaml:"output" envconfig:"CRAWLER_LOG_OUTPUT"`
		File        string `yaml:"file" envconfig:"CRAWLER_LOG_FILE"`
		Type        string `yaml:"type" envconfig:"CRAWLER_LOG_TYPE"`
		Permissions uint   `yaml:"perms" envconfig:"CRAWLER_LOG_FILE_PERMS"`
		Do          bool   `yaml:"do" envconfig:"CRAWLER_LOG"`
	} `yaml:"logging"`
}

// configGetEnvKeys returns the list of strings containing the environment variables' key names
func configGetEnvKeys() []string {
	envKeys := []string{
		"CRAWLER_REQ_TIMEOUT",
		"CRAWLER_REQ_RETRIES",
		"CRAWLER_LOG",
		"CRAWLER_LOG_LEVEL",
		"CRAWLER_LOG_OUTPUT",
		"CRAWLER_LOG_FILE",
		"CRAWLER_LOG_TYPE",
		"CRAWLER_LOG_FILE_PERMS",
	}

	return envKeys
}

// configGetEmergencyConf returns a minimal configuration only used when no config file or env vars are set
// Warning : logging is disabled.
func configGetEmergencyConf() *Config {
	return &Config{
		Requests: struct {
			Timeout time.Duration `yaml:"timeout" envconfig:"CRAWLER_REQ_TIMEOUT"`
			Retries uint          `yaml:"retries" envconfig:"CRAWLER_REQ_RETRIES"`
		}{0, 3},
		Logging: struct {
			Level       uint   `yaml:"level" envconfig:"CRAWLER_LOG_LEVEL"`
			Output      string `yaml:"output" envconfig:"CRAWLER_LOG_OUTPUT"`
			File        string `yaml:"file" envconfig:"CRAWLER_LOG_FILE"`
			Type        string `yaml:"type" envconfig:"CRAWLER_LOG_TYPE"`
			Permissions uint   `yaml:"perms" envconfig:"CRAWLER_LOG_FILE_PERMS"`
			Do          bool   `yaml:"do" envconfig:"CRAWLER_LOG"`
		}{2, "stdout", "", "text", 0, false},
	}
}

// initialiseCrawlConfiguration attempts to load the configuration from environment variables and a configuration file.
// If all environment variables are set, returns a Config containing their values.
// If some or all environment variables are missing, attempts to load a configuration from the default config file,
// and patches the missing environment variables.
// If no configuration file was found, an emergency configuration with minimal default values is spawned to patch the
// missing env vars.
//
// Returns errors when :
// 	- unable to load env vars
// 	- a config file is present but reading it failed
//	- an emergency configuration is used
//
// Note : When an emergency configuration is used, the env vars are populated and this function returns a valid Config
func initialiseCrawlConfiguration() (*Config, error) {
	// Load environment variables into Config
	envConf, err := configLoadEnv()
	if err != nil {
		return nil, err
	}

	// Check if environment variables are missing. If not, return them as Config
	missing := configCheckEnv()
	if len(missing) == 0 {
		return envConf, nil
	}

	// Here, env vars are not set or some are missing
	// Check is config file is present and if we can load it
	var fileConf Config
	isPresent, err := configLoadFile(&fileConf, configFile())
	if isPresent && err != nil {
		return nil, err
	}

	// If a configuration file is not present load the default emergency configuration (i.e. no logging, no timeout)
	var emergency error = nil
	if !isPresent {
		msg := fmt.Sprintf("Warning : Environmental values are missing and config file wasn't found." +
			"Using emergency configuration.")
		emergency = errors.New(msg)
		fileConf = *configGetEmergencyConf()
	}

	// Patch missing env vars
	err = configPatchEnv(missing, &fileConf)
	if err != nil {
		return nil, err
	}

	// Reload env vars to Config
	envConf, err = configLoadEnv()
	if err != nil {
		return nil, err
	}

	// Initialise logging
	configInitLogging(envConf)

	return envConf, emergency
}

// configCheckEnv checks mandatory environment variables, and returns missing keys, if any
func configCheckEnv() []string {
	envKeys := configGetEnvKeys()
	missing := make([]string, 0, len(envKeys))
	for _, key := range envKeys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

// configInitLogging sets logging behaviour
func configInitLogging(conf *Config) {
	// Enable or disable all logging
	if !conf.Logging.Do {
		log.SetOutput(ioutil.Discard)
	} else {
		// Set logging level
		log.SetLevel(logrus.Level(conf.Logging.Level))

		// Set logging output
		if conf.Logging.Output == "file" {
			log2File(conf.Logging.File, os.FileMode(conf.Logging.Permissions))
		}

		// Set logging format
		switch conf.Logging.Type {
		case "json":
			log.SetFormatter(&logrus.JSONFormatter{})
		case "text":
			log.SetFormatter(&logrus.TextFormatter{})
		default:
		}
	}
}

// log2File switches logging to be output to file only
func log2File(logFile string, perms os.FileMode) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, perms)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr.")
	}
}

// configPatchEnv populates missing environment variables with values found in the configuration file
//func configPatchEnv(envConf *Config, missing []string, fileConf *Config) error {
func configPatchEnv(missing []string, fileConf *Config) error {
	fileVal := reflect.ValueOf(*fileConf)

	for _, envName := range missing {
		/*if err := configSetMissingEnvVar(envConf, envName, fileVal); err != nil {
			return err
		}*/

		// Get value of missing var from config file
		val, err := configGetValFromTag(envName, fileVal)
		if err != nil {
			return err
		}

		// Update the environment variable with value
		if err := os.Setenv(envName, fmt.Sprintf("%v", val)); err != nil {
			fmt.Printf("Error in setting env var\n")
			return err
		}

		// Update the environment Config with value
		/*if err := configUpdateEnvConfig(envConf, envName, val); err != nil {
			return err
		}*/
	}

	return nil
}

// configGetValFromTag returns the value associated to key in the struct reflected by value parameter
func configGetValFromTag(key string, value reflect.Value) (interface{}, error) {
	// Iterate over all available fields
	for i := 0; i < value.NumField(); i++ {
		// Get the field
		field := value.Field(i)

		// iterate over all subfields, and look for the corresponding variable for the tag
		for j := 0; j < field.NumField(); j++ {
			// Get the tag
			entryTag := value.Type().Field(i).Type.Field(j).Tag.Get("envconfig")

			// If the unset environment variable is found in file config, return its value
			if entryTag == key {
				return field.Field(j).Interface(), nil
			}
		}
	}

	return nil, fmt.Errorf("could not find key '%s' in struct '%s'", key, value.Type())
}

// configUpdateEnvConfig populates key:value in conf
/*func configUpdateEnvConfig(conf *Config, key string, value interface{}) error {
	yamlVal := key + ": " + fmt.Sprintf("%v", value)
	decoder := yaml.NewDecoder(strings.NewReader(yamlVal))
	if err := decoder.Decode(conf); err != nil {
		msg := fmt.Sprintf("Unable to update env Config : %s", err)
		return errors.New(msg)
	}
	return nil
}*/

// configLoadFile loads the crawler's configuration from filePath
func configLoadFile(config *Config, filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		msg := fmt.Sprintf("Unable to open config file : %s", err)
		return false, errors.New(msg)
	}

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		msg := fmt.Sprintf("Unable to parse config file : %s", err)
		return true, errors.New(msg)
	}

	return true, nil
}

// configLoadEnv loads the crawler's configuration from environment variables
func configLoadEnv() (*Config, error) {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		msg := fmt.Sprintf("Unable to gather Env vars for configuration : %s", err)
		return nil, errors.New(msg)
	}
	return &config, nil
}
