package internals

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type VideoStatus int32

const (
	VideoUnwatched VideoStatus = iota + 1
	VideoWatched
	VideoLiked
	VideoSaved
)

type Video struct {
	Id        int32
	Filename  string
	CreatedAt time.Time
	Status    VideoStatus
}

type VideoJsonEntry struct {
	Name      string    `json:"name"`
	Date      time.Time `json:"date"`
	Favorited bool      `json:"favorited"`
	Saved     bool      `json:"save"`
}

type VideoJsonFile struct {
	Watched []VideoJsonEntry `json:"watched"`
	ToWatch []VideoJsonEntry `json:"toWatch"`
	Current VideoJsonEntry   `json:"current"`
}

type VideoFsEntry struct {
	Filename         string
	LastModifiedTime time.Time
	IsTruncated      bool
}

func StatusFromWatchedEntry(entry VideoJsonEntry) VideoStatus {
	if entry.Saved {
		return VideoSaved
	} else if entry.Favorited {
		return VideoLiked
	} else {
		return VideoWatched
	}
}

func StatusFromStringValue(value string) (VideoStatus, error) {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid value for video status \"%v\"", value)
	}

	switch VideoStatus(val) {
	case VideoUnwatched:
		return VideoUnwatched, nil
	case VideoWatched:
		return VideoWatched, nil
	case VideoLiked:
		return VideoLiked, nil
	case VideoSaved:
		return VideoSaved, nil
	default:
		return 0, fmt.Errorf("invalid video status value \"%v\"", val)
	}
}

func (status VideoStatus) PersistFile() bool {
	return status == VideoSaved
}

func hasVideoExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	return slices.Contains(VideoExtensions, ext)
}

// list taken from https://gist.github.com/aaomidi/0a3b5c9bd563c9e012518b495410dc0e
var VideoExtensions = []string{
	"webm",
	"mkv",
	"flv",
	"vob",
	"ogv",
	"ogg",
	"rrc",
	"gifv",
	"mng",
	"mov",
	"avi",
	"qt",
	"wmv",
	"yuv",
	"rm",
	"asf",
	"amv",
	"mp4",
	"m4p",
	"m4v",
	"mpg",
	"mp2",
	"mpeg",
	"mpe",
	"mpv",
	"m4v",
	"svi",
	"3gp",
	"3g2",
	"mxf",
	"roq",
	"nsv",
	"flv",
	"f4v",
	"f4p",
	"f4a",
	"f4b",
	"mod",
}
