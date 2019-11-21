package crawl

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		backupConfigFile:  configFile() + ".backup",
		invalidConfigFile: configFile() + ".invalid",
		nonExistFile:      configFile() + ".delete",
	}
	_ = os.Remove(conf.nonExistFile)
	return conf
}

func getEnv() []string {
	return os.Environ()
}

func (test *configTest) makeInvalidConfigFile(t *testing.T) bool {
	// Backup configuration
	if err := os.Rename(test.validConfigFile, test.backupConfigFile); err != nil {
		t.Errorf("Could not backup/rename %s to %s : %s", test.validConfigFile, test.backupConfigFile, err)
		return false
	}

	// Create phony file
	f, err := os.OpenFile(test.validConfigFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		t.Errorf("Could not create fake config file %s : %s", test.validConfigFile, err)
		return false
	}
	_, _ = f.WriteString("Invalid content\n  for a yaml file\n")
	return true
}

func restoreEnv(env []string) {
	for _, e := range env {
		kv := strings.Split(e, "=")
		_ = os.Setenv(kv[0], kv[1])
	}
}

func backupConfigFileAndEnv(t *testing.T, test *configTest) ([]string, bool) {
	if err := os.Rename(test.validConfigFile, test.backupConfigFile); err != nil {
		t.Errorf("Could not backup/rename %s to %s", test.validConfigFile, test.backupConfigFile)
		return nil, false
	}
	return getEnv(), true
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func restoreConfigFileAndEnv(t *testing.T, test *configTest, env []string) {
	restoreEnv(env)

	// If we have a backup, remove original and backup
	if !fileExists(test.backupConfigFile) {
		return
	}

	if fileExists(test.validConfigFile) {
		if err := os.Remove(test.validConfigFile); err != nil {
			t.Logf("Could not remove valid configfile (%s) before backup : %s", test.validConfigFile, err)
		}
	}

	if err := os.Rename(test.backupConfigFile, test.validConfigFile); err != nil {
		t.Errorf("Could not backup/rename %s to %s : %s", test.validConfigFile, test.backupConfigFile, err)
		return
	}

	// Remove backup file
	if fileExists(test.backupConfigFile) {
		if err := os.Remove(test.backupConfigFile); err != nil {
			t.Logf("Could not remove backup configfile %s : %s", test.backupConfigFile, err)
		}
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
	conf.Logging.Permissions = 0666

	file, err := ioutil.TempFile("", "*.log-crawler.log")
	if err != nil {
		log.Fatal(err)
	}
	conf.Logging.File = file.Name()

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

// Test case when hardcoded config file is not there, and env vars not set
func TestInitialiseCrawlConfigurationNoFileNoEnv(t *testing.T) {
	test := getConfigTest()
	defConf := getTestConfig()

	// Backup config file and env vars
	env, ok := backupConfigFileAndEnv(t, test)
	if !ok {
		return
	}
	os.Clearenv()

	conf, _ := initialiseCrawlConfiguration()

	assert.Equal(t, *defConf, *conf)
	/*
		if *conf != *defConf {
			t.Error("if env vars not set and no config file, should set emergency env.")
		}*/

	// Restore config file and env vars
	restoreConfigFileAndEnv(t, test, env)
}

func TestInitialiseCrawlConfigurationInvalidConfigFile(t *testing.T) {
	var err error
	test := getConfigTest()

	// Backup config file and env vars and clear them
	env := getEnv()
	os.Clearenv()

	// Place an invalid phony config file
	if !test.makeInvalidConfigFile(t) {
		goto restore
	}

	// Test case where there are missing environment vars, config file is present but could not be parsed
	_, err = initialiseCrawlConfiguration()
	if err == nil {
		t.Error("initialiseCrawlConfiguration() should fail if config file is not a valid yaml file.")
	}

	// Restore config file and env vars
restore:
	restoreConfigFileAndEnv(t, test, env)
}

func TestInitialiseCrawlConfigurationSuccess(t *testing.T) {
	var fileConf config
	isPresent, err := configLoadFile(&fileConf, configFile())
	if !isPresent || err != nil {
		t.Errorf("This test demands a valid configuration file is present and used : %s", err)
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
