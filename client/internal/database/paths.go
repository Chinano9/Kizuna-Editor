/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package database

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	DefaultAppFolder = "Kizuna"
	DefaultDBName    = "kizuna.db"
)

// ResolveDBPath determines the appropriate OS-compliant storage path for the SQLite database.
// It uses standard application data directories:
// - Windows: %APPDATA%/Kizuna/kizuna.db (fallback to ~/AppData/Roaming/Kizuna/kizuna.db)
// - macOS:   ~/Library/Application Support/Kizuna/kizuna.db
// - Linux:   $XDG_DATA_HOME/Kizuna/kizuna.db (fallback to ~/.local/share/Kizuna/kizuna.db)
// If resolving the directory fails, it returns the default local filename and the error.
func ResolveDBPath(appFolderName, dbName string) (string, error) {
	if appFolderName == "" {
		appFolderName = DefaultAppFolder
	}
	if dbName == "" {
		dbName = DefaultDBName
	}

	baseDir, err := getAppDataDir()
	if err != nil {
		return dbName, fmt.Errorf("unable to determine OS app data dir: %w", err)
	}

	fullPath := filepath.Join(baseDir, appFolderName, dbName)
	return fullPath, nil
}

// EnsureDirectory ensures that the parent directory for a given file path exists on disk.
func EnsureDirectory(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", dir, err)
	}
	return nil
}

func getAppDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			return appdata, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming"), nil

	case "darwin":
		configDir, err := os.UserConfigDir()
		if err == nil && configDir != "" {
			return filepath.Join(configDir), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil

	default: // Linux and other Unix-like systems
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return xdg, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}
