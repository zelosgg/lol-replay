package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/1lann/lol-replay/db"
	"github.com/1lann/lol-replay/record"
	"github.com/1lann/lol-replay/recording"
)

var platformToRegion = map[string]string{
	"NA1":  "na",
	"OC1":  "oce",
	"EUN1": "eune",
	"EUW1": "euw",
	"KR":   "kr",
	"BR1":  "br",
	"LA1":  "lan",
	"LA2":  "las",
	"RU":   "ru",
	"TR1":  "tr",
	"PBE1": "pbe",
}

type gameInfoMetadata struct {
	UserID          string
	BannedChampions []struct {
		ChampionID int `json:"championId"`
		PickTurn   int `json:"pickTurn"`
		TeamID     int `json:"teamId"`
	} `json:"bannedChampions"`
	GameID            int64  `json:"gameId"`
	GameLength        int    `json:"gameLength"`
	GameMode          string `json:"gameMode"`
	GameQueueConfigID int    `json:"gameQueueConfigId"`
	GameStartTime     int64  `json:"gameStartTime"`
	GameType          string `json:"gameType"`
	MapID             int    `json:"mapId"`
	Observers         struct {
		EncryptionKey string `json:"encryptionKey"`
	} `json:"observers"`
	Participants []struct {
		Bot        bool `json:"bot"`
		ChampionID int  `json:"championId"`
		Masteries  []struct {
			MasteryID int `json:"masteryId"`
			Rank      int `json:"rank"`
		} `json:"masteries"`
		ProfileIconID int `json:"profileIconId"`
		Runes         []struct {
			Count  int `json:"count"`
			RuneID int `json:"runeId"`
		} `json:"runes"`
		Spell1Id     int    `json:"spell1Id"`
		Spell2Id     int    `json:"spell2Id"`
		SummonerID   string `json:"summonerId"`
		SummonerName string `json:"summonerName"`
		TeamID       int    `json:"teamId"`
	} `json:"participants"`
	PlatformID string `json:"platformId"`
}

func monitorPlayers() {
	db, err := db.New(config.PostgresUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	waitSeconds := float64(config.RefreshRate) / float64(len(config.Players))
	waitPeriod := time.Millisecond * time.Duration(waitSeconds*1000.0)
	log.Println("Monitoring...")

	for {
		players, err := db.GetUsers()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting users: %v\n", err)
			time.Sleep(time.Second)
			continue
		}

		log.Printf("Monitoring %v users", len(players))

		for _, player := range players {
			time.Sleep(waitPeriod)
			log.Printf("Trying: %v", player.ID)
			info, ok := currentGameInfo(player, config.RiotAPIKey)

			if !ok {
				log.Println("Player not in a game...")
				continue
			}

			gameID := strconv.FormatInt(info.GameID, 10)
			keyName := info.PlatformID + "_" + gameID
			resume := false

			recordingsMutex.RLock()

			if _, found := recordings[keyName]; found {
				if recordings[keyName].temporary ||
					recordings[keyName].recording ||
					recordings[keyName].rec.IsComplete() {
					recordingsMutex.RUnlock()
					continue
				}

				if !recordings[keyName].rec.IsComplete() {
					resume = true
				}
			}

			recordingsMutex.RUnlock()
			recordingsMutex.Lock()

			if !resume {
				recordings[keyName] = &internalRecording{
					temporary: true,
					recording: false,
				}
			} else {
				recordings[keyName].temporary = true
				recordings[keyName].recording = false
			}

			cleanUp()
			recordingsMutex.Unlock()
			info.UserID = player.ID
			go recordGame(info, resume)
		}
	}
}

// recordingsMutex must be Locked before cleanUp is called.
func cleanUp() {
	for len(recordings) >= config.KeepNumRecordings {
		deleteRecording := sortedRecordings[0]
		deleteRecording.rec.Lock()
		deleteRecording.temporary = true
		deleteRecording.file.Close()
		err := os.Remove(deleteRecording.location)
		if err != nil {
			log.Println("failed to delete "+
				deleteRecording.location+":", err)
		} else {
			log.Println("deleted: " + deleteRecording.location)
		}

		deleteRecording.rec.Unlock()

		sortedRecordings = sortedRecordings[1:]

		for key, rec := range recordings {
			if rec == deleteRecording {
				delete(recordings, key)
				break
			}
		}
	}
}

func loadRecordGameFile(resume bool,
	keyName string) (*recording.Recording, *os.File, int, error) {
	var sortedKey = -1

	if !resume {
		file, err := os.Create(config.RecordingsDirectory + "/" + keyName + ".glr")
		if err != nil {
			log.Println("create recording error:", err)
			return nil, nil, sortedKey, err
		}

		rec, err := recording.NewRecording(file)
		if err != nil {
			log.Println("failed to initialize recording:", err)
			return nil, nil, sortedKey, err
		}

		return rec, file, sortedKey, nil
	}

	recordingsMutex.RLock()
	rec := recordings[keyName].rec
	file := recordings[keyName].file

	for i, internalRec := range sortedRecordings {
		if internalRec.rec == rec {
			sortedKey = i
			break
		}
	}
	recordingsMutex.RUnlock()

	return rec, file, sortedKey, nil
}

func recordGame(info gameInfoMetadata, resume bool) {
	gameID := strconv.FormatInt(info.GameID, 10)
	keyName := info.PlatformID + "_" + gameID

	defer func() {
		if e := recover(); e != nil {
			log.Printf("record game panic: %s: %s", e, debug.Stack())

			recordingsMutex.Lock()
			recordings[keyName].recording = false
			recordingsMutex.Unlock()
		}
	}()

	rec, file, sortedKey, err := loadRecordGameFile(resume, keyName)
	if err != nil {
		return
	}

	filename := path.Base(file.Name())

	recordingsMutex.Lock()
	recordings[keyName] = &internalRecording{
		file:      file,
		location:  config.RecordingsDirectory + "/" + filename,
		rec:       rec,
		temporary: false,
		recording: true,
	}

	if resume && sortedKey >= 0 {
		sortedRecordings[sortedKey] = recordings[keyName]
	} else {
		sortedRecordings = append(sortedRecordings, recordings[keyName])
	}
	recordingsMutex.Unlock()

	if !rec.HasUserMetadata() {
		if err = rec.StoreUserMetadata(&info); err != nil {
			log.Println("recording failed to store game metadata:", err)
			return
		}
	}

	if resume {
		log.Println("resuming recording " + keyName)
	} else {
		log.Println("recording " + keyName)
	}

	err = record.Record(info.PlatformID, gameID,
		info.Observers.EncryptionKey, rec)

	recordingsMutex.Lock()
	recordings[keyName].recording = false
	recordingsMutex.Unlock()

	if err != nil {
		log.Println("error while recording "+keyName+":", err)
		return
	}

	log.Println("recording " + keyName + " complete")
	go queueJob(info.UserID, info.GameID, info.Observers.EncryptionKey)
}

func queueJob(userID string, gameID int64, key string) {
	url := fmt.Sprintf("%s/api/v1/add-job", config.BackendUrl)

	body, _ := json.Marshal(map[string]interface{}{
		"userId":  userID,
		"matchId": gameID,
		"key":     key,
	})

	_, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	log.Printf("Queued up a job for user %v and match %v", userID, gameID)
}

func currentGameInfo(p db.Player, apiKey string) (gameInfoMetadata, bool) {
	url := "https://" + strings.ToLower(p.Platform) +
		".api.riotgames.com/lol/spectator/v4/active-games/by-summoner/" + p.SummonerID +
		"?api_key=" + apiKey

	for i := 0; i < 3; i++ {
		resp, err := http.Get(url)

		if err != nil {
			log.Println("URL:", url)
			log.Println("current game error:", err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return gameInfoMetadata{}, false
		}

		if resp.StatusCode != http.StatusOK {
			log.Println("URL:", url)
			log.Println("current game: not OK:", resp.Status)
			resp.Body.Close()
			continue
		}

		dec := json.NewDecoder(resp.Body)
		var info gameInfoMetadata
		err = dec.Decode(&info)
		resp.Body.Close()

		if err != nil {
			log.Println("URL:", url)
			log.Println("current error:", err)
			return gameInfoMetadata{}, false
		}

		return info, true
	}

	return gameInfoMetadata{}, false
}
