package main

import (
	"encoding/json"
	"log"
	"os"
)

type configPlayer struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
}

type configuration struct {
	Players             []configPlayer `json:"players"`
	RecordingsDirectory string         `json:"recordings_directory"`
	BindAddress         string         `json:"bind_address"`
	RiotAPIKey          string         `json:"riot_api_key"`
	RefreshRate         int            `json:"refresh_rate_seconds"`
	ShowPerPage         int            `json:"show_per_page"`
	ShowReplayPortAs    int            `json:"show_replay_port_as"`
	PostgresUrl         string         `json:"postgres_url"`
	BackendUrl          string         `json:"backend_url"`
	AdminKey            string         `json:"admin_key"`
	KeepRecordingsDays  int            `json:"keep_recordings_days"`
}

var config configuration

func readConfiguration(location string) {
	file, err := os.Open(location)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	dec := json.NewDecoder(file)

	err = dec.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	// default value, in case we delete everything
	if config.KeepRecordingsDays == 0 {
		config.KeepRecordingsDays = 30
	}
}
