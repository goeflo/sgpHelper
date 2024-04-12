package racedata

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
)

// RaceResult data struct to send race data to html template
type RaceResult struct {
	SeasonName            string
	RaceName              string
	QualiyResult          map[string]ResultLines
	RaceResult            map[string]ResultLines
	RaceResultWithPenalty map[string]ResultLines
}

type Driver struct {
	Driver     string
	Team       string
	Car        string
	RaceNumber string
	Class      string
}

type EntryList []Driver

type ResultLine struct {
	Pos              uint
	StartPos         string
	Driver           string
	Team             string
	Startnumber      string
	Car              string
	Class            string
	TotalTime        string
	BestLapTime      string
	BestCleanLapTime string
	Laps             string
	Penalty          string
}

type ResultLines []ResultLine

func (r ResultLines) Len() int {
	return len(r)
}

// Less sort totalTime first, lap count second
func (r ResultLines) Less(i, j int) bool {

	ti, _ := strconv.Atoi(r[i].TotalTime)
	tj, _ := strconv.Atoi(r[j].TotalTime)

	li, _ := strconv.Atoi(r[i].Laps)
	lj, _ := strconv.Atoi(r[j].Laps)

	if li != lj {
		return li > lj
	}

	return ti < tj

}

func (r ResultLines) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (s *RaceData) GetEntryList(filename string) (*EntryList, error) {
	csvEntryList, err := readEntryList(filename)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	entryList := EntryList{}

	for _, line := range *csvEntryList {
		entryList = append(entryList, Driver{
			Driver:     line.Driver,
			Team:       line.Team,
			Car:        line.Car,
			RaceNumber: line.RaceNumber,
			Class:      line.Class,
		})
	}

	return &entryList, nil
}

func (s *RaceData) GetRaceResult(seasonName string, raceName string) (*RaceResult, error) {

	qualyFilename, raceFilename, err := s.GetResultFilenames(seasonName, raceName)
	if err != nil {
		return nil, fmt.Errorf("season %v and race %v", seasonName, raceName)
	}

	entyListFilename, err := s.GetEntryListFilename(seasonName)
	if err != nil {
		return nil, fmt.Errorf("no entry list found for season %v", seasonName)
	}

	qualyResult, err := readResult(qualyFilename)
	if err != nil {
		return nil, fmt.Errorf("can not read qualy file %v", qualyFilename)
	}

	raceResult, err := readResult(raceFilename)
	if err != nil {
		return nil, fmt.Errorf("can not read result file %v", raceFilename)
	}

	entryList, err := readEntryList(entyListFilename)
	if err != nil {
		return nil, fmt.Errorf("can not read entry list %v", entryList)
	}

	rr := toRaceResult(qualyResult, raceResult, entryList)
	rr.RaceName = raceName
	rr.SeasonName = seasonName

	return rr, nil

}

func (r *ResultLine) addDriverAndRaceNumber(el *CSVEntryList) {
	for _, v := range *el {
		if v.Team == r.Team {
			r.Driver = v.Driver
			r.Startnumber = v.RaceNumber
			return
		}
	}
	r.Driver = "N/A"
}

func toRaceResult(qr *CSVResult, rr *CSVResult, el *CSVEntryList) *RaceResult {

	raceResult := &RaceResult{
		RaceResult:            map[string]ResultLines{},
		RaceResultWithPenalty: map[string]ResultLines{},
	}

	for _, line := range *rr {
		_, found := raceResult.RaceResult[line.Class]
		if !found {
			raceResult.RaceResult[line.Class] = ResultLines{}
		}
		oldLine := raceResult.RaceResult[line.Class]
		resultLine := csvResultToResultLine(line)
		resultLine.addDriverAndRaceNumber(el)
		oldLine = append(oldLine, resultLine)
		raceResult.RaceResult[line.Class] = oldLine
	}

	for k := range raceResult.RaceResult {
		for _, v := range raceResult.RaceResult[k] {

			resultLine := v
			if resultLine.Penalty != "0" {
				p, err := strconv.Atoi(resultLine.Penalty)
				if err != nil {
					log.Fatal(err)
				}
				t, err := strconv.Atoi(resultLine.TotalTime)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("total time: %v + %v = %v\n", resultLine.TotalTime, (p * 1000), (t + (p * 1000)))
				resultLine.TotalTime = fmt.Sprintf("%v", t+(p*1000))
			}
			raceResult.RaceResultWithPenalty[k] = append(raceResult.RaceResultWithPenalty[k], resultLine)
		}
	}

	for k := range raceResult.RaceResultWithPenalty {
		sort.Sort(raceResult.RaceResultWithPenalty[k])
	}

	for k := range raceResult.RaceResultWithPenalty {
		for i := range raceResult.RaceResultWithPenalty[k] {
			raceResult.RaceResultWithPenalty[k][i].BestLapTime = convertMilliseconds(raceResult.RaceResultWithPenalty[k][i].BestLapTime)
			raceResult.RaceResultWithPenalty[k][i].TotalTime = convertMilliseconds(raceResult.RaceResultWithPenalty[k][i].TotalTime)

			raceResult.RaceResult[k][i].BestLapTime = convertMilliseconds(raceResult.RaceResult[k][i].BestLapTime)
			raceResult.RaceResult[k][i].TotalTime = convertMilliseconds(raceResult.RaceResult[k][i].TotalTime)
		}

	}

	return raceResult
}

func csvResultToResultLine(line CSVResultLine) ResultLine {
	return ResultLine{
		Pos:              line.Pos,
		StartPos:         line.StartPos,
		Team:             line.Participant,
		Car:              line.Car,
		Class:            line.Class,
		TotalTime:        line.TotalTime,
		BestLapTime:      line.BestLapTime,
		BestCleanLapTime: line.BestCleanLapTime,
		Laps:             line.Laps,
		Penalty:          line.Penalty,
	}
}

func GetCSVExport(raceResult *RaceResult, split string, w io.Writer) {

	fmt.Fprintf(w, "split pos, race pos,laps,race number,team,driver,penalty,ziel zeit\n")
	resultLines := raceResult.RaceResultWithPenalty[split]

	for i, line := range resultLines {
		if line.Laps == "0" {
			continue
		}
		// TODO check amount of laps more than 50%
		fmt.Fprintf(w, "%v,%v,%v,%v,%v,%v,%v,%v\n", i+1, line.Pos, line.Laps, line.Startnumber, line.Team, line.Driver, line.Penalty, line.TotalTime)
	}

}
