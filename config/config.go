package config

import (
	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
	"os"
)

var (
	Data *File
)

type (
	File struct {
		Games []Game `yaml:"games"`
	}

	Game struct {
		Slug             string `yaml:"slug"`
		Title            string `yaml:"title"`
		Thumbnail        string `yaml:"thumbnail"`
		Source           string `yaml:"source"`
		WorkingDirectory string `yaml:"working_directory"`
		Entrypoint       string `yaml:"entrypoint"`
		Save             string `yaml:"save"`
	}
)

func init() {
	Data = load()
}

func load() *File {
	f, err := os.Open("./config.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			c := &File{}
			if f, err = os.Create("./config.yaml"); err == nil {
				defer f.Close()
				enc := yaml.NewEncoder(f)
				_ = enc.Encode(c)
			}
			return c
		} else {
			log.Fatal("failed to read config file", "err", err)
		}
	}
	defer f.Close()

	var c File
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&c); err != nil {
		log.Fatal("failed to parse config file", "err", err)
		return &File{}
	}

	return &c
}

func FindGame(slug string) (*Game, bool) {
	for _, g := range Data.Games {
		if g.Slug == slug {
			return &g, true
		}
	}
	return nil, false
}
