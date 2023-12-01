package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twsati/internal/bigfive"
	"twsati/internal/naming"
	"twsati/internal/sys"

	"google.golang.org/api/youtube/v3"
)

const (
	dataDirConst      = "stagingDir"
	initData          = "initMedia"
	initFromJsonConst = "initFromJson"
	basefyConst       = "normalize"
	bigfyConst        = "bigfy"
	txtfyConst        = "txtfy"
	properNameConst   = "properName"
)

var dataDir = flag.String(dataDirConst, "", "staging directory containing video files")
var initMedia = flag.Bool(initData, false, "create directory structure and place .mp4, .mp3 files into them")
var basefyFlag = flag.Bool(basefyConst, false, "recursively rename files of type .mp4, .srt, .txt to proper format")
var bigfyFlag = flag.Bool(bigfyConst, false, "recursively change file contents to big5")

// var txtfyFlag = flag.Bool(txtfyConst, false, "recursively find all srt file and create txt file out of it")
var properNameFlag = flag.Bool(properNameConst, false, "make sure file names are conforming to standard and converted to big5")
var initFromJsonArg = flag.String(initFromJsonConst, "", "init data files")
var auxProcessFlag = flag.Bool("auxProcess", false, "init data files")

func InitDataDir(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		ext := ""
		if !f.IsDir() {
			ext = filepath.Ext(f.Name())
			if !(ext == ".mp4" || ext == ".mp3") {
				fmt.Println("skipping unknown extensison for file: ", f.Name())
				continue
			}
		}
		fileOldPath := filepath.Join(path, f.Name())
		fileBaseName := bigfive.ToBig5(strings.TrimSuffix(f.Name(), ext))
		fileBaseName = naming.ProperName(fileBaseName, "")
		newPathDir := filepath.Join(path, fileBaseName)

		fileNewPath := newPathDir
		if !f.IsDir() {
			err = os.MkdirAll(newPathDir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			fileNewPath = filepath.Join(newPathDir, fileBaseName+ext)
		}
		if fileOldPath != fileNewPath {
			sys.CascadeRename(fileOldPath, fileNewPath)
		}
	}
}

func bigfy(path string) {
	file, err := os.Open(path)
	sys.CheckErr(err)
	content, err := ioutil.ReadAll(file)
	sys.CheckErr(err)
	bigContent := func() string {
		defer trace("convert file: " + path)()
		return bigfive.ToBig5(string(content))
	}()
	file.Close()
	file, err = os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0660)
	sys.CheckErr(err)
	_, err = file.WriteString(bigContent)
	sys.CheckErr(err)
}

func recurse(path string, doit func(string, fs.FileInfo)) {
	for _, f := range sys.ListFilesSorted(path, sys.TimeAsc) {
		if f.IsDir() {
			recurse(filepath.Join(path, f.Name()), doit)
		} else {
			doit(path, f)
		}
	}

}
func BigfyAll(path string) {

	recurse(path, func(basePath string, finfo fs.FileInfo) {
		fName := finfo.Name()
		if !finfo.IsDir() && (strings.HasSuffix(fName, ".txt") || strings.HasSuffix(fName, ".srt")) {
			time.Sleep(200 * time.Millisecond)
			bigfy(filepath.Join(basePath, fName))
		}
	})
}

func txtfy(path string) {
	file, err := os.Open(path)
	sys.CheckErr(err)
	scanner := bufio.NewScanner(file)
	var txt []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Trim(line, "\n\t") != "" {
			//new section
			if _, err := strconv.Atoi(line); err == nil {
				//skip time
				scanner.Scan()
				scanner.Scan()
				txt = append(txt, scanner.Text())
			}
		}
	}
	file.Close()
	newpath := strings.TrimSuffix(path, ".srt") + ".txt"
	file, err = os.OpenFile(newpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0660)
	sys.CheckErr(err)
	_, err = file.WriteString(strings.Join(txt, "\n"))
	sys.CheckErr(err)

}

func TxtfyAll(path string) {
	for _, f := range sys.ListFilesSorted(path, sys.TimeAsc) {
		if f.IsDir() {

			baseContents := sys.ListFilesSorted(filepath.Join(path, f.Name()), sys.TimeDesc)
			//sort descending by time
			for _, txtf := range baseContents {
				txtName := txtf.Name()
				if !txtf.IsDir() && strings.HasSuffix(txtName, ".srt") {
					txtfy(filepath.Join(path, f.Name(), txtName))
					time.Sleep(200 * time.Millisecond)
					break
					//break because we only txtfy the latest srt file
				}
			}
		}
	}
}

// changedF := func(files []string, ext string) bool {
// 	for i, fname := range files {
// 		if i == 0 && fname != baseName+ext {
// 			return true
// 		}
// 		if i > 0 && fname != fmt.Sprintf("bak_%d%s", i, ext) {
// 			return true
// 		}
// 	}
// 	return false
// }

// renameGroupF := func(files []string, ext string) {
// 	if !changedF(files, ext) {
// 		return
// 	}
// 	fmt.Println(baseName, ext, "changeset: ", files)

// 	tmpFiles := make([]string, 0)
// 	for i, fname := range files {
// 		oldPath := filepath.Join(basePath, fname)
// 		tmpPath := filepath.Join(basePath, fmt.Sprintf("%d%s", i, ext))
// 		err := os.Rename(oldPath, tmpPath)
// 		sys.CheckErr(err)
// 		// fmt.Println(oldPath, "->", tmpPath)
// 		time.Sleep(200 * time.Millisecond)
// 		tmpFiles = append(tmpFiles, tmpPath)
// 	}

// 	// fmt.Println(tmpFiles)
// 	for i, tmpPath := range tmpFiles {
// 		name := baseName
// 		if i > 0 {
// 			name = fmt.Sprintf("bak_%d", i)
// 		}
// 		newPath := filepath.Join(basePath, name+ext)
// 		// dependency[oldPath] = newPath
// 		err := os.Rename(tmpPath, newPath)
// 		time.Sleep(200 * time.Millisecond)
// 		sys.CheckErr(err)

// 		fmt.Println(basePath, files[i], "->", name+ext)
// 	}

// 	// for key := range dependency {
// 	// 	topoRename(key, dependency)
// 	// }

// }

func BasefyAll(path string) {

	for _, f := range sys.ListFilesSorted(path, sys.TimeAsc) {
		if f.IsDir() {
			sys.NormalizeDir(path, f)
		}
	}
}

func trace(msg string) func() {
	start := time.Now()
	log.Printf("enter %s", msg)
	return func() {
		log.Printf("exit %s (%s)", msg, time.Since(start))
	}
}

func toProperNames(dirPath string) {
	for _, f := range sys.ListFilesSorted(dirPath, sys.TimeAsc) {
		fName := f.Name()
		fName = strings.ReplaceAll(fName, " ", "")
		fName = strings.ReplaceAll(fName, "—", "-")
		fName = strings.ReplaceAll(fName, "--", "-")
		ext := filepath.Ext(fName)
		if f.IsDir() {
			ext = ""
		}
		propername := naming.ProperName(fName, ext)
		propername = bigfive.ToBig5(propername)
		if f.Name() != propername {
			fmt.Println(f.Name(), "->", propername)
			err := os.Rename(filepath.Join(dirPath, f.Name()), filepath.Join(dirPath, propername))
			sys.CheckErr(err)
		}
	}
}

func toBig5FileName(dirPath string) {
	recurse(dirPath, func(basePath string, finfo fs.FileInfo) {
		fName := finfo.Name()
		newName := bigfive.ToBig5(fName)
		if fName != newName {
			fmt.Println(fName, "->", newName)
			err := os.Rename(filepath.Join(dirPath, fName), filepath.Join(dirPath, newName))
			time.Sleep(200 * time.Millisecond)
			sys.CheckErr(err)
		}

	})

}

func processJson(jsonF string) {
	content, err := os.ReadFile(jsonF)
	sys.CheckErr(err)
	var items []youtube.PlaylistItem
	err = json.Unmarshal(content, &items)
	sys.CheckErr(err)
	re := regexp.MustCompile(`(\d\d\d\d)年(\d+)月(\d+)日`)
	for i, e := range items {
		if i <= 6 {
			continue
		}
		videoId := e.Snippet.ResourceId.VideoId
		title := e.Snippet.Title
		title = strings.ReplaceAll(title, "——隆波帕默尊者", "")
		title = strings.ReplaceAll(title, "-", "")
		title = strings.ReplaceAll(title, "微視頻", "")
		title = strings.ReplaceAll(title, "繁體中文", "")
		title = strings.TrimLeft(title, " ")
		titleParts := strings.Split(title, "｜")
		fmt.Print(videoId, " ")
		dirName := ""
		if len(titleParts) > 1 {
			dateStr := titleParts[1]
			match := re.FindAllStringSubmatch(dateStr, -1)[0]
			tmStr := fmt.Sprintf("%04d-%02d-%02d", naming.Atoi(match[1]), naming.Atoi(match[2]), naming.Atoi(match[3]))
			// fmt.Println(match[0], tmStr)
			tm, err := time.Parse("2006-01-02", tmStr)
			sys.CheckErr(err)
			dirName += tm.Format("zh060102")
		}
		dirName += titleParts[0]
		dirName += "(.-.)"
		fullPath := filepath.Join(*dataDir, dirName)
		fmt.Print(fullPath)
		fmt.Println()
		err = os.MkdirAll(fullPath, os.ModePerm)
		sys.CheckErr(err)
		jsMeta, err := os.Create(filepath.Join(fullPath, "_META_.json"))
		defer jsMeta.Close()
		sys.CheckErr(err)
		jsBytes, err := json.MarshalIndent(e, "", "    ")
		sys.CheckErr(err)
		jsMeta.WriteString(string(jsBytes))
	}
	// fmt.Println(len(strings.Split("a｜b｜c｜d", "｜")))
}

func auxProcess() {
	for _, f := range sys.ListFilesSorted(*dataDir, sys.NameAsc) {
		title := f.Name()
		ext := filepath.Ext(title)
		title = strings.TrimSuffix(title, ext)
		newfname := title + "(.-.)" + ext
		fmt.Println(newfname)
		err := os.Rename(filepath.Join(*dataDir, f.Name()), filepath.Join(*dataDir, newfname))
		sys.CheckErr(err)
	}

}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	flag.Parse()
	if *initMedia {
		InitDataDir(*dataDir)
	}
	if *properNameFlag {
		// toBig5FileName(*dataDir)
		toProperNames(*dataDir)
	}

	if *basefyFlag {
		func() {
			defer trace("BasefyAll")()
			BasefyAll(*dataDir)
		}()
	}
	if *bigfyFlag {
		BigfyAll(*dataDir)
	}
	// if *txtfyFlag {
	// 	func() {
	// 		defer trace("BasefyAll")()
	// 		TxtfyAll(*dataDir)
	// 	}()
	// }
	if isFlagPassed(initFromJsonConst) {
		processJson(*initFromJsonArg)

	}
	if *auxProcessFlag {
		auxProcess()

	}

}
