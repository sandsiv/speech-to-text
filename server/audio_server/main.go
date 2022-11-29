package audio_server

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Alliera/logging"
	"github.com/Alliera/speech-to-text/server/google"
	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"net"
	"sync"
	"time"
)

var logger = logging.NewDefault("audio server")

type RecognitionResult struct {
	Time     time.Time `json:"time"`
	Text     string    `json:"text"`
	Duration int64     `json:"duration"`
}

const listenAddr = ":7071"
const MaxCallDuration = 2 * time.Minute

var RecognitionResults sync.Map

func Start() {
	ctx := context.Background()
	go removeOldRecognitionResults()
	if err := Listen(ctx); err != nil {
		logger.LogFatal(err, "listen failure")
	}
	logger.Debug("exiting")
}

func removeOldRecognitionResults() {
	for {
		time.Sleep(time.Minute)
		RecognitionResults.Range(func(key, value interface{}) bool {
			recognitionResult := value.(RecognitionResult)
			diff := time.Now().Sub(recognitionResult.Time)
			if diff > time.Hour*3 {
				logger.Info("Text receiving timeout exeeded!")
				RecognitionResults.Delete(key)
			}
			return true
		})
	}
}

// Listen listens for and responds to Audiosocket connections
func Listen(ctx context.Context) error {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return logging.Trace(fmt.Errorf("failed to bind listener to socket %s: %s", listenAddr, err))
	}
	fmt.Println("Audiosocket server started on " + listenAddr)
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to accept new connection: %s", err))
			continue
		}

		go Handle(ctx, conn)
	}
}

func getCallID(c net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(c)
	if err != nil {
		return uuid.Nil, logging.Trace(err)
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, logging.Trace(fmt.Errorf("invalid message type %d getting CallID", m.Kind()))
	}
	uuidValue, err := uuid.FromBytes(m.Payload())
	err = logging.Trace(err)
	return uuidValue, err
}

func Handle(pCtx context.Context, c net.Conn) {
	ctx, cancel := context.WithTimeout(pCtx, MaxCallDuration)

	defer func() {
		cancel()

		if _, err := c.Write(audiosocket.HangupMessage()); err != nil {
			logger.Error(fmt.Sprintf("failed to send hangup message: %s", err))
		}
	}()

	id, err := getCallID(c)
	logger.Debug(id.String())
	idBytes := id.Bytes()
	enterpriseId := int(binary.LittleEndian.Uint16(idBytes[0:2]))
	languageCode := string(idBytes[2:4])

	if err != nil {
		logger.LogError(err, "failed to get call ID")
		return
	}
	logger.Info(fmt.Sprintf("processing call %s", id.String()))

	duration, text, err := google.SpeechToTextFromStream(ctx, c, MaxCallDuration, enterpriseId, languageCode)
	duration = int64(google.RoundSecs(float64(duration)))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to process command: %s", err))
		return
	}

	RecognitionResults.Store(id.String(), RecognitionResult{Time: time.Now(), Text: text, Duration: duration})
}
