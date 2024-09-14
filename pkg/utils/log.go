package utils

import (
	"bufio"
	"io"

	"github.com/sirupsen/logrus"
)

// logStream logs the output from the given reader
func LogStream(reader io.ReadCloser, logLevel string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		switch logLevel {
		case "INFO":
			logrus.Infof(scanner.Text())
		case "ERROR":
			logrus.Errorf(scanner.Text())
		default:
			logrus.Infof(scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		logrus.Errorf("Error reading log stream: %v", err)
	}
}
