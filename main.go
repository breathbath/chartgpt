package main

import (
	"breathbathChartGPT/pkg/cmd"
	"breathbathChartGPT/pkg/errs"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env.default", ".env")
	errs.Handle(err, true)

	err = cmd.Execute()
	errs.Handle(err, true)
}
