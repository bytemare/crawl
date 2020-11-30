package crawl

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrorList implements the Error interface
type ErrorList []error

func (e ErrorList) Error() string {
	s := "multiple errors: "
	for _, err := range e[0 : len(e)-1] {
		s += err.Error() + "; "
	}
	return s + e[len(e)-1].Error()
}

// todo
func (e ErrorList) ErrorOrNil() error {
	if e == nil {
		return nil
	}
	if len(e) == 0 {
		return nil
	}

	return e
}

// todo
type Logger struct {
	log         *logrus.Logger
	Level       uint   `yaml:"level" envconfig:"CRAWLER_LOG_LEVEL"`
	Output      string `yaml:"output" envconfig:"CRAWLER_LOG_OUTPUT"`
	File        string `yaml:"file" envconfig:"CRAWLER_LOG_FILE"`
	Type        string `yaml:"type" envconfig:"CRAWLER_LOG_TYPE"`
	Permissions uint   `yaml:"perms" envconfig:"CRAWLER_LOG_FILE_PERMS"`
	Do          bool   `yaml:"do" envconfig:"CRAWLER_LOG"`
	fileDes     *os.File
}

func (l *Logger) init() (err error) {
	// Enable or disable all logging
	if !l.Do {
		l.log.SetOutput(ioutil.Discard)
	} else {
		// Set logging level
		l.log.SetLevel(logrus.Level(l.Level))

		// Set logging output
		if l.Output == "file" {
			err = l.SetOutputFile(l.File, l.Permissions)
		}

		// Set logging format
		if _err := l.SetType(); _err != nil {
			err = errors.Wrapf(err, "%s", _err)
		}
	}
	return err
}

// todo
func (l *Logger) SetLevel(lvl uint) error {
	//todo
	l.Level = lvl
	return nil
}

// todo
func (l *Logger) SetOutput(o io.Writer) {
	l.log.SetOutput(o)
}

func getTempfile() (*os.File, error) {
	file, err := ioutil.TempFile("", "crawler-*.log")
	if err != nil {
		return nil, errors.Wrap(err, "Could not set temporary log output file")
	}
	return file, nil
}

// todo
func (l *Logger) SetOutputFile(filepath string, perms uint) error {
	var file *os.File
	if perms == 0 {
		perms = 0600
	}

	if filepath != "" {
		logfile, err := l.log2File(filepath, os.FileMode(perms))
		if err == nil {
			l.File = logfile.Name()
			l.fileDes = logfile
			return nil
		}

		// todo err := errors.Wrapf(err, "Could not set logging output file to ", filepath)
	}

	// If no log file was set, set a temporary default one with a random id
	file, err := getTempfile()
	if err != nil {
		return err
	}
	l.File = file.Name()
	l.fileDes = file

	// redirect logging to file
	l.log.SetOutput(file)

	return nil
}

// todo
func (l *Logger) SetType() error {
	switch l.Type {
	case "json":
		l.log.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		l.log.SetFormatter(&logrus.TextFormatter{})
	default:
		return fmt.Errorf("unknown logging type '%s'", l.Type)
	}

	return nil
}

// log2File switches logging to be output to file only
func (l *Logger) log2File(logFile string, perms os.FileMode) (file *os.File, err error) {
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to set logging to file '%s'", logFile)
	}
	l.log.SetOutput(file)
	return file, nil
}
