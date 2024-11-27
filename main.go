package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	. "go-video-viewer/entities"
	"go-video-viewer/templates"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/ini.v1"
)

const ConfigFilePath string = "./go-video-viewer.ini"

type PathConfig struct {
	Database    string
	VideoFolder string
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

func ReadVideoJsonFile(path string) (*VideoJsonFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload VideoJsonFile
	err = json.Unmarshal(content, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

type VideoFsEntry struct {
	Filename         string
	LastModifiedTime time.Time
	IsTruncated      bool
}

func ListDirVideos(path string) ([]VideoFsEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]VideoFsEntry, len(entries))
	for i := 0; i < len(entries); i++ {
		if entries[i].IsDir() {
			continue
		}

		info, err := entries[i].Info()
		if err != nil {
			return nil, err
		}

		files = append(files, VideoFsEntry{
			Filename:         entries[i].Name(),
			LastModifiedTime: info.ModTime(),
			IsTruncated:      info.Size() <= 0,
		})
	}

	return files, nil
}

func LoadConfig() (PathConfig, error) {
	cfg, err := ini.Load(ConfigFilePath)
	if err != nil {
		return PathConfig{}, err
	}

	pathConfig := PathConfig{}
	err = cfg.MapTo(pathConfig)
	if err != nil {
		return PathConfig{}, err
	}

	return pathConfig, nil
}

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		create table if not exists videos (
			id integer primary key,
			filename text not null unique,
			created_at datetime not null,
			status integer
		)
	`)

	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func ImportJson(db *sql.DB, jsonVideo *VideoJsonFile) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		insert into videos
			(filename, created_at, status)
		values
			(?, ?, ?)
		on conflict (filename) do nothing
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, video := range jsonVideo.Watched {
		var status VideoStatus

		if video.Saved {
			status = VideoSaved
		} else if video.Favorited {
			status = VideoLiked
		} else {
			status = VideoWatched
		}

		_, err = stmt.Exec(video.Name, video.Date, status)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = stmt.Exec(jsonVideo.Current.Name, jsonVideo.Current.Date, VideoUnwatched)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, video := range jsonVideo.ToWatch {
		_, err = stmt.Exec(video.Name, video.Date, VideoUnwatched)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func ListAllSaved(db *sql.DB) ([]Video, error) {
	rows, err := db.Query(
		`
		select
			id,
			filename,
			created_at,
			status
		from
			videos
		where
			status = ?
		`,
		VideoSaved,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		video, err := ReadVideoFromRow(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func NextVideoInQueue(db *sql.DB) (*Video, error) {
	rows, err := db.Query(
		`
		select
			id,
			filename,
			created_at,
			status
		from
			videos
		where
			status = ?
		order by
			created_at
		limit 1
		`,
		VideoUnwatched,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	video, err := ReadVideoFromRow(rows)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func FindVideoById(db *sql.DB, id int32) (*Video, error) {
	rows, err := db.Query(
		`
		select
			id,
			filename,
			created_at,
			status
		from
			videos
		where
			id = ?
		`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	video, err := ReadVideoFromRow(rows)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func ReadVideoFromRow(rows *sql.Rows) (Video, error) {
	var video Video
	err := rows.Scan(
		&video.Id,
		&video.Filename,
		&video.CreatedAt,
		&video.Status,
	)
	if err != nil {
		return Video{}, err
	}

	return video, nil
}

func UpdateVideo(db *sql.DB, video Video) error {
	_, err := db.Exec(
		`
		update videos set
			status = ?
		where
			id = ?
		`,
		video.Status,
		video.Id,
	)

	return err
}

func main() {
	db, err := OpenDatabase("./database.db")
	if err != nil {
		fmt.Printf("Failed to open database %v", err)
		return
	}
	defer db.Close()

	jsonFile := flag.String("json-file", "attempt.json", "json file path")
	flag.Parse()

	if *jsonFile != "" {
		fmt.Println("Importing json...")

		jsonFile, err := ReadVideoJsonFile(*jsonFile)
		if err != nil {
			fmt.Println("Failed to read json file")
			return
		}

		if ImportJson(db, jsonFile) != nil {
			fmt.Println("Failed to import json")
			return
		}
	}

	http.HandleFunc("GET /next-video", func(w http.ResponseWriter, r *http.Request) {
		video, err := NextVideoInQueue(db)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if video == nil {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		templates.WatchVideo(*video).Render(r.Context(), w)
	})
	http.HandleFunc("POST /next-video", func(w http.ResponseWriter, r *http.Request) {
		if r.ParseForm() != nil {
			http.Error(w, "malformed form request data", http.StatusBadRequest)
			return
		}

		statusParam := r.FormValue("status")
		if statusParam == "" {
			http.Error(w, "bad \"status\" value", http.StatusBadRequest)
			return
		}

		var status VideoStatus
		switch statusParam {
		case "meh":
			status = VideoWatched
		case "like":
			status = VideoLiked
		case "fave":
			status = VideoSaved
		default:
			{
				http.Error(w, "invalid value for \"status\"", http.StatusBadRequest)
				return
			}
		}

		video, err := NextVideoInQueue(db)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if video == nil {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		video.Status = status
		if UpdateVideo(db, *video) != nil {
			http.Error(w, "failed to update video status", http.StatusInternalServerError)
			return
		}

		video, err = NextVideoInQueue(db)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if video == nil {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		templates.WatchVideo(*video).Render(r.Context(), w)
	})
	http.HandleFunc("GET /video-list", func(w http.ResponseWriter, r *http.Request) {
		list, err := ListAllSaved(db)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		templates.ListSavedPage(list).Render(r.Context(), w)
	})
	http.HandleFunc("GET /watch/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, "FAILED TO PARSE ID", http.StatusBadRequest)
			return
		}

		video, err := FindVideoById(db, int32(id))
		if err != nil {
			http.Error(
				w,
				"FAILED TO READ VIDEO ENTITY",
				http.StatusInternalServerError,
			)
			return
		}

		if video == nil {
			http.Error(w, "ENTITY NOT FOUND", http.StatusNotFound)
			return
		}

		templates.WatchVideo(*video).Render(r.Context(), w)
	})
	http.HandleFunc("POST /watch/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, "FAILED TO PARSE ID", http.StatusBadRequest)
			return
		}

		video, err := FindVideoById(db, int32(id))
		if err != nil {
			http.Error(
				w,
				"FAILED TO READ VIDEO ENTITY",
				http.StatusInternalServerError,
			)
			return
		}

		if video == nil {
			http.Error(w, "ENTITY NOT FOUND", http.StatusNotFound)
			return
		}
		if r.ParseForm() != nil {
			http.Error(w, "malformed form request data", http.StatusBadRequest)
			return
		}

		statusParam := r.FormValue("status")
		if statusParam == "" {
			http.Error(w, "bad \"status\" value", http.StatusBadRequest)
			return
		}

		var status VideoStatus
		switch statusParam {
		case "meh":
			status = VideoWatched
		case "like":
			status = VideoLiked
		case "fave":
			status = VideoSaved
		default:
			{
				http.Error(w, "invalid value for \"status\"", http.StatusBadRequest)
				return
			}
		}

		video.Status = status
		if UpdateVideo(db, *video) != nil {
			http.Error(w, "failed to update video status", http.StatusInternalServerError)
			return
		}

		templates.WatchVideo(*video).Render(r.Context(), w)
	})

	http.HandleFunc("GET /video/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			http.Error(w, "FAILED TO PARSE ID", http.StatusBadRequest)
			return
		}

		video, err := FindVideoById(db, int32(id))
		if err != nil {
			http.Error(
				w,
				"FAILED TO READ VIDEO ENTITY",
				http.StatusInternalServerError,
			)
			return
		}

		if video == nil {
			http.Error(w, "ENTITY NOT FOUND", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, video.Filename, http.StatusMovedPermanently)
		// TODO: http.ServeFile(w, r, video.Filename)
	})

	fmt.Println("Listening on :3000")
	http.ListenAndServe("127.0.0.1:3000", nil)
}
