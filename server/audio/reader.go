package reader

import (
	"os"

	"github.com/cryptix/wav"
)

func getFileMeta(fileName string) wav.File {
	stat, err := os.Stat(fileName)
	checkErr(err)
	file, err := os.Open(fileName)
	checkErr(err)

	wavReader, err := wav.NewReader(file, stat.Size())
	checkErr(err)

	return wavReader.GetFile()
}

func GetRateAndLength(fileName string) (int32, float64) {
	file := getFileMeta(fileName)
	return int32(file.SampleRate), file.Duration.Seconds()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
