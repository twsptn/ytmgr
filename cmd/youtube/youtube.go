package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	drapi "twsati/internal/google/drive"
	ytapi "twsati/internal/google/youtube"
)

// ytId := ytapi.UploadVideo(upld.Title, upld.Transcript, "27", "meditation", videoPath)
// upld.VideoId = ytId
// upld.Privacy = "unlisted"
// updateVideoId(db, upld)
// updatePrivacy(db, upld)

func wrapTitle(vmeta *drapi.VideoMeta) string {
	return fmt.Sprintf("%s-%s (%s) ｜ %s", "微視頻", vmeta.Title, "繁體中文", vmeta.Date.Format("2006年01月02日"))
}

func wrapDesc(vmeta *drapi.VideoMeta) string {
	titleStr := "【" + vmeta.Title + "】"
	rangeStr := fmt.Sprintf("%02d'%02d\" ~ %02d'%02d\"", vmeta.Smin, vmeta.Ssec, vmeta.Emin, vmeta.Esec)
	addendum := `聽錄、摘錄自` + vmeta.Date.Format("2006年01月02日") + "直播開示" //+ 15:03～24:24
	footer := `
本文內容是根據尊者直播視頻聽錄、整理而成，文字未經尊者及譯者審校，若內容有任何疏失，皆歸咎於聽錄、整理者的責任與過失。
直播同聲翻譯｜坤能•禪窗
文字整理｜台灣四念處學會`
	return fmt.Sprintf("%s\n\n%s\n\n%s %s%s", titleStr, vmeta.DescriptionContent(), addendum, rangeStr, footer)
}

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

func setMeta(name string, vidId *string, capId *string, privacy *string) {

	vmeta := drapi.GetVideoMeta(name)
	vmeta.VideoId = vidId
	vmeta.CaptionId = capId
	vmeta.Privacy = privacy
	drapi.UpdateVideoMeta(vmeta)

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

func youtubeDeleteCaption(name string) {
	vmeta := drapi.GetVideoMeta(name)
	resp := ytapi.ListCaption(*vmeta.VideoId)
	for _, item := range resp.Items {
		ytapi.DeleteCaption(item.Id)
		fmt.Printf("successfully deleted youtube video caption %s id: %s  for video %s\n", item.Snippet.Language, item.Id, vmeta.Title)
	}
	setSptr(&vmeta.CaptionId, "")
	drapi.UpdateVideoMeta(vmeta)
}

func youtubeCaption(name string) {
	vmeta := drapi.GetVideoMeta(name)
	captionId := ""
	if vmeta.CaptionId != nil {
		captionId = *vmeta.CaptionId
	}
	setSptr(&vmeta.CaptionId, ytapi.UploadCaption(captionId, *vmeta.VideoId, "zh-tw", "繁體", vmeta.CaptionPath()))
	fmt.Println("updated youtube video caption id: ", *vmeta.CaptionId)
	drapi.UpdateVideoMeta(vmeta)
}

type privacy int

const (
	UNLISTED privacy = iota
	PUBLIC
)

func (p privacy) string() string {
	return []string{"unlisted", "public"}[p]
}

func youtubeUpdateVideo(name string, priv privacy) {
	vmeta := drapi.GetVideoMeta(name)
	defer vmeta.CleanUp()
	ytId := ytapi.UpdateVideo(*vmeta.VideoId, wrapTitle(vmeta), wrapDesc(vmeta), priv.string(), "")
	fmt.Printf("updated youtube video: %s id: %s, status: %s\n", vmeta.Title, ytId, priv.string())
	setSptr(&vmeta.Privacy, priv.string())
	drapi.UpdateVideoMeta(vmeta)

}

func youtubeUploadCover(name string) {
	vmeta := drapi.GetVideoMeta(name)
	// vmeta.SetTempDir(`C:\Users\kaile\AppData\Local\Temp\生命中別投降別氣餒2919784654`)
	defer vmeta.CleanUp()

	ytapi.UploadCover(*vmeta.VideoId, vmeta.ThumbnailPath())
}

func youtubeUpload(name string, overWriteExisting bool) {
	vmeta := drapi.GetVideoMeta(name)
	// vmeta.SetTempDir(`C:\Users\kaile\AppData\Local\Temp\生命中別投降別氣餒2919784654`)
	defer vmeta.CleanUp()
	// ytapi.UploadVideo()
	if vmeta.VideoId != nil && len(strings.TrimSpace(*vmeta.VideoId)) > 0 {
		if overWriteExisting {
			ytapi.DeleteVideo(*vmeta.VideoId)

			setSptr(&vmeta.VideoId, "")
			setSptr(&vmeta.CaptionId, "")
			setSptr(&vmeta.Privacy, "")
			drapi.UpdateVideoMeta(vmeta)
		} else {
			panic("upload an existing video: x" + *vmeta.VideoId + "x")
		}
	}

	description := vmeta.DescriptionContent()
	// if description == "" {
	// 	description = "empty description"
	// }
	// fmt.Printf("%+v\n %s\n", vmeta, description)
	vidId := ytapi.UploadVideo(vmeta.Title, description, "27", "meditation", vmeta.VideoFilePath())
	// upld.VideoId = ytId
	// upld.Privacy = "unlisted"
	// updateVideoId(db, upld)
	// updatePrivacy(db, upld)
	setSptr(&vmeta.VideoId, vidId)
	setSptr(&vmeta.Privacy, "unlisted")
	drapi.UpdateVideoMeta(vmeta)

}

var helloFlag = flag.Bool("hello", false, "hello")
var dumpFlag = flag.String("dump", "", "video clip name")
var setMetaFlag = flag.String("setMeta", "", "video clip name")
var metaKeys = flag.String("metaKeys", "", "CaptionId=xxxx;VideoId=xxxx;Privacy=xxx")
var uploadFlag = flag.String("upload", "", "video clip name")
var reUploadFlag = flag.String("reUpload", "", "video clip name")
var uploadCoverFlag = flag.String("uploadCover", "", "video clip name")
var captionFlag = flag.String("caption", "", "video clip name")
var captionDeleteFlag = flag.String("captionDelete", "", "video clip name")
var publishFlag = flag.String("publish", "", "video clip name")
var unlistFlag = flag.String("unlist", "", "video clip name")

func mapfromString(str string) map[string]*string {
	ret := make(map[string]*string)
	entries := strings.Split(str, ";")
	for _, e := range entries {
		parts := strings.Split(e, "=")
		k := strings.Trim(parts[0], " ")
		v := strings.Trim(parts[1], " ")
		ret[k] = &v
	}

	return ret

}

func main() {
	flag.Parse()
	if *helloFlag {
		ytapi.ChannelsListById("snippet,contentDetails,statistics", "UCrCmgRwcNRhuMEtpoH-VVWg")
		drapi.HelloDrive()

		fmt.Println("Hello Youtube!!\nHello Google Drive!!")

	} else if *dumpFlag != "" {
		// youtubeUpload(*uploadFlag)
		// EViH9AYi6UM
		fmt.Println("Dumping meta for: ", *dumpFlag)
		dumpMeta(*dumpFlag)
	} else if *setMetaFlag != "" {
		m := mapfromString(*metaKeys)

		setMeta(*setMetaFlag, m["VideoId"], m["CaptionId"], m["Privacy"])
	} else if *uploadFlag != "" {
		youtubeUpload(*uploadFlag, false)
	} else if *reUploadFlag != "" {
		youtubeUpload(*reUploadFlag, true)
	} else if *uploadCoverFlag != "" {
		youtubeUploadCover(*uploadCoverFlag)
	} else if *captionFlag != "" {
		youtubeCaption(*captionFlag)
	} else if *captionDeleteFlag != "" {
		youtubeDeleteCaption(*captionDeleteFlag)
	} else if *publishFlag != "" {
		youtubeUpdateVideo(*publishFlag, PUBLIC)
	} else if *unlistFlag != "" {
		youtubeUpdateVideo(*unlistFlag, UNLISTED)
	} else {
		flag.PrintDefaults()
		drapi.DriveFolders()

		// dir, _ := ioutil.TempDir(os.TempDir(), "zh20939Talk")
		// // defer os.RemoveAll(dir)
		// fmt.Println(dir)
	}

}
