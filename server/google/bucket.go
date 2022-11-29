package google

import (
	"bytes"
	"cloud.google.com/go/storage"
	"fmt"
	"github.com/Alliera/logging"
	"github.com/Alliera/speech-to-text/server"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func DeleteFile(link string, enterpriseId int) error {
	parts := strings.Split(link, "/")
	object := parts[len(parts)-1]
	ctx := context.Background()
	client, err := storage.NewClient(ctx, GetCredentials(enterpriseId))
	if err != nil {
		return logging.Trace(fmt.Errorf("storage.NewClient: %s", err))
	}
	defer func() {
		err = client.Close()
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket(GetBucketName(enterpriseId)).Object(object)
	if err = o.Delete(ctx); err != nil {
		return logging.Trace(fmt.Errorf("Object(%q).Delete: %s", object, err))
	}
	fmt.Println(link + " was removed")
	return nil
}

func WriteToCloudStorage(url string, enterpriseId int) (gsUrl string, localFilePath string, error error) {
	fmt.Println("Start downloading file " + url)
	resp, err := http.Get(url)
	if err != nil {
		return "", "", logging.Trace(err)
	}
	if resp.StatusCode != 200 {
		return "", "", logging.Trace(fmt.Errorf("wrong status code (%d) for url %s", resp.StatusCode, url))
	}
	fmt.Println("Start uploading file to gc " + url)
	var buf bytes.Buffer
	body := io.TeeReader(resp.Body, &buf)
	fileName := server.RandomString(8) + ".wav"

	defer func() {
		err = resp.Body.Close()
	}()
	ctx := context.Background()
	client, err := storage.NewClient(ctx, GetCredentials(enterpriseId))
	if err != nil {
		return "", "", logging.Trace(err)
	}
	bucketName := GetBucketName(enterpriseId)
	bkt := client.Bucket(bucketName)

	wc := bkt.Object(fileName).NewWriter(ctx)
	wc.ContentType = "audio/wave"
	_, err = io.Copy(wc, body)
	if err != nil {
		_ = client.Close()
		return "", "", logging.Trace(err)
	}
	err = wc.Close()
	_ = client.Close()
	if err != nil {
		return "", "", logging.Trace(err)
	}
	gsName := "gs://" + bucketName + "/" + fileName
	fmt.Println("File " + gsName + " was uploaded to google bucket")

	localFilePath = "/tmp/" + fileName
	file, err := os.Create(localFilePath)

	defer func() {
		err = file.Close()
	}()
	if err != nil {
		return "", "", logging.Trace(err)
	}
	_, _ = io.Copy(file, &buf)

	return gsName, localFilePath, nil
}
