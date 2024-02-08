package utils

import (
	"bytes"
	"fmt"
	"github.com/Vernacular-ai/godub/converter"
	"io"
	"regexp"
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

func NormalizeStringArrays(input string) string {
	re := regexp.MustCompile(`\[([^][]+)]`)
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		nonQuoted := match[1]
		if nonQuoted != "" {
			parts := strings.Split(nonQuoted, ",")
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if !isQuoted(part) {
					parts[i] = strconv.Quote(part)
				}
			}
			quoted := strings.Join(parts, ",")
			input = strings.Replace(input, "["+nonQuoted+"]", "["+quoted+"]", -1)
		}
	}

	return input
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
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
