-- Migration: 000002_add_indexes_and_constraints
-- Description: Create indexes for foreign keys and frequent query sorting to boost performance

CREATE INDEX IF NOT EXISTS idx_songs_album_id ON songs(album_id);
CREATE INDEX IF NOT EXISTS idx_songs_updated_at ON songs(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_song_id ON tracks(song_id);
CREATE INDEX IF NOT EXISTS idx_tracks_instrument_id ON tracks(instrument_id);
CREATE INDEX IF NOT EXISTS idx_audio_versions_song_id ON audio_versions(song_id);
CREATE INDEX IF NOT EXISTS idx_audio_versions_track_id ON audio_versions(track_id);
