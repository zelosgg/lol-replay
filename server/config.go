package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
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
	TotalShards         int            `json:"total_shards"`
	Shard               int            `json:"shard"`
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

	if config.TotalShards == 0 {
		// TODO: we should honestly just use environment variables for all the config values
		totalShards, err := strconv.Atoi(os.Getenv("REPLAY_TOTAL_SHARDS"))
		if err == nil {
			config.TotalShards = totalShards
		}
	}

	if config.Shard == 0 {
		shard, err := strconv.Atoi(os.Getenv("REPLAY_SHARD"))
		if err == nil {
			config.Shard = shard
		}
	}

	log.Printf("Loading configuration for shard %v\n", config.Shard)
}
