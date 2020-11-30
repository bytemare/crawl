package crawl

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const defConfigFile = "configs/config.yml"

func configFile() string {
	return defConfigFile
}

// config holds the crawler's secondary running parameters
// Environment variables overwrite configuration file values.
type config struct {
	Requests struct {
		Timeout time.Duration `yaml:"timeout" envconfig:"CRAWLER_REQ_TIMEOUT"`
		Retries uint          `yaml:"retries" envconfig:"CRAWLER_REQ_RETRIES"`
	} `yaml:"requests"`
	Logger Logger `yaml:"log"`
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
func configGetEmergencyConf() *config {
	return &config{
		Requests: struct {
			Timeout time.Duration `yaml:"timeout" envconfig:"CRAWLER_REQ_TIMEOUT"`
			Retries uint          `yaml:"retries" envconfig:"CRAWLER_REQ_RETRIES"`
		}{0, 3},
		Logger: Logger{logrus.New(), 2, "stdout", "", "text", 600, false, nil},
	}
}

// initialiseCrawlerConfiguration attempts to load the configuration from environment variables and a configuration file
// If all environment variables are set, returns a config containing their values.
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
// NB : When an emergency configuration is used, the env vars are populated and this function returns a valid config
func initialiseCrawlerConfiguration() (*config, error) {
	// Check if environment variables are missing. If not, return them as config
	missing := configCheckEnv()
	if len(missing) == 0 {
		// Load and return environment variables into config
		return configLoadEnv()
	}

	// Here, env vars are not set or some are missing
	// Check if config file is present and if we can load it
	var fileConf config
	isPresent, err := configLoadFile(&fileConf, configFile())
	if isPresent && err != nil {
		return nil, errors.Wrap(err, exitErrorConf)
	}
	fileConf.Logger.log = logrus.New()

	// If a configuration file is not present load the default emergency configuration (i.e. no logging, no timeout)
	var emergency error = nil
	if !isPresent {
		msg := fmt.Sprint("Warning : Environmental values are missing and config file wasn't found." +
			"Using emergency configuration.")
		emergency = errors.New(msg)
		fileConf = *configGetEmergencyConf()
	}

	// Patch missing env vars
	err = configPatchEnv(missing, &fileConf)
	if err != nil {
		return nil, errors.Wrap(err, exitErrorConf)
	}

	// Reload env vars to config
	envConf, err := configLoadEnv()
	if err != nil {
		return nil, errors.Wrap(err, exitErrorConf)
	}

	// Initialise logging
	if err := envConf.Logger.init(); err != nil {
		return envConf, errors.Wrapf(err, "%s", emergency)
	}

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

// configPatchEnv populates missing environment variables with values found in the configuration file
func configPatchEnv(missing []string, fileConf *config) error {
	fileVal := reflect.ValueOf(*fileConf)

	for _, envName := range missing {
		// Get value of missing var from config file
		val, err := configGetValFromTag(envName, fileVal)
		if err != nil {
			return err
		}

		// Update the environment variable with value
		value := fmt.Sprintf("%v", val)
		if err := os.Setenv(envName, value); err != nil {
			return errors.Wrapf(err, "Error in setting env var '%s:%s'", envName, value)
		}
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

// configLoadFile loads the crawler's configuration from filePath
func configLoadFile(config *config, filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		msg := fmt.Sprintf("Unable to open config file : %s", err)
		return false, errors.New(msg)
	}
	defer func() {
		_ = file.Close() //nolint:errcheck // we don't really care about that
	}()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return true, errors.Wrapf(err, "Unable to parse config file '%s'", filePath)
	}

	return true, nil
}

// configLoadEnv loads the crawler's configuration from environment variables
func configLoadEnv() (*config, error) {
	var conf config
	err := envconfig.Process("", &conf)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to gather Env vars for configuration")
	}
	conf.Logger.log = logrus.New()
	return &conf, nil
}

// configUpdateEnvConfig populates key:value in conf
/*func configUpdateEnvConfig(conf *config, key string, value interface{}) error {
	yamlVal := key + ": " + fmt.Sprintf("%v", value)
	decoder := yaml.NewDecoder(strings.NewReader(yamlVal))
	if err := decoder.Decode(conf); err != nil {
		msg := fmt.Sprintf("Unable to update env config : %s", err)
		return errors.New(msg)
	}
	return nil
}*/
