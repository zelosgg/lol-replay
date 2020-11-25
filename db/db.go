package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/pgxpool"
)

type DB struct {
	conn *pgxpool.Pool
}

type Player struct {
	ID         string `json:"id"`
	SummonerID string `json:"summoner_id"`
	Platform   string `json:"platform"`
}

func New(postgresUrl string) (*DB, error) {
	dbpool, err := pgxpool.Connect(context.Background(), postgresUrl)
	if err != nil {
		return nil, err
	}
	return &DB{dbpool}, nil
}

func (d *DB) GetUsers() ([]Player, error) {
	rows, err := d.conn.Query(context.Background(), `
		SELECT third_party_accounts.user_id, params ->> 'region', params ->> 'id'
		FROM third_party_accounts
		WHERE third_party_accounts.type = 'league'
	`)
	if err != nil {
		return nil, err
	}

	var players []Player
	for rows.Next() {
		var id, region, summonerId string
		err = rows.Scan(&id, &region, &summonerId)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading rows: %v\n", err)
			continue
		}
		players = append(players, Player{
			ID:         id,
			Platform:   region,
			SummonerID: summonerId,
		})
	}
	return players, nil
}

func (d *DB) Close() {
	defer d.conn.Close()
}
