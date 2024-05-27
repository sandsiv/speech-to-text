package reader

import (
	"github.com/cryptix/wav"
	"os"
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

func CreateFile(filename string) (*os.File, *wav.Writer) {
	wavOut, err := os.Create(filename)
	checkErr(err)

	meta := wav.File{
		Channels:        1,
		SampleRate:      8000,
		SignificantBits: 16,
	}

	writer, err := meta.NewWriter(wavOut)
	checkErr(err)

	return wavOut, writer
}
