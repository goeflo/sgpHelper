package sgphelper

import (
	"embed"
	"html/template"
	"io"
	"log"
	"net/http"
	"sgpHelper/racedata"
	"strings"
	"time"
)

const MAX_UPLOAD_SIZE = 1024 * 1024 // 1MB

type Server struct {
	server *http.Server
	season *racedata.RaceData
}

var funcMap = map[string]interface{}{
	"add": func(a, b int) int {
		return a + b
	},
}

var indexTmpl = template.Must(template.New("funcMap").Funcs(funcMap).ParseFiles("templates/index.html"))
var raceTmpl = template.Must(template.New("funcMap").Funcs(funcMap).ParseFiles("templates/race.html"))
var entrylistTmpl = template.Must(template.New("funcMap").Funcs(funcMap).ParseFiles("templates/entrylist.html"))

//go:embed public/*
var publicFS embed.FS

func NewServer(addr string, s *racedata.RaceData) *Server {
	return &Server{
		server: &http.Server{Addr: addr},
		season: s,
	}
}

func (s *Server) Start() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/show/{season}/{race}", s.handleShowRace)
	mux.HandleFunc("/delete/{season}/{race}", s.handleDeleteRace)
	mux.HandleFunc("/addPenalty/{season}/{race}", s.handleAddPenalty)
	mux.HandleFunc("/newSeason", s.handleNewSeason)
	mux.HandleFunc("/upload/{season}", s.handleUpload)
	mux.HandleFunc("/export/csv/{season}/{race}/{split}", s.handleExportRace)
	mux.HandleFunc("/uploadEntryList/{season}", s.handleUploadEntyList)
	mux.HandleFunc("/showEntryList/*", s.handleShowEntyList)
	mux.Handle("/public/", http.FileServer(http.FS(publicFS)))
	s.server.Handler = mux

	log.Println("server address: ", s.server.Addr)

	return s.server.ListenAndServe()

}

func (s *Server) handleShowEntyList(w http.ResponseWriter, r *http.Request) {
	log.Printf("-> handleShowEntyList entylist %v\n", strings.ReplaceAll(r.RequestURI, "/showEntryList/", ""))
	defer logDuration(r.RequestURI, time.Now())

	entryList, err := s.season.GetEntryList(strings.ReplaceAll(r.RequestURI, "/showEntryList/", ""))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := entrylistTmpl.ExecuteTemplate(w, "entrylist.html", entryList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleExportRace(w http.ResponseWriter, r *http.Request) {
	log.Printf("-> handleExportRace season %v, race %v, split %v\n", r.PathValue("season"), r.PathValue("race"), r.PathValue("split"))
	defer logDuration(r.RequestURI, time.Now())

	seasonName := r.PathValue("season")
	raceName := r.PathValue("race")
	splitName := r.PathValue("split")

	raceResult, err := s.season.GetRaceResult(seasonName, raceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	racedata.GetCSVExport(raceResult, splitName, w)
}

func (s *Server) handleAddPenalty(w http.ResponseWriter, r *http.Request) {
	pos := r.PostFormValue("pos")
	penalty := r.PostFormValue("penalty")
	seasonName := r.PathValue("season")
	raceName := r.PathValue("race")
	log.Printf("handleAddPenalty season: %v race: %v, add penalty %v to pos %v\n", seasonName, raceName, penalty, pos)
	s.season.AddPenalty(seasonName, raceName, penalty, pos)
	s.handleShowRace(w, r)
}

func (s *Server) handleDeleteRace(w http.ResponseWriter, r *http.Request) {
	seasonName := r.PathValue("season")
	raceName := r.PathValue("race")
	log.Printf("handleDeleteRace season %v race %v\n", seasonName, raceName)
	s.season.RemoveRace(seasonName, raceName)
	s.handleIndex(w, r)
}

func (s *Server) handleShowRace(w http.ResponseWriter, r *http.Request) {
	log.Printf("-> handleShowRace race %v, season %v\n", r.PathValue("season"), r.PathValue("name"))
	defer logDuration(r.RequestURI, time.Now())

	seasonName := r.PathValue("season")
	raceName := r.PathValue("race")

	raceResult, err := s.season.GetRaceResult(seasonName, raceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := raceTmpl.ExecuteTemplate(w, "race.html", raceResult); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *Server) handleNewSeason(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleNewSeason\n")
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	newSeasonName := r.PostFormValue("new_season_name")
	log.Printf("new season: %v\n", newSeasonName)
	if err := s.season.AddSeason(newSeasonName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.handleIndex(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("-> handleIndex\n")
	defer logDuration(r.RequestURI, time.Now())
	if err := indexTmpl.ExecuteTemplate(w, "index.html", s.season.Seasons); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) handleUploadEntyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	seasonName := r.PathValue("season")
	log.Printf("handleUploadEntyList for seasons %v\n", seasonName)

	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)

	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose an file that's less than 1MB in size", http.StatusBadRequest)
		return
	}

	entryListFile, _, err := r.FormFile("entry_list")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer entryListFile.Close()

	entryList, err := io.ReadAll(entryListFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.season.AddEntryList(seasonName, entryList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.handleIndex(w, r)

}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleUpload\n")

	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	newRaceName := r.PostFormValue("new_race_name")
	seasonName := r.PathValue("season")

	if err := s.season.AddRace(seasonName, newRaceName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)

	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose an file that's less than 1MB in size", http.StatusBadRequest)
		return
	}

	qualyFile, _, err := r.FormFile("qualy_result")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer qualyFile.Close()

	qualyResult, err := io.ReadAll(qualyFile)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	RaceFile, _, err := r.FormFile("race_result")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer RaceFile.Close()

	raceResult, err := io.ReadAll(RaceFile)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.season.AddResults(seasonName, newRaceName, qualyResult, raceResult); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.handleIndex(w, r)

}

func logDuration(s string, t time.Time) {
	d := time.Now().Sub(t)
	log.Printf("<- %v time: %v\n", s, d)
}
