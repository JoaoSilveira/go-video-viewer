package internals

import (
	"time"
	"strconv"
	"fmt"
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
	Saved     bool      `json:"saved"`
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

	switch val {
	case VideoUnwatched:
		return inter.VideoUnwatched, nil
	case VideoWatched:
		return inter.VideoWatched, nil
	case VideoLiked:
		return inter.VideoLiked, nil
	case VideoSaved:
		return inter.VideoSaved, nil
	default:
		return 0, fmt.Errorf("invalid video status value \"%v\"", val)
	}
}

func (status VideoStatus) PersistFile() bool {
	return status == VideoSaved
}