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
