package main

import (
	"log"
	sgphelper "sgpHelper"
	"sgpHelper/racedata"
)

func main() {

	season := racedata.NewRaceData("race_data.json")

	s := sgphelper.NewServer(":8080", season)
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
