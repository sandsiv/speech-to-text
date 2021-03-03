package google

import (
	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"github.com/Alliera/speech-to-text/server"
	"github.com/Alliera/speech-to-text/server/dto"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"io/ioutil"
	"os"
	"strconv"
)

const configPath = "config/"
const bucketsFilePath = configPath + "buckets.json"
const credentialsPath = configPath + "credentials/"

func GetCredentials(enterpriseId int) option.ClientOption {
	credentials := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if enterpriseId != 0 {
		path := credentialsPath + strconv.Itoa(enterpriseId) + ".json"
		if _, err := os.Stat(path); err == nil {
			credentials = path
		}
	}
	return option.WithCredentialsFile(credentials)
}

func GetBucketName(enterpriseId int) string {
	bucketName := os.Getenv("BUCKET_NAME")
	if enterpriseId == 0 {
		return bucketName
	}
	if _, err := os.Stat(bucketsFilePath); err == nil {
		content, err := ioutil.ReadFile(bucketsFilePath)
		if err != nil {
			panic(err)
		}
		var buckets map[int]string
		err = json.Unmarshal(content, &buckets)
		if err != nil {
			panic(err)
		}
		if bucketFromConfig, ok := buckets[enterpriseId]; ok {
			bucketName = bucketFromConfig
		}
	}

	return bucketName
}

func AddBucketName(enterpriseId int, bucketName string) error {
	buckets := make(map[int]string)
	if _, err := os.Stat(bucketsFilePath); err == nil {
		content, err := ioutil.ReadFile(bucketsFilePath)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(content, &buckets)
		if err != nil {
			panic(err)
		}
	}
	buckets[enterpriseId] = bucketName
	text, err := json.Marshal(buckets)
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll(configPath, 0700)
		if err != nil {
			return err
		}
	}
	err = writeFile(bucketsFilePath, text)
	if err != nil {
		return err
	}

	return nil
}

func AddCredentials(enterpriseId int, credentials map[string]string) error {
	text, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	path := credentialsPath + strconv.Itoa(enterpriseId) + ".json"
	err = saveFile(credentialsPath, path, text)
	if err != nil {
		return err
	}

	return nil
}

func saveFile(path string, filePath string, text []byte) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0700)
		if err != nil {
			return err
		}
	}

	err := writeFile(filePath, text)
	if err != nil {
		return err
	}

	return nil
}

func writeFile(fileName string, data []byte) error {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func CheckCredentials(credentials dto.Credentials) error {
	tmpPath := "tmp/"
	tempFile := tmpPath + server.RandomString(5) + ".json"
	text, err := json.Marshal(credentials.Credentials)
	if err != nil {
		return err
	}
	err = saveFile(tmpPath, tempFile, text)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client, err := speech.NewClient(ctx, option.WithCredentialsFile(tempFile))
	if err != nil {
		_ = os.Remove(tempFile)
		return err
	}
	_, err = client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: "gs://cloud-samples-data/speech/brooklyn_bridge.raw"},
		},
	})
	_ = client.Close()

	if err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	clientBucket, err := storage.NewClient(ctx, option.WithCredentialsFile(tempFile))
	if err != nil {
		_ = os.Remove(tempFile)
		return err
	}
	ctx = context.Background()
	bkt := clientBucket.Bucket(credentials.BucketName)
	it := bkt.Objects(ctx, nil)
	_, err = it.Next()
	_ = clientBucket.Close()

	if err != nil {
		_ = os.Remove(tempFile)
		return err
	}
	_ = os.Remove(tempFile)

	return nil
}
