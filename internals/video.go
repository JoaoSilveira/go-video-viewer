package internals

import (
	"database/sql"
	"encoding/json"
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

type NullString sql.NullString

func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(ns.String)
}

func (ns *NullString) UnmarshalJSON(data []byte) error {
	if len(data) == 4 {
		isNull := data[0] == 'n' &&
			data[1] == 'u' &&
			data[2] == 'l' &&
			data[3] == 'l'

		if isNull {
			*ns = NullString{String: "", Valid: false}
			return nil
		}
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	*ns = NullString{String: str, Valid: true}
	return nil
}

func (ns *NullString) Scan(value any) error {
	var str sql.NullString
	if err := str.Scan(value); err != nil {
		return err
	}

	*ns = NullString{str.String, str.Valid}
	return nil
}

type Video struct {
	Id        int32       `json:"id"`
	Filename  string      `json:"filename"`
	Nickname  NullString  `json:"nickname"`
	Tags      []string    `json:"tags"`
	CreatedAt time.Time   `json:"created_at"`
	Status    VideoStatus `json:"status"`
}

type LastUpdateResponse struct {
	LastUpdate *time.Time `json:"last_update"`
}

type VideoUpdatePayload struct {
	Nickname NullString  `json:"nickname"`
	Tags     []string    `json:"tags"`
	Status   VideoStatus `json:"status"`
}

type VideoResponse struct {
	Video Video  `json:"video"`
	Next  *Video `json:"next"`
}

type VideoListResponse struct {
	Videos []Video `json:"videos"`
}

type VideoStatsResponse struct {
	Stats VideoStats `json:"stats"`
}

type VideoJsonEntry struct {
	Name      string    `json:"name"`
	Nickname  string    `json:"nickname"`
	Tags      []string  `json:"tags"`
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

type VideoStats struct {
	Unwatched int `json:"unwatched"`
	Watched   int `json:"watched"`
	Liked     int `json:"liked"`
	Saved     int `json:"saved"`
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
	".webm",
	".mkv",
	".flv",
	".vob",
	".ogv",
	".ogg",
	".rrc",
	".gifv",
	".mng",
	".mov",
	".avi",
	".qt",
	".wmv",
	".yuv",
	".rm",
	".asf",
	".amv",
	".mp4",
	".m4p",
	".m4v",
	".mpg",
	".mp2",
	".mpeg",
	".mpe",
	".mpv",
	".m4v",
	".svi",
	".3gp",
	".3g2",
	".mxf",
	".roq",
	".nsv",
	".flv",
	".f4v",
	".f4p",
	".f4a",
	".f4b",
	".mod",
}
