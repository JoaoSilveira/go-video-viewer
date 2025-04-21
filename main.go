package main

import (
	"context"
	"encoding/json"
	"fmt"
	inter "go-video-viewer/internals"
	"go-video-viewer/templates"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var app inter.App

func renderError(
	ctx context.Context,
	w http.ResponseWriter,
	msg string,
	err error,
	status int,
) {
	w.WriteHeader(status)
	templates.ErrorPage(status, msg, err).Render(ctx, w)
}

func renderNoNextVideo(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	templates.NoNextVideoPage().Render(ctx, w)
}

func renderNotFound(ctx context.Context, w http.ResponseWriter, id int) {
	w.WriteHeader(http.StatusNotFound)
	templates.VideoNotFoundPage(id).Render(ctx, w)
}

func handleGetHome(w http.ResponseWriter, r *http.Request) {
	stats, err := app.Repo.QueryStats()
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to query the stats from the database",
			err,
			http.StatusInternalServerError,
		)
		log.Println("QueryStats() failed", err)
		return
	}

	templates.HomePage(stats).Render(r.Context(), w)
}

func handleGetNextVideo(w http.ResponseWriter, r *http.Request) {
	videos, err := app.Repo.NextInQueue(1)
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to fetch next video in queue",
			err,
			http.StatusInternalServerError,
		)
		log.Println("NextInQueue() failed", err)
		return
	}

	if len(videos) == 0 {
		renderNoNextVideo(r.Context(), w)
		return
	}

	templates.WatchVideo(videos[0]).Render(r.Context(), w)
}

func handlePostNextVideo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to parse form data",
			err,
			http.StatusBadRequest,
		)
		log.Println("ParseForm() failed", err)
		return
	}

	newStatus, err := inter.StatusFromStringValue(r.FormValue("status"))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"invalid form field",
			err,
			http.StatusBadRequest,
		)
		log.Print(err)
		return
	}

	videos, err := app.Repo.NextInQueue(2)
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to fetch next video in queue",
			err,
			http.StatusInternalServerError,
		)
		log.Println("NextInQueue failed:", err)
		return
	}

	if len(videos) == 0 {
		renderNoNextVideo(r.Context(), w)
		return
	}

	videos[0].Status = newStatus
	videos[0].Nickname = inter.NullString{String: r.FormValue("nickname"), Valid: true}
	videos[0].Tags = strings.Split(r.FormValue("tags"), ",")
	err = app.UpdateVideo(videos[0])
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to update video status",
			err,
			http.StatusInternalServerError,
		)
		log.Println("UpdateVideo failed:", err)
		return
	}

	if len(videos) == 1 {
		renderNoNextVideo(r.Context(), w)
		return
	}

	templates.WatchVideo(videos[1]).Render(r.Context(), w)
}

func handleGetVideoList(w http.ResponseWriter, r *http.Request) {
	list, err := app.Repo.ListAllSaved()
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to list saved videos",
			err,
			http.StatusInternalServerError,
		)
		log.Println("ListAllSaved failed:", err)
		return
	}

	templates.ListSavedPage(list).Render(r.Context(), w)
}

func handleGetWatch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		renderError(r.Context(), w, "invalid id in url path", err, http.StatusBadRequest)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to read video from the database",
			err,
			http.StatusInternalServerError,
		)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		renderNotFound(r.Context(), w, id)
		return
	}

	templates.WatchVideo(*video).Render(r.Context(), w)
}

func handlePostWatch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"invalid id in url path",
			err,
			http.StatusBadRequest,
		)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to read video from the database",
			err,
			http.StatusInternalServerError,
		)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		renderNotFound(r.Context(), w, id)
		return
	}

	if r.ParseForm() != nil {
		renderError(
			r.Context(),
			w,
			"failed to parse form data",
			err,
			http.StatusBadRequest,
		)
		return
	}

	newStatus, err := inter.StatusFromStringValue(r.FormValue("status"))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"invalid form field",
			err,
			http.StatusBadRequest,
		)
		log.Println(err)
		return
	}

	video.Status = newStatus
	video.Nickname = inter.NullString{String: r.FormValue("nickname"), Valid: true}
	video.Tags = strings.Split(r.FormValue("tags"), ",")
	err = app.UpdateVideo(*video)
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to update video status",
			err,
			http.StatusInternalServerError,
		)
		log.Println("UpdateVideo failed:", err)
		return
	}

	templates.WatchVideo(*video).Render(r.Context(), w)
}

func handleGetVideo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		renderError(r.Context(), w, "invalid id in url path", err, http.StatusBadRequest)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to read video from the database",
			err,
			http.StatusInternalServerError,
		)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		renderNotFound(r.Context(), w, id)
		return
	}

	http.ServeFile(w, r, app.VideoPath(*video))
}

func handlePostUpdate(w http.ResponseWriter, r *http.Request) {
	err := app.UpdateRepoFromFolder()
	if err != nil {
		renderError(
			r.Context(),
			w,
			"failed to update the database",
			err,
			http.StatusInternalServerError,
		)
		log.Println("UpdateRepoFromFolder() failed:", err)
	}

	http.Redirect(w, r, "/next-video", http.StatusMovedPermanently)
}

func handleApiGetLastUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	date, err := app.LastFolderUpdate()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("LastFolderUpdate failed", err)
		return
	}

	response := inter.LastUpdateResponse{
		LastUpdate: date,
	}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to encode json", err)
		return
	}
}

func handleApiGetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	stats, err := app.Repo.QueryStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("QueryStats() failed", err)
		return
	}

	response := inter.VideoStatsResponse{
		Stats: stats,
	}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to encode json", err)
		return
	}
}

func handleApiGetNextVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	videos, err := app.Repo.NextInQueue(2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("NextInQueue() failed", err)
		return
	}

	if len(videos) < 1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := inter.VideoResponse{
		Video: videos[0],
		Next:  nil,
	}

	if len(videos) > 1 {
		response.Next = &videos[1]
	}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to encode json", err)
		return
	}
}

func handleApiListVideos(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	videos, err := app.Repo.ListAllSaved()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("NextInQueue() failed", err)
		return
	}

	reponse := inter.VideoListResponse{
		Videos: videos,
	}

	if err = json.NewEncoder(w).Encode(reponse); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to encode json", err)
		return
	}
}

func handleApiGetVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		renderError(r.Context(), w, "invalid id in url path", err, http.StatusBadRequest)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := inter.VideoResponse{
		Video: *video,
		Next:  nil,
	}

	video, err = app.Repo.NextSavedById(int32(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("NextSavedById '%v' failed: %v", id, err)
		return
	}

	response.Next = video

	if err = json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to encode json", err)
		return
	}
}

func handleApiServeVideo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, app.VideoPath(*video))
}

func handleApiScanVideos(w http.ResponseWriter, r *http.Request) {
	err := app.UpdateRepoFromFolder()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("UpdateRepoFromFolder() failed:", err)
		return
	}

	handleApiGetLastUpdate(w, r)
}

func handleApiUpdateVideo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid id:", r.PathValue("id"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("failed to read request body")
		return
	}

	var payload inter.VideoUpdatePayload
	if err = json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		log.Println("invalid request body")
		return
	}

	video, err := app.Repo.FindById(int32(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("FindById '%v' failed: %v", id, err)
		return
	}

	if video == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	video.Nickname = payload.Nickname
	video.Status = payload.Status
	video.Tags = payload.Tags

	err = app.UpdateVideo(*video)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("UpdateVideo failed:", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Booting up!")

	app = inter.NewApp()
	defer app.Close()

	app.Init()
	log.Println("Application initialized")

	// http.HandleFunc("GET /", handleGetHome)
	// http.HandleFunc("GET /next-video", handleGetNextVideo)
	// http.HandleFunc("POST /next-video", handlePostNextVideo)
	// http.HandleFunc("GET /video-list", handleGetVideoList)
	// http.HandleFunc("GET /watch/{id}", handleGetWatch)
	// http.HandleFunc("POST /watch/{id}", handlePostWatch)
	// http.HandleFunc("GET /video/{id}", handleGetVideo)
	// http.HandleFunc("POST /update", handlePostUpdate)

	http.HandleFunc("GET /api/last-update", handleApiGetLastUpdate)
	http.HandleFunc("GET /api/video/stats", handleApiGetStats)
	http.HandleFunc("GET /api/video/next", handleApiGetNextVideo)
	http.HandleFunc("GET /api/video/{id}", handleApiGetVideo)
	http.HandleFunc("GET /api/video/{id}/serve", handleApiServeVideo)
	http.HandleFunc("GET /api/video/list", handleApiListVideos)
	http.HandleFunc("POST /api/video/scan", handleApiScanVideos)
	http.HandleFunc("POST /api/video/{id}", handleApiUpdateVideo)

	fileServer := http.FileServer(http.Dir("public"))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join("public", r.URL.Path)
		if _, err := os.Stat(filePath); err == nil || !os.IsNotExist(err) {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, "public/index.html")
	}))

	log.Printf("Listening on %v:%v\n", app.Config.Address, app.Config.Port)
	err := http.ListenAndServe(
		fmt.Sprintf("%v:%v", app.Config.Address, app.Config.Port),
		nil,
	)

	if err != nil {
		log.Println("Cannot listen: ", err)
	}
}
