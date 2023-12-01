package bigfive

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/liuzl/gocc"
)

var s2t *gocc.OpenCC

type base64Loader struct {
}

func (base64Loader) Open(configFile string) (io.ReadCloser, error) {

	fmt.Println(configFile)
	return io.NopCloser(bytes.NewReader(openZippedFile(configFile))), nil
}

var zipStream []byte

func init() {
	var err error
	zipStream, err = base64.StdEncoding.DecodeString(configData)
	if err != nil {
		log.Fatal(err)
	}

	// s2t, err = gocc.New("s2tw") //gocc.WithDir("/home/kailee/go/src/github.com/liuzl/gocc"))
	s2t, err = gocc.New("s2tw", gocc.WithLoader(base64Loader{})) //gocc.WithDir("/home/kailee/go/src/github.com/liuzl/gocc"))
	if err != nil {
		log.Fatal(err)
	}
	// s2t.Convert("abc")
}

func openZippedFile(fileToOpen string) []byte {

	zipReader, err := zip.NewReader(bytes.NewReader(zipStream), int64(len(zipStream)))
	if err != nil {
		log.Fatal(err)
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		if zipFile.Name == strings.TrimSpace(fileToOpen) {
			fmt.Println("Reading file:", zipFile.Name)
			unzippedFileBytes, err := readZipFile(zipFile)
			if err != nil {
				log.Fatal(err)
			}

			return unzippedFileBytes
			// fmt.Println(string(unzippedFileBytes))
			// _ = unzippedFileBytes // this is unzipped file bytes

		}
	}
	log.Fatal("can't open file: " + fileToOpen)
	// won't reach here
	return nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func ToBig5(s string) string {

	ret, err := s2t.Convert(s)
	if err != nil {
		log.Fatal(err)
	}
	return ret
}
