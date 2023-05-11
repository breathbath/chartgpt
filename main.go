package main

import (
	"breathbathChatGPT/pkg/cmd"
	"breathbathChatGPT/pkg/errs"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	err := godotenv.Overload(".env.default", ".env.secret", ".env.local")
	errs.Handle(err, true)

	err = cmd.Execute()
	errs.Handle(err, true)
}
