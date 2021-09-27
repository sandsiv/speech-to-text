package audio_server

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Alliera/speech-to-text/server/google"
	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"log"
	"net"
	"sync"
	"time"
)

type RecognitionResult struct {
	Time   time.Time
	Result string
}

const listenAddr = ":7071"
const MaxCallDuration = 2 * time.Minute

var RecognitionResults sync.Map

func Start() {
	ctx := context.Background()
	go removeOldRecognitionResults()
	if err := Listen(ctx); err != nil {
		log.Fatalln("listen failure:", err)
	}
	log.Println("exiting")
}

func removeOldRecognitionResults() {
	for {
		time.Sleep(time.Minute)
		RecognitionResults.Range(func(key, value interface{}) bool {
			recognitionResult := value.(RecognitionResult)
			diff := time.Now().Sub(recognitionResult.Time)
			if diff > time.Hour*3 {
				log.Fatalf("Text receiving timeout exeeded!")
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
		return errors.Wrapf(err, "failed to bind listener to socket %s", listenAddr)
	}
	fmt.Println("Audiosocket server started on " + listenAddr)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("failed to accept new connection:", err)
			continue
		}

		go Handle(ctx, conn)
	}
}

func getCallID(c net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(c)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, errors.Errorf("invalid message type %d getting CallID", m.Kind())
	}

	return uuid.FromBytes(m.Payload())
}

func Handle(pCtx context.Context, c net.Conn) {
	ctx, cancel := context.WithTimeout(pCtx, MaxCallDuration)

	defer func() {
		cancel()

		if _, err := c.Write(audiosocket.HangupMessage()); err != nil {
			log.Println("failed to send hangup message:", err)
		}
	}()

	id, err := getCallID(c)
	fmt.Println(id)
	idBytes := id.Bytes()
	enterpriseId := int(binary.LittleEndian.Uint16(idBytes[0:2]))
	languageCode := string(idBytes[2:4])

	if err != nil {
		log.Println("failed to get call ID:", err)
		return
	}
	log.Printf("processing call %s", id.String())

	resp, err := google.SpeechToTextFromStream(ctx, c, MaxCallDuration, enterpriseId, languageCode)
	if err != nil {
		log.Println("failed to process command:", err)
		return
	}
	fmt.Println(resp)
	RecognitionResults.Store(id.String(), RecognitionResult{Time: time.Now(), Result: resp})
}
