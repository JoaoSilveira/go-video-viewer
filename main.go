package main

import (
	"encoding/json"
	"fmt"
	inter "go-video-viewer/internals"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var app inter.App

func filterEmptyStrings(slice []string) []string {
	var result []string

	for _, s := range slice {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}

	return result
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

	payload.Tags = filterEmptyStrings(payload.Tags)
	log.Print(payload)

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

func publicFolder() (string, error) {
	exec, err := os.Executable()
	if err != nil {
		return "", err
	}

	execDir := filepath.Dir(exec)
	return filepath.Join(execDir, "..", "public"), nil
}

func handleServeFile(relativePath string) http.HandlerFunc {
	folder, _ := publicFolder()
	filepath := path.Join(folder, relativePath)
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath)
	}
}

func main() {
	_, err := publicFolder()
	if err != nil {
		log.Fatalln("Could not resolve index file path", err)
		return
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Booting up!")

	app = inter.NewApp()
	defer app.Close()

	app.Init()
	log.Println("Application initialized")

	http.HandleFunc("GET /", handleServeFile("index.html"))
	http.HandleFunc("GET /index.js", handleServeFile("index.js"))
	http.HandleFunc("GET /api/last-update", handleApiGetLastUpdate)
	http.HandleFunc("GET /api/video/stats", handleApiGetStats)
	http.HandleFunc("GET /api/video/next", handleApiGetNextVideo)
	http.HandleFunc("GET /api/video/{id}", handleApiGetVideo)
	http.HandleFunc("GET /api/video/{id}/serve", handleApiServeVideo)
	http.HandleFunc("GET /api/video/list", handleApiListVideos)
	http.HandleFunc("POST /api/video/scan", handleApiScanVideos)
	http.HandleFunc("POST /api/video/{id}", handleApiUpdateVideo)

	log.Printf("Listening on %v:%v\n", app.Config.Address, app.Config.Port)
	err = http.ListenAndServe(
		fmt.Sprintf("%v:%v", app.Config.Address, app.Config.Port),
		nil,
	)

	if err != nil {
		log.Println("Cannot listen: ", err)
	}
}
