package google

import (
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

func WriteToCloudStorage(url string) (gsUrl string, filePath string, error error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	fileName := randomString(8) + ".wav"
	file, err := os.Create(fileName)
	if err != nil {
		return "", "", err
	}
	_, err = io.Copy(file, resp.Body)
	//defer resp.Body.Close()
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", "", err
	}
	bkt := client.Bucket(os.Getenv("BUCKET_NAME"))

	wc := bkt.Object(fileName).NewWriter(ctx)
	wc.ContentType = "audio/wave"
	_, err = io.Copy(wc, resp.Body)
	if err != nil {
		return "", "", err
	}
	err = wc.Close()
	if err != nil {
		return "", "", err
	}
	gsUrl = "gs://" + os.Getenv("BUCKET_NAME") + "/" + fileName
	fmt.Println("File " + fileName + " was uploaded to google bucket")

	return gsUrl, file.Name(), nil
}
