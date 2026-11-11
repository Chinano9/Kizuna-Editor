-- Migration: 000001_initial_schema
-- Description: Core schema initialization for Kizuna Editor (albums, songs, instruments, tracks, audio_versions)

CREATE TABLE IF NOT EXISTS albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    artist TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS songs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    album_id INTEGER,
    title TEXT NOT NULL,
    bpm INTEGER DEFAULT 120,
    time_signature TEXT DEFAULT '4/4',
    key_signature TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS instruments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    default_clef TEXT DEFAULT 'treble'
);

CREATE TABLE IF NOT EXISTS tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    instrument_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    data_content TEXT NOT NULL,
    display_mode TEXT DEFAULT 'BOTH',
    is_muted BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    FOREIGN KEY (instrument_id) REFERENCES instruments(id)
);

CREATE TABLE IF NOT EXISTS audio_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    track_id INTEGER,
    version_name TEXT,
    file_path TEXT NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
);
