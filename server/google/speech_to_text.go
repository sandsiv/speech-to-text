package google

import (
	speech "cloud.google.com/go/speech/apiv1"
	"context"
	"errors"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"time"
	"unicode"
)

var languages = map[string]string{
	"en": "en-US",
}

func SpeechToText(pathToFile string, rate int32, language string) (error, string) {
	language, err := getLanguage(language)
	if err != nil {
		return err, ""
	}
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return err, ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Second)
	defer cancel()

	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: rate,
			LanguageCode:    language,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: pathToFile},
		},
	}

	op, err := client.LongRunningRecognize(ctx, req)
	if err != nil {
		return err, ""
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return err, ""
	}

	resultText := ""
	for _, result := range resp.Results {
		var confidence float32 = 0
		transcript := ""
		for _, alt := range result.Alternatives {
			if alt.Confidence > confidence {
				confidence = alt.Confidence
				transcript = alt.Transcript
			}
		}
		resultText += ucFirst(transcript) + "."
	}

	return nil, resultText
}

func getLanguage(language string) (string, error) {
	if val, ok := languages[language]; ok {
		return val, nil
	}
	return "", errors.New("language with code '" + language + "' is not supported")
}

func ucFirst(str string) string {
	hasSpace := false
	for _, v := range str {
		if str == " " {
			hasSpace = true
			continue
		}
		u := string(unicode.ToUpper(v))
		start := ""
		if hasSpace {
			start = " "
		}
		return start + u + str[len(u):]
	}
	return ""
}
