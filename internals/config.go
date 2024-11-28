package internals

import "gopkg.in/ini.v1"

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

	pathConfig := Config{}
	err = cfg.MapTo(pathConfig)
	if err != nil {
		return Config{}, err
	}

	return pathConfig, nil
}
