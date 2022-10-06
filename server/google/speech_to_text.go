package google

import (
	speech "cloud.google.com/go/speech/apiv1"
	"context"
	"github.com/CyCoreSystems/audiosocket"
	"github.com/pkg/errors"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"io"
	"log"
	"sync"
	"time"
	"unicode"
)

var speechToTextClients sync.Map

var languages = map[string]string{
	"en": "en-US",
	"it": "it-IT",
	"nl": "nl-NL",
	"es": "es-ES",
	"ca": "ca-ES",
	"gl": "gl-ES",
	"pt": "pt-PT",
	"pl": "pl-PL",
	"ro": "ro-RO",
	"el": "el-GR",
	"da": "da-DK",
	"ru": "ru-RU",
	"sl": "sl-SI",
	"hr": "hr-HR",
	"de": "de-DE",
	"fr": "fr-FR",
	"bg": "bg-BG",
	"sr": "sr-SP",
	"mk": "mk-MK",
	"eu": "eu-ES",
	"fi": "fil-PH",
}

var phoneCallModelLanguage = map[string]string{
	"en": "en-US",
	"fr": "fr-FR",
}

func getSpeechToTextClient(ctx context.Context, enterpriseId int) (*speech.Client, error) {
	client, ok := speechToTextClients.Load(enterpriseId)
	if !ok {
		newClient, err := speech.NewClient(ctx, GetCredentials(enterpriseId))
		if err != nil {
			return nil, err
		}
		speechToTextClients.Store(enterpriseId, newClient)

		return getSpeechToTextClient(ctx, enterpriseId)
	}

	return client.(*speech.Client), nil
}

//Google use 15 sec blocks billing
func RoundSecs(sec float64) int32 {
	var secondsTarification float64 = 15
	blocks := sec / secondsTarification
	blocksInt := int32(blocks)
	remainder := blocks - float64(blocksInt)
	var overSecs float64 = 0
	if remainder != 0 {
		overSecs = secondsTarification
	}

	return blocksInt*15 + int32(overSecs)
}

func SpeechToTextFromStream(
	pCtx context.Context,
	r io.ReadWriter,
	timeout time.Duration,
	enterpriseId int,
	languageCode string) (duration int64, text string, err error) {
	ctx, cancel := context.WithTimeout(pCtx, timeout)
	defer cancel()
	client, err := getSpeechToTextClient(ctx, enterpriseId)
	if err != nil {
		return 0, "", err
	}
	svc, err := client.StreamingRecognize(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "failed to start streaming recognition")
	}
	language, err := getLanguage(languageCode)
	if err != nil {
		return 0, "", err
	}
	model := "default"
	if _, ok := phoneCallModelLanguage[languageCode]; ok {
		model = "phone_call"
	}

	if err := svc.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 8000,
					LanguageCode:    language,
					Model:           model,
					UseEnhanced:     true,
				},
			},
		},
	}); err != nil {
		return 0, "", errors.Wrap(err, "failed to send recognition config")
	}

	go pipeFromSocket(ctx, r, svc)

	for {
		resp, err := svc.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Cannot stream results: %v", err)
			break
		}
		if err := resp.Error; err != nil {
			log.Printf("Could not recognize: %v", err)
			break
		}
		for _, result := range resp.Results {
			duration = result.GetResultEndTime().Seconds
			for _, alt := range result.GetAlternatives() {
				if alt.Transcript != "" {
					text += alt.Transcript + "."
					break
				}
			}

		}
	}

	return duration, text, nil
}
func pipeFromSocket(ctx context.Context, in io.Reader, out speechpb.Speech_StreamingRecognizeClient) {
	var err error
	var m audiosocket.Message

	// Uncomment for test received voice
	//wavOut, writer := reader.CreateFile("Test.wav")
	//defer wavOut.Close()
	//defer writer.Close()
	//defer out.CloseSend()

	for ctx.Err() == nil {
		m, err = audiosocket.NextMessage(in)
		if errors.Cause(err) == io.EOF {
			// Wait last words recognition by google
			time.Sleep(1 * time.Second)
			err = out.CloseSend()
			if err != nil {
				log.Println(err)
			}
			log.Println("audiosocket closed")
			return
		}
		if m.Kind() == audiosocket.KindHangup {
			log.Println("audiosocket received hangup command")
			return
		}
		if m.Kind() == audiosocket.KindError {
			log.Println("error from audiosocket")
			continue
		}
		if m.Kind() != audiosocket.KindSlin {
			log.Println("ignoring non-slin message", m.Kind())
			continue
		}
		if m.ContentLength() < 1 {
			log.Println("no content")
			continue
		}
		// Uncomment for write Payload in file (for texting)
		//writer.Write(m.Payload())
		if err = out.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: m.Payload(),
			},
		}); err != nil {
			if err == io.EOF {
				log.Println("recognition client closed")
				return
			}
			log.Println("failed to send audio data for recognition:", err)
		}
	}
}

func SpeechToTextFromFile(pathToFile string, rate int32, language string, enterpriseId int) (error, string) {
	language, err := getLanguage(language)
	if err != nil {
		return err, ""
	}
	ctx := context.Background()
	client, err := speech.NewClient(ctx, GetCredentials(enterpriseId))
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
	_ = client.Close()

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
