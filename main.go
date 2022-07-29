package main

import (
	"encoding/json"
	"fmt"
	"github.com/Alliera/logging"
	reader "github.com/Alliera/speech-to-text/server/audio"
	"github.com/Alliera/speech-to-text/server/audio_server"
	"github.com/Alliera/speech-to-text/server/dto"
	"github.com/Alliera/speech-to-text/server/google"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var logger = logging.NewDefault("main")

func main() {
	logger.Info("Starting Speech to text service...")
	if os.Getenv("BUCKET_NAME") == "" {
		logger.Fatal("Env variable BUCKET_NAME is required")
	}
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		logger.Fatal("Env variable GOOGLE_APPLICATION_CREDENTIALS is required")
	}

	go startRestApiServer()
	go audio_server.Start()

	forever := make(chan bool)

	logger.Info(" [*] Server started. To exit press CTRL+C")
	<-forever
}

func startRestApiServer() {
	handler := http.NewServeMux()
	handler.HandleFunc("/getTexts", Logger(textsHandler))
	handler.HandleFunc("/getTextById", Logger(getTextById))
	handler.HandleFunc("/addCredentials", Logger(addCredentials))
	handler.HandleFunc("/healthz", healthz)
	port := ":7070"
	s := http.Server{
		Addr:           "0.0.0.0" + port,
		Handler:        handler,
		ReadTimeout:    1000 * time.Second,
		WriteTimeout:   1000 * time.Second,
		IdleTimeout:    1000 * time.Second,
		MaxHeaderBytes: 1 << 20, //1*2^20 - 128 kByte
	}
	fmt.Println("REST server started on " + port)
	err := s.ListenAndServe()
	if err != nil {
		logger.Error(err.Error())
	}
}

func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		logger.Info(fmt.Sprintf("Server [http] method [%s] connnection from [%v]", r.Method, r.RemoteAddr))
		next.ServeHTTP(w, r)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleError(err error) {
	if err != nil {
		logger.LogError(err)
	}
}

func responseError(w http.ResponseWriter, code int, err error) bool {
	if err == nil {
		return false
	}
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, "{\"err\":\""+err.Error()+"\"}")
	return true
}

func addCredentials(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	err = logging.Trace(err)
	if responseError(w, http.StatusBadRequest, err) {
		return
	}
	var credentials dto.Credentials
	err = logging.Trace(json.Unmarshal(body, &credentials))
	if responseError(w, http.StatusBadRequest, err) {
		return
	}
	if credentials.BucketName == "" || credentials.Credentials == nil || credentials.EnterpriseId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "{\"error\":\"credentials, bucketName and enterpriseId are required fields\"}")
		return
	}
	err = logging.Trace(google.CheckCredentials(credentials))
	if responseError(w, http.StatusConflict, err) {
		return
	}
	err = logging.Trace(google.AddBucketName(credentials.EnterpriseId, credentials.BucketName))
	if responseError(w, http.StatusConflict, err) {
		return
	}
	err = logging.Trace(google.AddCredentials(credentials.EnterpriseId, credentials.Credentials))
	if responseError(w, http.StatusConflict, err) {
		return
	}
}

func getTextById(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	id := v.Get("id")
	ok := false
	var res interface{}
	retryCount := 10
	for !ok && retryCount != 0 {
		res, ok = audio_server.RecognitionResults.LoadAndDelete(id)
		if ok {
			break
		}
		time.Sleep(time.Millisecond * 500)
		retryCount = retryCount - 1
	}

	var recognitionResult []byte

	if ok {
		recognitionResult, _ = json.Marshal(res.(audio_server.RecognitionResult))
		_, _ = w.Write(recognitionResult)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(recognitionResult)
	return
}

func textsHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	handleError(logging.Trace(err))
	var texts []dto.Text
	err = json.Unmarshal(body, &texts)
	handleError(logging.Trace(err))
	totalNum := strconv.Itoa(len(texts))
	fmt.Println(totalNum + " texts received")
	c := make(chan dto.Text)
	c1 := make(chan dto.Text)
	chunks := chunkBy(texts, 30)
	for _, chunk := range chunks {
		for _, text := range chunk {
			go uploadToCloud(text, c)
		}
		for i := 0; i < len(chunk); i++ {
			go recognize(<-c, c1)
		}
	}

	var results []dto.Text
	errorsNum := 0
	for i := 0; i < len(texts); i++ {
		text := <-c1
		if text.RecognitionError != nil {
			errorsNum++
		}
		results = append(results, text)
	}
	result, err := json.Marshal(results)
	handleError(logging.Trace(err))
	_, err = w.Write(result)
	handleError(logging.Trace(err))
	fmt.Println("Audio Recognition completed successfully. Errors: " + strconv.Itoa(errorsNum))
}

func chunkBy(texts []dto.Text, chunkSize int) (chunks [][]dto.Text) {
	for chunkSize < len(texts) {
		texts, chunks = texts[chunkSize:], append(chunks, texts[0:chunkSize:chunkSize])
	}

	return append(chunks, texts)
}

func uploadToCloud(text dto.Text, c chan dto.Text) {
	text.Link, text.FilePath, text.Error = google.WriteToCloudStorage(text.FileUrl, text.EnterpriseId)

	c <- text
}

func recognize(text dto.Text, c chan dto.Text) {
	err := text.Error
	if err == nil {
		rate, duration := reader.GetRateAndLength(text.FilePath)
		text.Duration = google.RoundSecs(duration)
		retry := 0
		for {
			text.RecognitionError, text.Text = google.SpeechToTextFromFile(
				text.Link,
				rate,
				text.Language,
				text.EnterpriseId)
			if text.RecognitionError == nil {
				break
			} else {
				errorText := text.RecognitionError.Error()
				fmt.Println(errorText)
				if strings.Contains(errorText, "Invalid audio file") || strings.Contains(errorText, "language with code") {
					break
				}

				time.Sleep(20 * time.Second)
				retry++
				fmt.Println("Retrying recognition request #" + strconv.Itoa(retry) + " after error:" + text.RecognitionError.Error())
			}
		}
	}
	if text.Link != "" {
		err = google.DeleteFile(text.Link, text.EnterpriseId)
		handleError(logging.Trace(err))
	}
	if text.FilePath != "" {
		err = os.Remove(text.FilePath)
		handleError(logging.Trace(err))
	}
	handleError(err)

	c <- text
}
