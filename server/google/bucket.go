package google

import (
	"bytes"
	"cloud.google.com/go/storage"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]

	}
	return string(b)
}

func DeleteFile(link string) error {
	parts := strings.Split(link, "/")
	object := parts[len(parts)-1]
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket(os.Getenv("BUCKET_NAME")).Object(object)
	if err := o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}
	fmt.Println(link + " was removed")
	return nil
}

func WriteToCloudStorage(url string) (gsUrl string, localFilePath string, error error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	var buf bytes.Buffer
	body := io.TeeReader(resp.Body, &buf)
	fileName := randomString(8) + ".wav"

	defer resp.Body.Close()
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", "", err
	}
	bkt := client.Bucket(os.Getenv("BUCKET_NAME"))

	wc := bkt.Object(fileName).NewWriter(ctx)
	wc.ContentType = "audio/wave"
	_, err = io.Copy(wc, body)
	if err != nil {
		return "", "", err
	}
	err = wc.Close()
	if err != nil {
		return "", "", err
	}
	gsName := "gs://" + os.Getenv("BUCKET_NAME") + "/" + fileName
	fmt.Println("File " + gsName + " was uploaded to google bucket")

	localFilePath = "/tmp/" + fileName
	file, err := os.Create(localFilePath)
	defer file.Close()
	if err != nil {
		return "", "", err
	}
	_, err = io.Copy(file, &buf)

	return gsName, localFilePath, nil
}
