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
	for _, text := range texts {
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
	fmt.Println("Ваш финиш еще полчаса назад был!!")
}

func recognize(text dto.Text, c chan dto.Text) {
	link, filePath, err := google.WriteToCloudStorage(text.FileUrl)
	if err == nil {
		rate, duration := reader.GetRateAndLength(filePath)
		text.Duration = roundSecs(duration)
		err, text.Text = google.SpeechToText(link, rate, text.Language)
		handleError(err)
		err = google.DeleteFile(link)
		handleError(err)
		err := os.Remove(filePath)
		handleError(err)
	}

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
