package racedata

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/artyom/csvstruct"
)

const TOTAL_TIME_COLUMN = 5
const POS_COLUMN = 0
const PENALTY_COLUMN = 11

type CSVEntryListLine struct {
	Driver     string `csv:"driver"`
	Team       string `csv:"team"`
	Car        string `csv:"car"`
	RaceNumber string `csv:"race_number"`
	Class      string `csv:"class"`
}

type CSVEntryList []CSVEntryListLine

type CSVResultLine struct {
	Pos              uint   `csv:"pos"`
	StartPos         string `csv:"startPos"`
	Participant      string `csv:"participant"`
	Car              string `csv:"car"`
	Class            string `csv:"class"`
	TotalTime        string `csv:"totalTime"`
	BestLapTime      string `csv:"bestLapTime"`
	BestCleanLapTime string `csv:"bestCleanLapTime"`
	Laps             string `csv:"laps"`
	Penalty          string `csv:"penalty"`
}

type CSVResult []CSVResultLine

func readEntryList(entryListFilename string) (*CSVEntryList, error) {
	log.Printf("read entry list: %v\n", entryListFilename)
	file, err := os.Open(entryListFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	scan, err := csvstruct.NewScanner(header, &CSVEntryListLine{})
	if err != nil {
		log.Fatalf("new scanner for header %v - %v", header, err)
	}

	lines := CSVEntryList{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		var line CSVEntryListLine
		if err := scan(record, &line); err != nil {
			log.Fatalf("can not parse line %v - %v", record, err)
		}
		lines = append(lines, line)
	}

	return &lines, nil
}

func readResult(resultFilename string) (*CSVResult, error) {
	file, err := os.Open(resultFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	scan, err := csvstruct.NewScanner(header, &CSVResultLine{})
	if err != nil {
		log.Fatal(err)
	}

	result := CSVResult{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		var line CSVResultLine
		if err := scan(record, &line); err != nil {
			log.Fatal(err)
		}
		result = append(result, line)
	}
	return &result, nil
}

func convertMilliseconds(milliseconds string) string {

	if milliseconds == "" {
		return ""
	}

	m, err := strconv.Atoi(milliseconds)
	if err != nil {
		log.Fatal(err)
	}
	d := time.Duration(m) * time.Millisecond

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)

}

func checkEntryListUnique(filename string) error {

	entryList, err := readEntryList(filename)
	if err != nil {
		return err
	}

	teamMap := make(map[string]string)
	for _, line := range *entryList {
		_, found := teamMap[line.Team]
		if found {
			log.Printf("team %v already exists in team map", line.Team)
		}
		teamMap[line.Team] = line.Team
	}
	if len(teamMap) != len(*entryList) {
		return fmt.Errorf("team names %v %v in entry list %v are not unique", len(teamMap), len(*entryList), filename)
	}
	log.Printf("entry list %v is ok\n", filename)

	return nil
}

func addPenaltyToResult(filename string, penalty string, pos string) error {

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	file.Close()

	if err := os.Remove(filename); err != nil {
		return err
	}

	for i := range records {
		if i == 0 { // skip header line
			continue
		}

		if records[i][POS_COLUMN] != pos {
			continue
		}

		if records[i][PENALTY_COLUMN] == "0" {
			records[i][PENALTY_COLUMN] = penalty
		} else {
			penaltyInt, err := strconv.Atoi(records[i][PENALTY_COLUMN])
			if err != nil {
				return err
			}

			addPenaltyInt, err := strconv.Atoi(penalty)
			if err != nil {
				return err
			}
			log.Printf("pos %v, penalty %v + %v = %v\n", pos, penaltyInt, addPenaltyInt, (addPenaltyInt + penaltyInt))
			records[i][PENALTY_COLUMN] = fmt.Sprintf("%v", addPenaltyInt+penaltyInt)
		}

	}

	writeFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	w := csv.NewWriter(writeFile)
	if err := w.WriteAll(records); err != nil {
		return err
	}

	return nil
}

func areTeamNamesUnique(filename string) error {
	results, err := readResult(filename)
	if err != nil {
		return err
	}

	var teamMap = make(map[string]string)
	for _, result := range *results {
		teamMap[result.Participant] = result.Participant
	}

	log.Printf("team map length %v team result length %v", len(teamMap), len(*results))

	if len(teamMap) != len(*results) {
		return fmt.Errorf("team names are not unique")
	}
	return nil
}

func addPenaltyColumn(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	r := csv.NewReader(file)

	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	file.Close()

	if err := os.Remove(filename); err != nil {
		return err
	}

	for i := range records {
		if i == 0 {
			records[i] = append(records[i], "penalty")
		} else {
			records[i] = append(records[i], "0")
		}
	}

	writeFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	w := csv.NewWriter(writeFile)
	if err := w.WriteAll(records); err != nil {
		return err
	}

	return nil
}
