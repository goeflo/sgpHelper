package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Port     string `yaml:"port"`
	DataDir  string `yaml:"dataDir"`
	RaceData string `yaml:"raceData"`
}

func (c Config) String() string {
	return fmt.Sprintf("config server port: %v data dir: %v race data file: %v\n",
		c.Server.Port, c.Server.DataDir, c.Server.RaceData)
}

// NewConfig create new default config
func NewConfig() *Config {
	return &Config{
		Server: Server{
			Port:     "8080",
			DataDir:  "data",
			RaceData: "race_data.json",
		},
	}
}

// ReadFile read config yml file in default root location
func (c *Config) ReadFile() {
	f, err := os.Open("config.yml")
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(c)
	if err != nil {
		log.Panic(err)
	}
}
