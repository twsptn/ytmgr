package ytapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"twsati/internal/sys"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

const missingClientSecretsMessage = `
Please configure OAuth 2.0
`

var service *youtube.Service

func init() {
	client := getClient(youtube.YoutubeForceSslScope)
	var err error
	service, err = youtube.New(client)
	handleError(err, "Error creating YouTube client")

}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(scope string) *http.Client {
	ctx := context.Background()
	usr, err := user.Current()
	sys.CheckErr(err)

	b, err := ioutil.ReadFile(filepath.Join(usr.HomeDir, "client_secret.json"))
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
		url.QueryEscape("youtube-go-quickstart.json")), err
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

// func startWebServer() (codeCh chan string, err error) {
// 	listener, err := net.Listen("tcp", "127.0.0.1:8099")
// 	if err != nil {
// 		log.Panic(err)
// 	}
// 	codeCh = make(chan string)

// 	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		code := r.FormValue("code")
// 		codeCh <- code // send code to OAuth flow
// 		listener.Close()
// 		w.Header().Set("Content-Type", "text/plain")
// 		fmt.Fprintf(w, "Received code: %v\r\nYou can now safely close this browser window.", code)
// 	}))

// 	return codeCh, nil
// }

func ChannelsListById(part string, id string) {
	call := service.Channels.List(strings.Split(part, ","))
	call = call.Id(id)
	response, err := call.Do()
	handleError(err, "")
	fmt.Println(fmt.Sprintf("This channel's ID is %s. Its title is '%s', "+
		"and it has %d views.",
		response.Items[0].Id,
		response.Items[0].Snippet.Title,
		response.Items[0].Statistics.ViewCount))
}

func ChannelsListByUsername(part string, forUsername string) {
	call := service.Channels.List(strings.Split(part, ","))
	call = call.ForUsername(forUsername)
	response, err := call.Do()
	handleError(err, "")
	fmt.Println(fmt.Sprintf("This channel's ID is %s. Its title is '%s', "+
		"and it has %d views.",
		response.Items[0].Id,
		response.Items[0].Snippet.Title,
		response.Items[0].Statistics.ViewCount))
}

func PlaylistsItemsAll(part string, playlistId string) []*youtube.PlaylistItem {
	pageToken := ""
	resp := PlaylistsItems(part, playlistId, pageToken)
	var retItems []*youtube.PlaylistItem
	for {

		retItems = append(retItems, resp.Items...)
		// for _, item := range resp.Items {
		// 	fmt.Println(item.Id, item.Snippet.Title, item.Snippet.Description)
		// }
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
		resp = PlaylistsItems(part, playlistId, pageToken)
	}
	return retItems

}

func PlaylistsItemDelete(itemId string) {

	call := service.PlaylistItems.Delete(itemId)
	err := call.Do()
	handleError(err, "error making playlist delete call")
}

func PlaylistsItemUpdate(itemId string, playlistId string, videoId string, position int64) *youtube.PlaylistItem {

	item := &youtube.PlaylistItem{
		Id: itemId,
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistId,
			Position:   position,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoId,
			},
		},
	}

	call := service.PlaylistItems.Update([]string{"snippet"}, item)
	resp, err := call.Do()
	handleError(err, "error making playlist update call")
	return resp
}

func PlaylistsItemInsert(playlistId string, videoId string, position int64) *youtube.PlaylistItem {
	item := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistId,
			Position:   position,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoId,
			},
		},
	}

	call := service.PlaylistItems.Insert([]string{"snippet"}, item)
	resp, err := call.Do()
	handleError(err, "error making playlist insert call")
	return resp
}

func PlaylistsItems(part string, playlistId string, pageToken string) *youtube.PlaylistItemListResponse {
	call := service.PlaylistItems.List([]string{part})
	call.MaxResults(50)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	if playlistId != "" {
		call = call.PlaylistId(playlistId)
	}
	response, err := call.Do()
	handleError(err, "")
	return response
}

func PlaylistsList(part string, channelId string, maxResults int64) *youtube.PlaylistListResponse {
	call := service.Playlists.List([]string{part})
	if channelId != "" {
		call = call.ChannelId(channelId)
	} else {
		call = call.Mine(true)
	}
	// if hl != "" {
	// 	call = call.Hl(hl)
	// }
	call = call.MaxResults(maxResults)
	// if onBehalfOfContentOwner != "" {
	// 	call = call.OnBehalfOfContentOwner(onBehalfOfContentOwner)
	// }
	// if pageToken != "" {
	// 	call = call.PageToken(pageToken)
	// }
	// if playlistId != "" {
	// 	call = call.Id(playlistId)
	// }
	response, err := call.Do()
	handleError(err, "")
	return response
}

func UpdateVideo(videoId string, title string, description string, privacy string, keywords string) string {

	// privacy := "unlisted"
	update := &youtube.Video{
		Id: videoId,
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: description,
			CategoryId:  "27",
		},
		Status: &youtube.VideoStatus{Embeddable: true, PrivacyStatus: privacy, SelfDeclaredMadeForKids: false, MadeForKids: false},
	}
	if strings.Trim(keywords, "") != "" {
		update.Snippet.Tags = strings.Split(keywords, ",")
	}
	call := service.Videos.Update([]string{"snippet", "status"}, update)
	response, err := call.Do()
	handleError(err, "")
	fmt.Printf("Update successful! Video ID: %v\n", response)
	return response.Id
}

func DeleteCaption(captionId string) {
	call := service.Captions.Delete(captionId)
	err := call.Do()
	handleError(err, "error deleting caption Id: "+captionId)
}

func ListCaption(videoId string) *youtube.CaptionListResponse {
	call := service.Captions.List([]string{"snippet"}, videoId)
	resp, err := call.Do()
	handleError(err, "error listing captions for video Id"+videoId)
	return resp
}

func UploadCaption(captionId string, videoId string, lang string, name string, captionFilePath string) string {
	upload := &youtube.Caption{
		Snippet: &youtube.CaptionSnippet{
			VideoId:  videoId,
			Language: lang,
			Name:     name,
		},
	}

	file, err := os.Open(captionFilePath)
	handleError(err, "can't open media file")
	defer file.Close()

	var response *youtube.Caption
	if len(strings.TrimSpace(captionId)) > 0 {
		upload.Id = captionId
		call := service.Captions.Update([]string{"snippet"}, upload)
		response, err = call.Media(file).Do()
	} else {
		call := service.Captions.Insert([]string{"snippet"}, upload)
		response, err = call.Media(file).Do()
	}

	handleError(err, "")
	fmt.Printf("Update successful! Caption ID: %v\n", response)
	return response.Id

}

func UploadCover(videoId string, filePath string) {

	call := service.Thumbnails.Set(videoId)
	file, err := os.Open(filePath)
	handleError(err, "can't open media file")
	defer file.Close()

	resp, err := call.Media(file).Do()
	handleError(err, "error updating thumbnail")
	println(resp.ServerResponse.Header)
}

func UploadVideo(title string, description string, category string, keywords string, filePath string) string {

	privacy := "unlisted"
	upload := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: description,
			CategoryId:  category,
		},
		Status: &youtube.VideoStatus{PrivacyStatus: privacy, SelfDeclaredMadeForKids: false},
	}
	if strings.Trim(keywords, "") != "" {
		upload.Snippet.Tags = strings.Split(keywords, ",")
	}
	call := service.Videos.Insert([]string{"snippet", "status"}, upload)
	file, err := os.Open(filePath)
	handleError(err, "can't open media file")
	defer file.Close()
	response, err := call.Media(file).Do()
	handleError(err, "")
	fmt.Printf("Upload successful! Video ID: https://www.youtube.com/watch?v=%v\n", response.Id)
	return response.Id
}

func DeleteVideo(ytVideoId string) {

	call := service.Videos.Delete(ytVideoId)
	err := call.Do()
	handleError(err, "")
	fmt.Printf("Delete successful! Video ID: %v\n", ytVideoId)
}

func main() {
	// ch, err := startWebServer()
	// code := <-ch
	// fmt.Println("code is: ", code)

	ChannelsListByUsername("snippet,contentDetails,statistics", "GoogleDevelopers")

	// method = flag.String("method", "list", "The API method to execute. (List is the only method that this sample currently supports.")

	part := "snippet"
	pageToken := ""

	// jstr, err := json.MarshalIndent(resp, "", "    ")
	// if err != nil {
	// 	log.Panic(err)
	// }
	// file, err := os.OpenFile("play.json", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0660)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// fmt.Fprint(file, string(jstr))

	resp := PlaylistsItems(part, "PLNFPO1g9Ma2cXa_9pCD85rzY2ISS4koGf", pageToken)
	for {
		for _, item := range resp.Items {
			fmt.Println(item.Id, item.Snippet.Title, item.Snippet.Description)
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
		resp = PlaylistsItems(part, "PLNFPO1g9Ma2cXa_9pCD85rzY2ISS4koGf", pageToken)
	}

}
