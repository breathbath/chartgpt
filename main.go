package main

import (
	"breathbathChatGPT/pkg/cmd"
	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	cfgFiles := []string{".env.default", ".env.secret", ".env.local"}
	existingFiles := make([]string, 0, len(cfgFiles))
	for _, cfgF := range cfgFiles {
		if utils.FileExists(cfgF) {
			existingFiles = append(existingFiles, cfgF)
		}
	}
	err := godotenv.Overload(existingFiles...)
	errs.Handle(err, true)

	err = cmd.Execute()
	errs.Handle(err, true)
}
