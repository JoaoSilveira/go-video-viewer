package main

import (
	"flag"
	"fmt"
	. "go-video-viewer/entities"
	"go-video-viewer/templates"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

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
