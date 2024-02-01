package utils

import (
	"bytes"
	"github.com/Vernacular-ai/godub/converter"
	"io"
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
