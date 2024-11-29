package internals

import (
	"errors"
	"strconv"

	"gopkg.in/ini.v1"
)

const ConfigFilePath string = "./go-video-viewer.ini"

type Config struct {
	Database    string `ini:"database"`
	VideoFolder string `ini:"video_folder"`
	Port        string `ini:"port"`
}

func LoadConfig(path string) (Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return Config{}, err
	}

	pathConfig := Config{
		Database:    "",
		VideoFolder: "",
		Port:        "3000",
	}

	err = cfg.MapTo(&pathConfig)
	if err != nil {
		return Config{}, err
	}

	err = pathConfig.validate()
	if err != nil {
		return Config{}, err
	}

	return pathConfig, nil
}

func (cfg Config) validate() error {
	if cfg.Database == "" {
		return errors.New("\"database\" config was not set. Should be the path of the database file")
	}

	if cfg.VideoFolder == "" {
		return errors.New("\"video_folder\" config was not set. Should be the path of the folder that contains the videos")
	}

	if v, err := strconv.Atoi(cfg.Port); err != nil || v <= 0 || v > 65535 {
		return errors.New("\"port\" config was not properly set. Should be a number in the range [1, 65535]")
	}

	return nil
}
