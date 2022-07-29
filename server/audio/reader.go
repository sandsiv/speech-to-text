package reader

import (
	"github.com/Alliera/logging"
	"os"

	"github.com/cryptix/wav"
)

var logger = logging.NewDefault("reader")

func getFileMeta(fileName string) wav.File {
	stat, err := os.Stat(fileName)
	checkErr(logging.Trace(err))
	file, err := os.Open(fileName)
	checkErr(logging.Trace(err))

	wavReader, err := wav.NewReader(file, stat.Size())
	checkErr(logging.Trace(err))

	return wavReader.GetFile()
}

func GetRateAndLength(fileName string) (int32, float64) {
	file := getFileMeta(fileName)
	return int32(file.SampleRate), file.Duration.Seconds()
}

func checkErr(err error) {
	if err != nil {
		logger.LogFatal(err)
	}
}
