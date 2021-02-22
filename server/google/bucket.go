package google

import (
	"bytes"
	"cloud.google.com/go/storage"
	"errors"
	"fmt"
	"github.com/Alliera/speech-to-text/server"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func DeleteFile(link string, enterpriseId int) error {
	parts := strings.Split(link, "/")
	object := parts[len(parts)-1]
	ctx := context.Background()
	client, err := storage.NewClient(ctx, GetCredentials(enterpriseId))
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket(GetBucketName(enterpriseId)).Object(object)
	if err := o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}
	fmt.Println(link + " was removed")
	return nil
}

func WriteToCloudStorage(url string, enterpriseId int) (gsUrl string, localFilePath string, error error) {
	fmt.Println("Start downloading file " + url)
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != 200 {
		return "", "", errors.New("Wrong status code (" + strconv.Itoa(resp.StatusCode) + ") for url " + url)
	}
	fmt.Println("Start uploading file to gc " + url)
	var buf bytes.Buffer
	body := io.TeeReader(resp.Body, &buf)
	fileName := server.RandomString(8) + ".wav"

	defer resp.Body.Close()
	ctx := context.Background()
	client, err := storage.NewClient(ctx, GetCredentials(enterpriseId))
	if err != nil {
		return "", "", err
	}
	bucketName := GetBucketName(enterpriseId)
	bkt := client.Bucket(bucketName)

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
	gsName := "gs://" + bucketName + "/" + fileName
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
