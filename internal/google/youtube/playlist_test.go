package ytapi

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/youtube/v3"
)

func dummyItems(items []string) []YtPlItem {
	ret := []YtPlItem{}
	for _, e := range items {
		ret = append(ret, YtPlItem{VideoId: e})
	}

	return ret
}

/*
 */
func loadJson(path string) []YtPlItem {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Panic(err)
	}
	var items []youtube.PlaylistItem
	err = json.Unmarshal(contentBytes, &items)
	if err != nil {
		log.Panic(err)
	}

	var ret []YtPlItem
	for _, e := range items {
		ye := ToYtPlItem(&e)
		ret = append(ret, *ye)
	}
	return ret
}

func Test_rebuild(t *testing.T) {
	items := loadJson("list.json")
	fmt.Println(items)
	// rebuild(strings.Split("adcebfjihg", ""), items)
	del, ins, upd := RebuildPlaylist("abc", strings.Split("h63qnV0B0jc,xz0L3KknMU0,Yq8t3sFao4s,SeZl4r8F6hM,s2BSE3LvLpc", ","), items)
	fmt.Println(del, ins, upd)
	// commit(items, del, ins, upd)
	// rebuild(strings.Split("hello world", ""), strings.Split("aloha", ""))
}
