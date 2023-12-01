package ytapi

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/youtube/v3"
)

type YtPlItem struct {
	ItemId     string
	VideoId    string
	PlaylistId string
	Title      string
}

type YtPlist []YtPlItem

func (playlist *YtPlist) VideoIds() []string {
	var ret []string
	for _, e := range *playlist {
		ret = append(ret, e.VideoId)
	}
	return ret
}

func (playlist YtPlist) IdxByVideoId(videoId string, start int) int {
	for i := 0; i < len(playlist); i++ {
		if videoId == playlist[i].VideoId {
			return i
		}
	}
	return -1
}

// return the common subset of 2 slices, with their respective order
// (A^B, A-B, B^A, B-A),
// A^B and B^A have the same elements but different order
func subsets(xs []string, ys []string) ([]string, []string, []string, []string) {
	_split := func(to []string, from []string) ([]string, []string) {
		var intersect []string
		var xtra []string
		dupMap := make(map[int]bool)
		for _, x := range to {
			common := false
			for j, y := range from {
				if x == y && !dupMap[j] {
					dupMap[j] = true
					common = true
					intersect = append(intersect, x)
					break
				}
			}
			if !common {
				xtra = append(xtra, x)
			}
		}
		return intersect, xtra
	}
	a, b := _split(xs, ys)
	c, d := _split(ys, xs)
	return a, b, c, d
}

func assert(cond bool, msg string) {
	if !cond {
		log.Panic(msg)
	}
}

type MoveOper struct {
	title          string
	VideoId        string
	PlaylistId     string
	fromPosition   int
	toPosition     int
	playlistItemId string
}

type DeleteOper struct {
	title          string
	PlaylistItemId string
}

type InsertOper struct {
	title      string
	VideoId    string
	PlaylistId string
	position   int
}

func ToYtPlItem(e *youtube.PlaylistItem) *YtPlItem {
	return &YtPlItem{ItemId: e.Id, VideoId: e.Snippet.ResourceId.VideoId, PlaylistId: e.Snippet.PlaylistId, Title: e.Snippet.Title}
}

func _insertAt[T comparable](slice []T, elem T, i int) []T {
	return append(slice[:i], append([]T{elem}, slice[i:]...)...)
}

func _deleteAt[T comparable](slice []T, i int) []T {
	return append(slice[:i], slice[i+1:]...)
}

func _move[T comparable](slice []T, to, from int) []T {
	elem := slice[from]
	slice = _insertAt(slice, elem, to)
	if from > to {
		from += 1
	}
	slice = _deleteAt(slice, from)
	return slice
}

// hola -> leho
func RebuildPlaylist(playlistId string, to []string, currentItems YtPlist) ([]DeleteOper, []InsertOper, []MoveOper) {
	from := currentItems.VideoIds()
	tgt, toAdd, src, toDel := subsets(to, from)
	assert(len(tgt) == len(src), fmt.Sprintf("subset length mismatch: %v: %v", tgt, src))
	fmt.Println(src, "-->", tgt)
	fmt.Println(toAdd, ",", toDel)

	var deleteOpers []DeleteOper
	var insertOpers []InsertOper
	var moveOpers []MoveOper

	operMapping := make(map[string]int)

	_find := func(slice []string, elem string, start int) int {
		for i, x := range slice {
			if i >= start && x == elem {
				return i
			}
		}
		// log.Panic("element "+elem+" not found in: ", slice)
		return -1
	}

	_insertOper := func(slice []string, elem string, i int) []string {
		// opers = append(opers, fmt.Sprintf(`" ++'%s'@%d"`, elem, i))
		title := ""
		idx := currentItems.IdxByVideoId(elem, 0)
		if idx >= 0 {
			title = currentItems[idx].Title
		}
		insertOpers = append(insertOpers, InsertOper{title: title, position: i, VideoId: elem, PlaylistId: playlistId})
		return _insertAt(slice, elem, i)
	}

	_deleteOper := func(slice []string, i int) []string {
		key := slice[i]
		_idx := currentItems.IdxByVideoId(key, operMapping[key])
		if i < 0 {
			log.Panic("can't delete item ", _idx)
		}
		deleteOpers = append(deleteOpers, DeleteOper{PlaylistItemId: currentItems[_idx].ItemId, title: currentItems[_idx].Title})
		operMapping[key] = _idx + 1
		return _deleteAt(slice, i)
	}

	_moveOper := func(slice []string, to, from int) []string {
		elem := slice[from]
		title := ""
		idx := currentItems.IdxByVideoId(elem, 0)
		if idx >= 0 {
			title = currentItems[idx].Title
		}
		moveOpers = append(moveOpers, MoveOper{title: title, fromPosition: from, toPosition: to, VideoId: elem, PlaylistId: playlistId, playlistItemId: ""})
		return _move(slice, to, from)
	}
	// first delete
	for _, e := range toDel {
		j := _find(from, e, 0)
		from = _deleteOper(from, j)
	}
	assert(len(src) == len(from), fmt.Sprintf("src and from should be same length after deletion, %v : %v", src, from))

	// insert
	for i := range toAdd {
		last := len(toAdd) - 1 - i
		from = _insertOper(from, toAdd[last], 0)
	}
	assert(len(from) == len(to), "src and from should match after deletion")

	// reorder
	searchPoint := make(map[string]int)
	for i := 0; i < len(from); i++ {
		j := _find(from, to[i], searchPoint[to[i]])
		if i != j {
			from = _moveOper(from, i, j)
		}
		searchPoint[to[i]] = i + 1
	}
	assert(strings.Join(from, "") == strings.Join(to, ""), fmt.Sprintf("from and to should be identical after reorder: %v : %v", from, to))
	fmt.Println("original", from)
	// filter moveOpers
	fmt.Println("delete", deleteOpers)
	fmt.Println("insert", insertOpers)
	fmt.Println("reorder", moveOpers)
	return deleteOpers, insertOpers, moveOpers
}

func CommitPlaylist(currentItems YtPlist, deletes []DeleteOper, inserts []InsertOper, reorders []MoveOper) {
	targetItems := currentItems
	for _, op := range deletes {
		PlaylistsItemDelete(op.PlaylistItemId)
	}
	for _, op := range inserts {
		item := PlaylistsItemInsert(op.PlaylistId, op.VideoId, int64(op.position))
		targetItems = _insertAt(targetItems, *ToYtPlItem(item), op.position)
	}

	mapping := make(map[string]int)
	for _, op := range reorders {
		// fmt.Println(item)
		targetItems = _move(targetItems, op.toPosition, op.fromPosition)
		// inserted item, find from list
		idx := targetItems.IdxByVideoId(op.VideoId, mapping[op.VideoId])
		mapping[op.VideoId] = idx + 1
		itemId := targetItems[idx].ItemId
		PlaylistsItemUpdate(itemId, op.PlaylistId, op.VideoId, int64(op.toPosition))
	}

}
