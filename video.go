package entities

import "time"

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
	if video.Saved {
		return VideoSaved
	} else if video.Favorited {
		return VideoLiked
	} else {
		return VideoWatched
	}
}
