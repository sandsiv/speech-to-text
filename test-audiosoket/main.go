package main

import (
	"encoding/binary"
	"fmt"
	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const serverAddr = "localhost:7071"
const audioFilePath = "/Users/username/Documents/IVR/Open.wav"

func main() {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	callID := GenerateId()
	if err := sendCallID(conn, callID); err != nil {
		log.Fatalf("failed to send call ID: %v", err)
	}

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		log.Fatalf("failed to open audio file: %v", err)
	}
	defer audioFile.Close()

	if err := sendAudioFile(conn, audioFile); err != nil {
		log.Fatalf("failed to send audio file: %v", err)
	}

	if err := sendHangupMessage(conn); err != nil {
		log.Fatalf("failed to send hangup message: %v", err)
	}

	fmt.Println("Audio file sent and connection closed successfully.")
}

func sendCallID(conn net.Conn, callID uuid.UUID) error {
	msg := audiosocket.IDMessage(callID)
	_, err := conn.Write(msg)
	return err
}

func sendAudioFile(conn net.Conn, file *os.File) error {
	bufSize := 160
	buf := make([]byte, bufSize)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read audio file: %w", err)
		}
		if n == 0 {
			break
		}

		audioMsg := audiosocket.SlinMessage(buf[:n])
		_, err = conn.Write(audioMsg)
		if err != nil {
			return fmt.Errorf("failed to send audio data: %w", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

func sendHangupMessage(conn net.Conn) error {
	hangupMsg := audiosocket.HangupMessage()
	_, err := conn.Write(hangupMsg)
	return err
}

func GenerateId() uuid.UUID {
	var entID uint16 = 1
	lang := "en"
	id := uuid.Must(uuid.NewV4())
	idBytes := id.Bytes()
	enterpriseIdBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(enterpriseIdBytes, entID)
	idBytes[0] = enterpriseIdBytes[0]
	idBytes[1] = enterpriseIdBytes[1]
	langCode := []byte(lang)
	idBytes[2] = langCode[0]
	idBytes[3] = langCode[1]
	newId, _ := uuid.FromBytes(idBytes)

	return newId
}
