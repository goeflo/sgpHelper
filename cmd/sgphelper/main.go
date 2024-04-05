package main

import (
	"log"
	sgphelper "sgpHelper"
	"sgpHelper/config"
	"sgpHelper/racedata"
)

func main() {

	config := config.NewConfig()
	config.ReadFile()
	log.Printf("%v", config)

	season := racedata.NewRaceData(config.Server.DataDir, config.Server.RaceData)

	s := sgphelper.NewServer(":"+config.Server.Port, season)
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
