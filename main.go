package main

import (
	"encoding/json"
	"fmt"
	reader "github.com/Alliera/speech-to-text/server/audio"
	"github.com/Alliera/speech-to-text/server/dto"
	"github.com/Alliera/speech-to-text/server/google"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	log.Println("Starting Speech to text service...")
	if os.Getenv("BUCKET_NAME") == "" {
		panic("Env variable BUCKET_NAME is required")
	}
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		panic("Env variable GOOGLE_APPLICATION_CREDENTIALS is required")
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/getTexts", Logger(textsHandler))
	s := http.Server{
		Addr:           "0.0.0.0:7070",
		Handler:        handler,
		ReadTimeout:    1000 * time.Second,
		WriteTimeout:   1000 * time.Second,
		IdleTimeout:    1000 * time.Second,
		MaxHeaderBytes: 1 << 20, //1*2^20 - 128 kByte
	}
	log.Println(s.ListenAndServe())
}

func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		log.Printf("Server [http] method [%s] connnection from [%v]", r.Method, r.RemoteAddr)
		next.ServeHTTP(w, r)
	}
}

func handleError(error error) {
	if error != nil {
		log.Print(error)
	}
}

func textsHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	handleError(err)
	var texts []dto.Text
	err = json.Unmarshal(body, &texts)
	handleError(err)
	c := make(chan dto.Text)
	chunks := chunkBy(texts, 20)
	var uploadedTexts []dto.Text
	for _, chunk := range chunks {
		for _, text := range chunk {
			go uploadToCloud(text, c)
		}
		for i := 0; i < len(chunk); i++ {
			uploadedTexts = append(uploadedTexts, <-c)
		}
	}
	for _, text := range uploadedTexts {
		go recognize(text, c)
	}
	var results []dto.Text
	for i := 0; i < len(texts); i++ {
		results = append(results, <-c)
	}
	result, err := json.Marshal(results)
	handleError(err)
	_, err = w.Write(result)
	handleError(err)
	fmt.Println("Recognition Audio completed successfully")
}

func chunkBy(texts []dto.Text, chunkSize int) (chunks [][]dto.Text) {
	for chunkSize < len(texts) {
		texts, chunks = texts[chunkSize:], append(chunks, texts[0:chunkSize:chunkSize])
	}

	return append(chunks, texts)
}

func uploadToCloud(text dto.Text, c chan dto.Text) {
	text.Link, text.FilePath, text.Error = google.WriteToCloudStorage(text.FileUrl)

	c <- text
}

func recognize(text dto.Text, c chan dto.Text) {
	err := text.Error
	if err == nil {
		rate, duration := reader.GetRateAndLength(text.FilePath)
		text.Duration = roundSecs(duration)
		err, text.Text = google.SpeechToText(text.Link, rate, text.Language)
		handleError(err)
		err = google.DeleteFile(text.Link)
		handleError(err)
		err := os.Remove(text.FilePath)
		handleError(err)
	}
	handleError(err)

	c <- text
}

//Google use 15 sec blocks billing
func roundSecs(sec float64) int32 {
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
