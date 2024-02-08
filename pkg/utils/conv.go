package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Vernacular-ai/godub/converter"
	"github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func ConvertAudioFileFromOggToMp3(oggFile io.ReadCloser) (io.Reader, error) {
	defer oggFile.Close()

	buf := new(bytes.Buffer)

	err := converter.NewConverter(buf).
		WithDstFormat("mp3").
		Convert(oggFile)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

type RangeFloat struct {
	From float64
	To   float64
}

func NormalizeJSON(ctx context.Context, input string) string {
	log := logrus.WithContext(ctx).WithField("function call", "NormalizeJSON")

	minifiedString := strings.ReplaceAll(input, "\n", "")
	minifiedString = strings.ReplaceAll(minifiedString, "\r", "")

	myCmd := exec.Command("jsonrepair")
	stdin, err := myCmd.StdinPipe()
	if err != nil {
		log.Errorf("Error creating stdin pipe: %v", err)
		return input
	}

	if _, err := stdin.Write([]byte(minifiedString)); err != nil {
		log.Errorf("Error writing to stdin: %v", err)
		return input
	}

	stdin.Close()

	stdout, err := myCmd.StdoutPipe()
	if err != nil {
		log.Errorf("Error getting command's stdout: %v", err)
		return input
	}

	if err := myCmd.Start(); err != nil {
		log.Errorf("Error starting command: %v", err)
		return input
	}

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		log.Errorf("Error getting command's stdout: %v", err)
		return input
	}

	// Wait for the command to finish
	if err := myCmd.Wait(); err != nil {
		log.Errorf("Error waiting for command: %v", err)
	}

	stdErr, err := myCmd.StderrPipe()
	if err == nil {
		if stdErrText, err := io.ReadAll(stdErr); err == nil {
			log.Errorf("command's stderr: %s", string(stdErrText))
		}
		return input
	}

	return string(stdOut)
}

func ParseRangeFloat(rawRange []interface{}) (rangeRes *RangeFloat) {
	for i, rawVal := range rawRange {
		floatVal, ok := ParseFloat(rawVal)
		if !ok {
			continue
		}
		if rangeRes == nil {
			rangeRes = &RangeFloat{}
		}

		if len(rawRange) == 1 {
			rangeRes.From = floatVal
		} else {
			if i == 0 {
				rangeRes.From = floatVal
			} else {
				rangeRes.To = floatVal
			}
		}
	}

	return rangeRes
}

func ParseEnumStr(rawVal interface{}, enums []string) string {
	result := fmt.Sprint(rawVal)
	for i := range enums {
		if strings.ToLower(result) == strings.ToLower(enums[i]) {
			return result
		}
	}

	return ""
}

func ParseStrings(rawValues []interface{}) []string {
	result := make([]string, 0, len(rawValues))
	for _, val := range rawValues {
		valStr := fmt.Sprint(val)
		if valStr != "" {
			result = append(result, valStr)
		}
	}

	return result
}

func ParseFloat(rawFloat interface{}) (float64, bool) {
	fl64, ok := rawFloat.(float64)
	if ok {
		return fl64, true
	}

	fl32, ok := rawFloat.(float32)
	if ok {
		return float64(fl32), true
	}

	intVal32, ok := rawFloat.(int)
	if ok {
		return float64(intVal32), true
	}

	intVal64, ok := rawFloat.(int64)
	if ok {
		return float64(intVal64), true
	}

	fl64, err := strconv.ParseFloat(fmt.Sprint(rawFloat), 64)
	if err == nil {
		return fl64, true
	}

	return 0, false
}
