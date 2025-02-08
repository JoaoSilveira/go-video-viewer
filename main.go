package main

import (
	"context"
	"fmt"
	inter "go-video-viewer/internals"
	"go-video-viewer/templates"
	"log"
	"net/http"
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
	video.Nickname = r.FormValue("nickname")
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

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Booting up!")

	app = inter.NewApp()
	defer app.Close()

	app.Init()
	log.Println("Application initialized")

	http.HandleFunc("GET /", handleGetHome)
	http.HandleFunc("GET /next-video", handleGetNextVideo)
	http.HandleFunc("POST /next-video", handlePostNextVideo)
	http.HandleFunc("GET /video-list", handleGetVideoList)
	http.HandleFunc("GET /watch/{id}", handleGetWatch)
	http.HandleFunc("POST /watch/{id}", handlePostWatch)
	http.HandleFunc("GET /video/{id}", handleGetVideo)
	http.HandleFunc("POST /update", handlePostUpdate)

	log.Printf("Listening on %v:%v\n", app.Config.Address, app.Config.Port)
	http.ListenAndServe(fmt.Sprintf("%v:%v", app.Config.Address, app.Config.Port), nil)
}
