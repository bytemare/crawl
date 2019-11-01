package crawl

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

type configTest struct {
	validConfigFile   string
	backupConfigFile  string
	invalidConfigFile string
	nonExistFile      string
}

func getConfigTest() *configTest {
	conf := &configTest{
		validConfigFile:   configFile(),
		backupConfigFile:  configFile() + "backup",
		invalidConfigFile: "config_test.go",
		nonExistFile:      configFile() + ".delete",
	}
	_ = os.Remove(conf.nonExistFile)
	return conf
}

func getEnv() []string {
	return os.Environ()
}

func restoreEnv(env []string) {
	for _, e := range env {
		kv := strings.Split(e, "=")
		_ = os.Setenv(kv[0], kv[1])
	}
}

func TestConfigLoadFileFail(t *testing.T) {
	conf := getTestConfig()
	test := getConfigTest()

	// Test loading a non-existent configuration file
	if isPresent, _ := configLoadFile(conf, test.nonExistFile); isPresent {
		t.Errorf("configLoadFile() shouldn't indicate that file is present for invalid filename '%s'\n.", "non-exist")
	}

	// Test parsing an invalid yaml configuration file
	if _, err := configLoadFile(conf, test.invalidConfigFile); err == nil {
		t.Error("configLoadFile() shouldn't be able to parse a non-yaml file.")
	}
}

func TestConfigInitLoggingSuccess(t *testing.T) {
	conf := getTestConfig()
	conf.Logging.Do = true
	conf.Logging.Level = 1
	conf.Logging.Output = "file"
	conf.Logging.File = "./log-crawler.log.test"
	conf.Logging.Permissions = 0666

	// Tests on
	conf.Logging.Type = "json"
	if err := configInitLogging(conf); err != nil {
		t.Errorf("init logging failed : '%s'.\n", err)
	}
	conf.Logging.Type = "text"
	if err := configInitLogging(conf); err != nil {
		t.Errorf("init logging failed : '%s'.\n", err)
	}

	if err := os.Remove(conf.Logging.File); err != nil {
		fmt.Printf("could not remove test log file '%s' : %s\n", conf.Logging.File, err)
	}
}

func TestInitialiseCrawlConfigurationNoFileNoEnv(t *testing.T) {
	test := getConfigTest()
	defConf := getTestConfig()

	// Backup config file and env vars
	_ = os.Rename(test.validConfigFile, test.backupConfigFile)
	env := getEnv()
	os.Clearenv()

	// Test case when hardcoded config file is not there, and env vars not set
	_ = os.Rename(test.validConfigFile, test.backupConfigFile)

	conf, _ := initialiseCrawlConfiguration()
	if *conf != *defConf {
		t.Error("if env vars not set and no config file, should set emergency env.")
	}

	// Restore config file and env vars
	restoreEnv(env)
	_ = os.Rename(test.backupConfigFile, test.validConfigFile)
}

func TestInitialiseCrawlConfigurationInvalidConfigFile(t *testing.T) {
	test := getConfigTest()

	// Backup config file and env vars and clear them
	_ = os.Rename(test.validConfigFile, test.backupConfigFile)
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	_ = os.Link(test.invalidConfigFile, test.validConfigFile)

	// Test case where there are missing environment vars, config file is present but could not be parsed
	_, err := initialiseCrawlConfiguration()
	if err == nil {
		t.Error("initialiseCrawlConfiguration() should fail if config file is not a valid yaml file.")
	}

	// Restore config file and env vars
	restoreEnv(env)
	_ = os.Rename(test.backupConfigFile, test.validConfigFile)
}

func TestInitialiseCrawlConfigurationSuccess(t *testing.T) {
	var fileConf config
	isPresent, err := configLoadFile(&fileConf, configFile())
	if !isPresent || err != nil {
		t.Error("This test demands a valid configuration file is present and used.")
		return
	}

	// Backup and clear env vars
	env := getEnv()
	os.Clearenv()

	// Initialise configuration
	initConf, err := initialiseCrawlConfiguration()
	if err != nil {
		t.Errorf("initialiseCrawlConfiguration() failed : %s\n", err)
		goto restore
	}

	// Check if configuration and environment were set up properly
	if *initConf != fileConf {
		t.Errorf("The initialised configuration is different from the one loaded from file."+
			"\n\t\tfile : %v\n\t\tinit : %v\n", fileConf, initConf)
	}

	// Clean and restore environment
restore:
	os.Clearenv()
	restoreEnv(env)
}
