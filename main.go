package main

import (
	"flag"
	"fmt"
	inter "go-video-viewer/internals"
	"go-video-viewer/templates"
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	config, err := inter.LoadConfig(inter.ConfigFilePath)
	if err != nil {
		log.Fatalln("Failed to load config file.", err)
	}

	repo, err := inter.NewRepository(config)
	if err != nil {
		fmt.Printf("Failed to open database %v", err)
		return
	}
	defer repo.Close()

	jsonFile := flag.String("json-file", "", "json file path")
	flag.Parse()

	if *jsonFile != "" {
		log.Println("importing json file...")

		err := repo.ImportJsonFile(*jsonFile)
		if err != nil {
			log.Fatalln("Failed to read json file")
		}
	}

	http.HandleFunc("GET /next-video", func(w http.ResponseWriter, r *http.Request) {
		videos, err := repo.NextInQueue(1)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(videos) == 0 {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		templates.WatchVideo(videos[0]).Render(r.Context(), w)
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

		var status inter.VideoStatus
		switch statusParam {
		case "meh":
			status = inter.VideoWatched
		case "like":
			status = inter.VideoLiked
		case "fave":
			status = inter.VideoSaved
		default:
			{
				http.Error(w, "invalid value for \"status\"", http.StatusBadRequest)
				return
			}
		}

		videos, err := repo.NextInQueue(2)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(videos) == 0 {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		videos[0].Status = status
		if err = repo.Update(videos[0]); err != nil {
			http.Error(w, "failed to update video status", http.StatusInternalServerError)
			return
		}

		if len(videos) == 1 {
			http.Error(w, "no next video available", http.StatusNotFound)
			return
		}

		templates.WatchVideo(videos[1]).Render(r.Context(), w)
	})
	http.HandleFunc("GET /video-list", func(w http.ResponseWriter, r *http.Request) {
		list, err := repo.ListAllSaved()
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

		video, err := repo.FindById(int32(id))
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

		video, err := repo.FindById(int32(id))
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

		var status inter.VideoStatus
		switch statusParam {
		case "meh":
			status = inter.VideoWatched
		case "like":
			status = inter.VideoLiked
		case "fave":
			status = inter.VideoSaved
		default:
			{
				http.Error(w, "invalid value for \"status\"", http.StatusBadRequest)
				return
			}
		}

		video.Status = status
		if repo.Update(*video) != nil {
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

		video, err := repo.FindById(int32(id))
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
