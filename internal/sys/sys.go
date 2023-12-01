package sys

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SortOrder uint

const (
	TimeAsc SortOrder = iota
	TimeDesc
	NameAsc
	NameDesc
)

func CheckErr(err error) {
	if err != nil {
		log.Panicf("%T: %s", err, err)
		// panic(err)
		// log.Fatal(err)
	}
}

func CascadeRename(fromPath, toPath string) {
	if _, err := os.Stat(toPath); !os.IsNotExist(err) {
		//file exist
		// fmt.Println("exist", fromPath, ":", toPath)
		suffix := time.Now().Format("_060102_1504-05")

		fileName := filepath.Base(toPath)
		fileExt := filepath.Ext(fileName)
		fileBase := strings.TrimSuffix(fileName, fileExt)
		newFileName := fmt.Sprintf("%s%s%s", fileBase, suffix, fileExt)
		newPath := filepath.Join(filepath.Dir(toPath), newFileName)
		CascadeRename(toPath, newPath)
	}
	time.Sleep(2 * time.Second)
	fmt.Println(fromPath, "->", toPath)
	err := os.Rename(fromPath, toPath)
	CheckErr(err)
	currentTime := time.Now().Local()
	err = os.Chtimes(toPath, currentTime, currentTime)
	if err != nil {
		CheckErr(err)
	}

}

func NormalizeDir(root string, f fs.FileInfo) {
	baseName := f.Name()
	basePath := filepath.Join(root, baseName)
	baseContents := ListFilesSorted(filepath.Join(root, f.Name()), TimeDesc)

	var txtFs []string
	var srtFs []string
	var mp4Fs []string
	var mp3Fs []string
	for _, f := range baseContents {
		f.Name()

		if !f.IsDir() {
			ext := filepath.Ext(f.Name())
			// idxName := fmt.Sprintf("%d%s", i, ext)
			switch ext {
			case ".txt":
				txtFs = append(txtFs, f.Name())
			case ".srt":
				srtFs = append(srtFs, f.Name())
			case ".mp4":
				mp4Fs = append(mp4Fs, f.Name())
			case ".mp3":
				mp3Fs = append(mp3Fs, f.Name())
			default:
				log.Println("unknown file extension: ", f.Name())
				continue
			}
			// os.Rename(filepath.Join(basePath, f.Name()), filepath.Join(basePath, idxName))
		}
	}

	changedF := func(files []string, ext string) bool {
		// fmt.Println(files)
		if len(files) > 0 {
			return files[0] != baseName+ext
		}
		return false
	}

	renameGroupF := func(files []string, ext string) {
		if !changedF(files, ext) {
			return
		}
		fmt.Println("changeset: ", files)

		oldPath := filepath.Join(basePath, files[0])
		newPath := filepath.Join(basePath, baseName+ext)
		CascadeRename(oldPath, newPath)

	}

	renameGroupF(txtFs, ".txt")
	renameGroupF(srtFs, ".srt")
	renameGroupF(mp4Fs, ".mp4")
	renameGroupF(mp3Fs, ".mp3")

}
func ListFilesSorted(path string, order SortOrder) []fs.FileInfo {
	files, err := ioutil.ReadDir(path)
	CheckErr(err)
	var orderFunc func(int, int) bool

	nameAsc := func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	}

	timeAsc := func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime()) //.Before(baseContents)
	}
	if order == NameAsc {
		orderFunc = nameAsc
	} else if order == NameDesc {
		orderFunc = func(i, j int) bool { return !nameAsc(i, j) }
	} else if order == TimeDesc {
		orderFunc = func(i, j int) bool { return !timeAsc(i, j) }
	} else if order == TimeAsc {
		orderFunc = timeAsc
	} else {
		log.Panic("unknown sort order")
	}
	sort.Slice(files, orderFunc)
	return files
}
