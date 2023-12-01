package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	drapi "twsati/internal/google/drive"
	"twsati/internal/sys"
)

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "   ")
	return string(s)
}

func setSptr(ptr **string, rvalue string) {
	if *ptr == nil {
		*ptr = new(string)
	}
	**ptr = rvalue
}

func dumpFolderUrl(name string) {
	vmeta := drapi.GetVideoMeta(name)
	url := fmt.Sprintf("https://drive.google.com/drive/folders/%s", vmeta.FolderId)
	fmt.Println(url)
	cli := exec.Command("explorer", url)
	err := cli.Run()
	sys.CheckErr(err)

}
func dumpMeta(name string) {
	vmeta := drapi.GetVideoMeta(name)
	// vmeta.CaptionPath()
	// defer vmeta.CleanUp()

	// vmeta.VideoId = "EViH9AYi6UM"
	// vmeta.Privacy = "unlisted"
	// drapi.UpdateVideoMeta(vmeta)
	// fmt.Printf("%s\n", prettyPrint(vmeta))
	// fmt.Printf("%+v\n", vmeta)
	fmt.Println(prettyPrint(vmeta))
}

func download(name string, localRoot string) {
	vmeta := drapi.GetVideoMeta(name)

	path := filepath.Join(localRoot, name)
	err := os.MkdirAll(path, os.ModePerm)
	sys.CheckErr(err)
	vmeta.SetTempDir(path)
	if vmeta.HasCaption() {
		vmeta.CaptionPath()
	}
	if vmeta.HasDescription() {

		vmeta.DescriptionPath()
	}
	if vmeta.HasVideo() {
		vmeta.VideoFilePath()
	}
}

func upload(name string, localRoot string) {

}

type privacy int

const (
	UNLISTED privacy = iota
	PUBLIC
)

func (p privacy) string() string {
	return []string{"unlisted", "public"}[p]
}

var helloFlag = flag.Bool("hello", false, "hello")
var dumpFlag = flag.String("dump", "", "video clip name")
var downloadFlag = flag.String("download", "", "video clip name")
var urlFlag = flag.String("url", "", "video clip name")
var stagingDir = flag.String("stagingDir", "", "working directory")

func main() {
	flag.Parse()
	if *helloFlag {
		drapi.HelloDrive()

		fmt.Println("Hello Google Drive!!")

	} else if *dumpFlag != "" {
		// youtubeUpload(*uploadFlag)
		// EViH9AYi6UM
		dumpMeta(*dumpFlag)
	} else if *urlFlag != "" {
		dumpFolderUrl(*urlFlag)

	} else if *downloadFlag != "" {
		download(*downloadFlag, ".")
	} else {
		flag.PrintDefaults()
		drapi.DriveFolders()

	}

}
