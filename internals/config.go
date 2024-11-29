package internals

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	Database    string `ini:"database"`
	VideoFolder string `ini:"video_folder"`
	Address     string `ini:"address"`
	Port        string `ini:"port"`
}

func LoadConfig() (Config, error) {
	cfg, err := ini.Load(getIniPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, tryCreateIniFile()
		}

		return Config{}, err
	}

	pathConfig := Config{
		Database:    "",
		VideoFolder: "",
		Address:     "127.0.0.1",
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

func getIniPath() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("failed to get executable path", err)
	}

	ext := filepath.Ext(exePath)
	return strings.TrimSuffix(exePath, ext) + ".ini"
}

func tryCreateIniFile() error {
	iniPath := getIniPath()
	f, err := os.Create(iniPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprint(f, "database=\nvideo_folder=")
	if err != nil {
		return err
	}

	return errors.New("The .ini file was absent, a new one was created, please fill it up. Path: " + iniPath)
}
