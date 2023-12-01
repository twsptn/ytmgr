package drapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"twsati/internal/naming"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	VIDEO_ID   = "videoId"
	CAPTION_ID = "captionId"
	PRIVACY    = "privacy"
)

type VideoMeta struct {
	Title     string
	Date      time.Time
	Smin      int
	Ssec      int
	Emin      int
	Esec      int
	VideoId   *string
	Privacy   *string
	CaptionId *string

	FolderId            string
	folderName          string
	Children            []*drive.File `json:"-"`
	descriptionFilePath string        `json:"-"`
	videoFilePath       string        `json:"-"`
	captionFilePath     string        `json:"-"`
	thumbnailFilePath   string        `json:"-"`
	tempDir             string        `json:"-"`
	// Transcript      string
	// Subtitle        string
	// Tags       []string
	// Questions  []string
	// Author     string
	// Suffix     string
}

func (vmeta *VideoMeta) CleanUp() {
	os.RemoveAll(vmeta.tempDir)
}

func (vmeta *VideoMeta) ThumbnailPath() string {

	if vmeta.thumbnailFilePath == "" {
		vmeta.thumbnailFilePath = vmeta.downloadFile(".png", ".jpg")
	}
	return vmeta.thumbnailFilePath
}

func (vmeta *VideoMeta) CaptionPath() string {

	if vmeta.captionFilePath == "" {
		vmeta.captionFilePath = vmeta.downloadFile(".srt")
	}
	return vmeta.captionFilePath
}

func (vmeta *VideoMeta) DescriptionPath() string {

	if vmeta.descriptionFilePath == "" {
		vmeta.descriptionFilePath = vmeta.downloadFile(".txt")
	}
	return vmeta.descriptionFilePath
}
func (vmeta *VideoMeta) VideoFilePath() string {

	if vmeta.videoFilePath == "" {
		vmeta.videoFilePath = vmeta.downloadFile(".mp4")
	}
	return vmeta.videoFilePath
}

func (vmeta *VideoMeta) DescriptionContent() string {
	if !vmeta.HasDescription() {
		return ""
	}
	path := vmeta.DescriptionPath()
	f, err := os.Open(path)
	handleError(err, "open description file: "+path)
	defer f.Close()
	payload, err := ioutil.ReadAll(f)
	handleError(err, "load description contenbt: "+path)
	return string(payload)
}
func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func (vmeta *VideoMeta) HasDescription() bool {
	return vmeta.HasExt(".txt")
}

func (vmeta *VideoMeta) HasCaption() bool {
	return vmeta.HasExt(".srt")
}

func (vmeta *VideoMeta) HasVideo() bool {
	return vmeta.HasExt(".mp4")
}

func (vmeta *VideoMeta) HasExt(ext string) bool {
	for _, f := range vmeta.Children {
		if /*strings.HasPrefix(f.Name, vmeta.folderName) &&*/ strings.HasSuffix(f.Name, ext) {
			return true
		}
	}
	return false

}

func (vmeta *VideoMeta) SetTempDir(dir string) {
	vmeta.tempDir = dir
}

func (vmeta *VideoMeta) downloadFile(exts ...string) string {

	if vmeta.tempDir == "" {
		// creating temp dir
		dir, err := ioutil.TempDir(os.TempDir(), vmeta.Title)
		handleError(err, "creating tmp dir:  "+dir)
		vmeta.tempDir = dir
	}

	lastModTime := ""
	var candidateFile *drive.File
	for _, f := range vmeta.Children {
		for _, ext := range exts {
			if /*strings.HasPrefix(f.Name, vmeta.folderName) &&*/ strings.HasSuffix(f.Name, ext) {
				if f.ModifiedTime > lastModTime {
					candidateFile = f
					lastModTime = f.ModifiedTime
					fmt.Println("found better candidate :", f.Name, f.ModifiedTime)

				}
			}
		}
	}
	if candidateFile != nil {
		// bingo, load description
		path := downloadFileTo(vmeta.tempDir, candidateFile)
		return path
	} else {
		panic("failed to download for ext: " + strings.Join(exts, ","))
	}
}

func fromString(str string) *VideoMeta {

	info := naming.ExtractName2(str)
	meta := &VideoMeta{}
	meta.Date = info.Date
	meta.Title = info.Title
	meta.Smin = info.Smin
	meta.Ssec = info.Ssec
	meta.Emin = info.Emin
	meta.Esec = info.Esec
	return meta
}

var service *drive.Service

func init() {

	var err error
	cli := getClient(drive.DriveScope)
	service, err = drive.New(cli)
	handleError(err, "drive cli initialization")

}

func getClient(scope string) *http.Client {
	ctx := context.Background()
	// driveService, err := drive.NewService(ctx)

	b, err := ioutil.ReadFile("client_secret_drive.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/youtube-go-quickstart.json
	config, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)

	// token, err := config.Exchange(ctx, ...)
	// driveService, err := drive.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func DriveFolders() {

	call := service.Files.List().
		// Q("mimeType='application/vnd.google-apps.folder'").
		// Q("name='zh221001_[34.20-37.14]_分離五蘊，看見“我”不存在'").
		Q(fmt.Sprintf("name='%s'", "zh221001_[34.20-37.14]_分離五蘊，看見“我”不存在")).
		Fields("files/*").
		Spaces("drive")
	resp, err := call.Do()
	handleError(err, "list call()")
	for _, f := range resp.Files {
		fmt.Println(f.Name)

	}
}
func driveFolderListByName(name string) (*drive.File, []*drive.File) {
	fmt.Println("query for folder: ", name)
	call := service.Files.List().
		Q(fmt.Sprintf("name='%s'", name)).
		// Fields("id", "name", "description", "appProperties").
		Fields("files/*").
		Spaces("drive")

	resp, err := call.Do()
	handleError(err, "list call()")
	if len(resp.Files) > 1 {
		panic("folder name not unique " + name)
	} else if len(resp.Files) == 0 {
		panic("folder name not found" + name)

	}
	return resp.Files[0], driveFolderListById(resp.Files[0].Id)
}

func driveFolderListById(folderId string) []*drive.File {
	call := service.Files.List().
		// Q("title='zh230114_[37.34-38.51]_生命中別投降別氣餒'").
		Q(fmt.Sprintf("'%s' in parents", folderId)).
		Fields("files/*")
		// Fields("files/name", "files/trashed", "files/id")
		// Spaces("drive")

	resp, err := call.Do()
	handleError(err, "list call()")
	return resp.Files
}

func HelloDrive() {
	resp, err := service.About.Get().Fields("user").Do()
	handleError(err, "drive about()")
	fmt.Printf("This drive is owned by: %s, and email: %s\n", resp.User.DisplayName, resp.User.EmailAddress)
}

func downloadFileTo(dir string, f *drive.File) string {
	resp, err := service.Files.Get(f.Id).Download()
	handleError(err, "drive download")
	defer resp.Body.Close()
	newF := filepath.Join(dir, f.Name)
	descF, err := os.Create(newF)
	handleError(err, "create description file")
	io.Copy(descF, resp.Body)
	return newF
}

func setSptr(ptr **string, rvalue string) {
	if *ptr == nil {
		*ptr = new(string)
	}
	**ptr = rvalue
}
func GetVideoMeta(name string) *VideoMeta {

	var hasKey = func(dict map[string]string, key string) bool {
		if dict != nil {
			if _, ok := dict[key]; ok {
				return true
			}
		}
		return false
	}
	vmeta := fromString(name)
	vmeta.folderName = name
	folder, children := driveFolderListByName(name)
	// fmt.Printf("%+v\n", folder)
	vmeta.FolderId = folder.Id
	// fmt.Println(folder.Description, folder.AppProperties)
	if hasKey(folder.AppProperties, VIDEO_ID) {
		setSptr(&vmeta.VideoId, folder.AppProperties[VIDEO_ID])
	}
	if hasKey(folder.AppProperties, CAPTION_ID) {
		setSptr(&vmeta.CaptionId, folder.AppProperties[CAPTION_ID])
	}
	if hasKey(folder.AppProperties, PRIVACY) {
		setSptr(&vmeta.Privacy, folder.AppProperties[PRIVACY])
	}
	for _, f := range children {
		if !f.Trashed {
			vmeta.Children = append(vmeta.Children, f)
		}
	}

	return vmeta
}

func UpdateVideoMeta(vmeta *VideoMeta) {

	// update meta
	nf := &drive.File{Description: prettyPrint(vmeta)}
	nf.AppProperties = make(map[string]string)
	if vmeta.VideoId != nil {
		nf.AppProperties[VIDEO_ID] = *vmeta.VideoId
	}

	if vmeta.CaptionId != nil {
		nf.AppProperties[CAPTION_ID] = *vmeta.CaptionId
	}

	if vmeta.Privacy != nil {
		nf.AppProperties[PRIVACY] = *vmeta.Privacy
	}

	fmt.Println("Writing App properties")
	fmt.Println(nf.AppProperties)
	_, err := service.Files.Update(vmeta.FolderId, nf).Do()
	handleError(err, "write meta")
}

func main() {
	// call := driveService.Files.List().Q("mimetype='application/vnd.google-apps.folder'")
	// Q("mimeType='application/vnd.google-apps.folder'").
	folderId := "1W1_eweowezPS-0rhUrN-TMEWaCK0HA38"
	f, err := service.Files.Get(folderId).Fields("id", "name", "description", "appProperties").Do()
	handleError(err, "read meta")
	fmt.Println(f.Description, f.AppProperties)
	nf := &drive.File{Description: "description=abcdef"}
	nf.AppProperties = make(map[string]string)
	nf.AppProperties["youtubeId"] = "abcdef"
	f, err = service.Files.Update(folderId, nf).Do()
	handleError(err, "write meta")

	// zh230114_[37.34-38.51]_生命中別投降別氣餒
	// zh230101_[08.33-10.19]_無論生活還是修行，必須小心自己的念頭，照顧好自己的心

	// call := service.About
	// resp, _ := call.Get().Do()
	// fmt.Println(resp.StorageQuota.Usage)
	// for _, file := range driveFolderListByName("zh230114_[37.34-38.51]_生命中別投降別氣餒") {
	// 	fmt.Println(file.Id, file.Name, file.MimeType, file.Trashed)
	// }

	// fmt.Println(resp)
}
