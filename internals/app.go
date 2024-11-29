package internals

import (
	"go-video-viewer/cmd_args"
	"log"
	"os"
	"path"
)

type App struct {
	Config Config
	Repo   VideoRepository
}

func NewApp() App {
	config, err := LoadConfig(ConfigFilePath)
	if err != nil {
		log.Fatalln("Failed to load config file.", err)
	}

	repo, err := NewRepository(config)
	if err != nil {
		log.Fatalln("Failed to initialize repository", err)
	}

	return App{Config: config, Repo: repo}
}

func (app App) Close() {
	app.Repo.Close()
}

func (app App) Init() {
	args := cmd_args.ReadArgs()
	if !args.HasJsonFile() {
		return
	}

	log.Println("importing json file...")

	err := app.Repo.ImportJsonFile(args.JsonFile)
	if err != nil {
		log.Fatalln("Failed to read json file", err)
	}
}

func (app App) UpdateVideo(video Video) error {
	err := app.Repo.Update(video)
	if err != nil {
		return err
	}

	if !video.Status.PersistFile() {
		return os.Truncate(app.VideoPath(video), 0)
	}

	return nil
}

func (app App) UpdateRepoFromFolder() error {
	entries, err := app.Repo.ListDirVideos()
	if err != nil {
		return err
	}

	return app.Repo.ImportFsEntries(entries)
}

func (app App) VideoPath(video Video) string {
	return path.Join(app.Config.VideoFolder, video.Filename)
}
