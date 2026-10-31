// space bounty hunters: criminals + hunt logs
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// criminal_id points to a criminal that does not exist
var ErrBadReference = errors.New("referenced row does not exist")

type Criminal struct {
	ID                   int64   `db:"id"                     json:"id"`
	Name                 *string `db:"name"                   json:"name"`
	Nickname             string  `db:"nickname"               json:"nickname"`
	Reward               int     `db:"reward"                 json:"reward"`
	LastKnownCoordinates *string `db:"last_known_coordinates" json:"last_known_coordinates"`
	Difficulty           int     `db:"difficulty"             json:"difficulty"`
	Armed                bool    `db:"armed"                  json:"armed"`
	HasCrew              bool    `db:"has_crew"               json:"has_crew"`
	EscapeMaster         bool    `db:"escape_master"          json:"escape_master"`
}

type CriminalFilter struct {
	MinReward    *int
	MaxReward    *int
	Armed        *bool
	HasCrew      *bool
	EscapeMaster *bool
	Sort         string // "", "reward" or "-reward"
}

type HuntLog struct {
	ID         int64     `db:"id"          json:"id"`
	CriminalID int64     `db:"criminal_id" json:"criminal_id"`
	Success    bool      `db:"success"     json:"success"`
	Notes      string    `db:"notes"       json:"notes"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}

const criminalCols = `id, name, nickname, reward, last_known_coordinates, difficulty, armed, has_crew, escape_master`
const huntCols = `id, criminal_id, success, notes, created_at`

func (s *Store) ListCriminals(ctx context.Context, f CriminalFilter) ([]Criminal, error) {
	var conds []string
	var args []any
	add := func(format string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(format, len(args)))
	}
	if f.MinReward != nil {
		add("reward >= $%d", *f.MinReward)
	}
	if f.MaxReward != nil {
		add("reward <= $%d", *f.MaxReward)
	}
	if f.Armed != nil {
		add("armed = $%d", *f.Armed)
	}
	if f.HasCrew != nil {
		add("has_crew = $%d", *f.HasCrew)
	}
	if f.EscapeMaster != nil {
		add("escape_master = $%d", *f.EscapeMaster)
	}

	query := "SELECT " + criminalCols + " FROM criminals"
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	// order by comes from a fixed list, never from user text
	switch f.Sort {
	case "reward":
		query += " ORDER BY reward, id"
	case "-reward":
		query += " ORDER BY reward DESC, id"
	default:
		query += " ORDER BY id"
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[Criminal])
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Criminal{}
	}
	return list, nil
}

func (s *Store) GetCriminal(ctx context.Context, id int64) (Criminal, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+criminalCols+" FROM criminals WHERE id = $1", id)
	if err != nil {
		return Criminal{}, err
	}
	c, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Criminal])
	if errors.Is(err, pgx.ErrNoRows) {
		return Criminal{}, ErrNotFound
	}
	return c, err
}

// criminalID nil = all logs
func (s *Store) ListHunts(ctx context.Context, criminalID *int64) ([]HuntLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+huntCols+` FROM hunt_logs
		WHERE $1::bigint IS NULL OR criminal_id = $1
		ORDER BY created_at DESC, id DESC`, criminalID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[HuntLog])
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []HuntLog{}
	}
	return list, nil
}

func (s *Store) GetHunt(ctx context.Context, id int64) (HuntLog, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+huntCols+" FROM hunt_logs WHERE id = $1", id)
	if err != nil {
		return HuntLog{}, err
	}
	h, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[HuntLog])
	if errors.Is(err, pgx.ErrNoRows) {
		return HuntLog{}, ErrNotFound
	}
	return h, err
}

// created_at is not in the insert, db default now() fills it
func (s *Store) CreateHunt(ctx context.Context, criminalID int64, success bool, notes string) (HuntLog, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO hunt_logs (criminal_id, success, notes)
		VALUES ($1, $2, $3)
		RETURNING `+huntCols, criminalID, success, notes)
	if err != nil {
		return HuntLog{}, mapFKError(err)
	}
	h, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[HuntLog])
	if err != nil {
		return HuntLog{}, mapFKError(err)
	}
	return h, nil
}

// 23503 = foreign key violation
func mapFKError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrBadReference
	}
	return err
}

func (s *Store) UpdateHunt(ctx context.Context, id int64, success *bool, notes *string) (HuntLog, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE hunt_logs SET
			success = COALESCE($2, success),
			notes   = COALESCE($3, notes)
		WHERE id = $1
		RETURNING `+huntCols, id, success, notes)
	if err != nil {
		return HuntLog{}, err
	}
	h, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[HuntLog])
	if errors.Is(err, pgx.ErrNoRows) {
		return HuntLog{}, ErrNotFound
	}
	return h, err
}

func (s *Store) DeleteHunt(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM hunt_logs WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
