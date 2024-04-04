package racedata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

const UPLOAD_DIR = "data"

type RaceData struct {
	Seasons      SeasonMap `json:"season"`
	RaceDataFile string    `json:"racedata_filename"`
}

type Season struct {
	EntyListFile string `json:"entylist_filename"`
	Races        []Race
}

type SeasonMap map[string]Season

type Race struct {
	Name            string `json:"name"`
	QualyResultFile string `json:"qualy_result_file"`
	RaceResultFile  string `json:"race_result_file"`
}

func NewRaceData(filename string) *RaceData {

	newSeason := RaceData{
		RaceDataFile: filename,
		Seasons:      SeasonMap{},
	}

	if _, err := os.Stat(filename); err == nil {
		newSeason.readSeasonsFile()
	} else {
		newSeason.createSeasonsFile()
	}

	return &newSeason
}

func (s *RaceData) AddEntryList(seasonName string, entryList []byte) error {
	season, found := s.Seasons[seasonName]
	if !found {
		return fmt.Errorf("season %v not found", seasonName)
	}

	entyListFilename := path.Join(UPLOAD_DIR, nameToDir(seasonName), "enty_list.csv")
	log.Printf("add entry list %v to season %v\n", entyListFilename, season)

	if err := os.WriteFile(entyListFilename, entryList, 0644); err != nil {
		return err
	}

	// check entry list team name unique
	if err := checkEntryListUnique(entyListFilename); err != nil {
		log.Printf("%v\n", err)
		return err
	}

	season.EntyListFile = entyListFilename
	s.Seasons[seasonName] = season
	s.writeSeasonsFile()
	return nil

}

func (s *RaceData) AddPenalty(seasonName string, raceName string, penalty string, pos string) error {
	season, found := s.Seasons[seasonName]
	if !found {
		return fmt.Errorf("season %v not found", seasonName)
	}

	for _, race := range season.Races {
		if race.Name == raceName {
			raceResultFilename := race.RaceResultFile
			if err := addPenaltyToResult(raceResultFilename, penalty, pos); err != nil {
				return fmt.Errorf("can not apply penalty %v to pos %v in season %v and race %v", penalty, pos, seasonName, raceName)
			}
		}
	}
	return fmt.Errorf("race %v in season %v not found", raceName, seasonName)
}

func (s *RaceData) writeSeasonsFile() {
	b, err := json.MarshalIndent(s.Seasons, "", "   ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(s.RaceDataFile, b, 0644); err != nil {
		log.Fatal(err)
	}

}

func (s *RaceData) GetEntryListFilename(seasonName string) (string, error) {
	season, found := s.Seasons[seasonName]
	if !found {
		return "", fmt.Errorf("season %v not found", seasonName)
	}
	return season.EntyListFile, nil
}

func (s *RaceData) GetResultFilenames(seasonName string, raceName string) (string, string, error) {
	season, found := s.Seasons[seasonName]
	if !found {
		return "", "", fmt.Errorf("season %v not found", seasonName)
	}

	for _, race := range season.Races {
		if race.Name == raceName {
			return race.QualyResultFile, race.RaceResultFile, nil
		}
	}
	return "", "", fmt.Errorf("no race %v in season %v not found", raceName, seasonName)
}

func (s *RaceData) AddResults(seasonName string, raceName string, qualyResult []byte, raceResult []byte) error {
	log.Printf("add result to season %v race %v", seasonName, raceName)
	season, found := s.Seasons[seasonName]
	if !found {
		return fmt.Errorf("season %v not found", seasonName)
	}

	for i, race := range season.Races {
		if race.Name == raceName {
			qualyCsvFilename := path.Join(dataDir(seasonName, raceName), "qualy_result.csv")
			season.Races[i].QualyResultFile = qualyCsvFilename
			if err := os.WriteFile(qualyCsvFilename, qualyResult, 0644); err != nil {
				log.Fatal(err)
			}
			if err := addPenaltyColumn(qualyCsvFilename); err != nil {
				log.Fatal(err)
			}

			raceCsvFilename := path.Join(dataDir(seasonName, raceName), "race_result.csv")
			season.Races[i].RaceResultFile = raceCsvFilename
			if err := os.WriteFile(raceCsvFilename, raceResult, 0644); err != nil {
				log.Fatal(err)
			}
			if err := addPenaltyColumn(raceCsvFilename); err != nil {
				log.Fatal(err)
			}

			if err := areTeamNamesUnique(qualyCsvFilename); err != nil {
				return err
			}

			if err := areTeamNamesUnique(raceCsvFilename); err != nil {
				return err
			}
		}
	}

	s.Seasons[seasonName] = season
	s.writeSeasonsFile()
	return nil
}

func (s *RaceData) RemoveRace(seasonName string, raceName string) error {
	season, found := s.Seasons[seasonName]
	if !found {
		return fmt.Errorf("season %v not found", seasonName)
	}

	deleteIdx := -1
	for i, r := range season.Races {
		if r.Name == raceName {
			deleteIdx = i
		}
	}

	if deleteIdx == -1 {
		return fmt.Errorf("race %v in season %v not found", raceName, seasonName)
	}

	season.Races = append(season.Races[:deleteIdx], season.Races[deleteIdx+1:]...)
	s.Seasons[seasonName] = season
	s.writeSeasonsFile()
	return nil
}

func (s *RaceData) AddRace(seasonName string, raceName string) error {
	log.Printf("season %v add race %v", seasonName, raceName)
	season, found := s.Seasons[seasonName]
	if !found {
		return fmt.Errorf("season %v not found", seasonName)
	}

	for _, race := range season.Races {
		if race.Name == raceName {
			return fmt.Errorf("race name %v is not unique", raceName)
		}
	}

	season.Races = append(season.Races, Race{Name: raceName})
	s.Seasons[seasonName] = season
	s.writeSeasonsFile()

	if err := os.MkdirAll(path.Join(UPLOAD_DIR, nameToDir(seasonName), nameToDir(raceName)), 0770); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (s *RaceData) AddSeason(name string) error {
	_, found := s.Seasons[name]
	if found {
		return fmt.Errorf("season name %v is not unique", name)
	}

	s.Seasons[name] = Season{Races: []Race{}}
	s.writeSeasonsFile()

	if err := os.MkdirAll(path.Join(UPLOAD_DIR, nameToDir(name)), 0770); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (s *RaceData) readSeasonsFile() {
	file, err := os.Open(s.RaceDataFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(b, &s.Seasons)
}

func (s *RaceData) createSeasonsFile() {
	f, err := os.Create(s.RaceDataFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
}

func dataDir(seaon string, race string) string {
	return path.Join(UPLOAD_DIR,
		strings.ToLower(strings.ReplaceAll(seaon, " ", "_")),
		strings.ToLower(strings.ReplaceAll(race, " ", "_")))
}

func nameToDir(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}
