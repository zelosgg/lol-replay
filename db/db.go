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
	ID       string `json:"id"`
	Platform string `json:"platform"`
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
		SELECT params ->> 'region', params ->> 'summonerId'
		FROM third_party_accounts
		INNER JOIN users_roles ON users_roles.user_id = third_party_accounts.user_id
		WHERE third_party_accounts.type = 'league' AND users_roles.role = 'clip'
	`)
	if err != nil {
		return nil, err
	}

	var players []Player
	for rows.Next() {
		var region, accountId string
		err = rows.Scan(&region, &accountId)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading rows: %v\n", err)
		}
		players = append(players, Player{
			ID:       accountId,
			Platform: region,
		})
	}
	return players, nil
}

func (d *DB) Close() {
	defer d.conn.Close()
}
