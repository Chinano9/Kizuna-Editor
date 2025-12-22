package models

import "time"

// Album: Un contenedor de canciones
type Album struct {
	ID          int       `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Artist      string    `db:"artist" json:"artist"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// Song: La unidad básica
type Song struct {
	ID            int       `db:"id" json:"id"`
	AlbumID       *int      `db:"album_id" json:"album_id,omitempty"` // Puntero porque puede ser null
	Title         string    `db:"title" json:"title"`
	BPM           int       `db:"bpm" json:"bpm"`
	TimeSignature string    `db:"time_signature" json:"time_signature"`
	KeySignature  string    `db:"key_signature" json:"key_signature"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`

	// Relaciones (útiles para JSON, se ignoran en SQL simple)
	Tracks []Track `db:"-" json:"tracks,omitempty"`
}

// Instrument: Catálogo de instrumentos disponibles
type Instrument struct {
	ID          int    `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Type        string `db:"type" json:"type"` // 'Cuerda', 'Percusion'
	DefaultClef string `db:"default_clef" json:"default_clef"`
}

// Track: Una pista específica dentro de una canción
type Track struct {
	ID           int       `db:"id" json:"id"`
	SongID       int       `db:"song_id" json:"song_id"`
	InstrumentID int       `db:"instrument_id" json:"instrument_id"`
	Name         string    `db:"name" json:"name"`
	DataContent  string    `db:"data_content" json:"data_content"` // Tu código AlphaTex
	DisplayMode  string    `db:"display_mode" json:"display_mode"`
	IsMuted      bool      `db:"is_muted" json:"is_muted"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type AudioVersion struct {
	ID          int       `db:"id" json:"id"`
	SongID      int       `db:"song_id" json:"song_id"`
	VersionName string    `db:"version_name" json:"version_name"`
	FilePath    string    `db:"file_path" json:"file_path"`
	Notes       string    `db:"notes" json:"notes"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
