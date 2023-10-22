// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.22.0

package sqlvods

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Stream struct {
	ID                               uuid.UUID
	StreamerID                       string
	StreamID                         string
	StartTime                        time.Time
	MaxViews                         int64
	LastUpdatedAt                    time.Time
	StreamerLoginAtStart             string
	LanguageAtStart                  string
	TitleAtStart                     string
	GameNameAtStart                  string
	GameIDAtStart                    string
	IsMatureAtStart                  bool
	LastUpdatedMinusStartTimeSeconds float64
	RecordingFetchedAt               sql.NullTime
	GzippedBytes                     []byte
	HlsDomain                        sql.NullString
	HlsDurationSeconds               sql.NullFloat64
	BytesFound                       sql.NullBool
	Public                           sql.NullBool
	BoxArtUrlAtStart                 sql.NullString
	ProfileImageUrlAtStart           sql.NullString
}

type Streamer struct {
	ID                     uuid.UUID
	StartTime              time.Time
	StreamerLoginAtStart   string
	StreamerID             string
	ProfileImageUrlAtStart sql.NullString
}
