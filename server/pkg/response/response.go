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

package response

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard JSON API response payload.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorItem  `json:"error,omitempty"`
}

// ErrorItem defines structured error details.
type ErrorItem struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// JSON sends a formatted JSON response with the given HTTP status code.
func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// Success sends an HTTP 2xx success response wrapped in standard envelope.
func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, Response{
		Success: true,
		Data:    data,
	})
}

// Error sends an HTTP error response wrapped in standard envelope.
func Error(w http.ResponseWriter, status int, message string, details ...string) {
	var detail string
	if len(details) > 0 {
		detail = details[0]
	}

	JSON(w, status, Response{
		Success: false,
		Error: &ErrorItem{
			Code:    status,
			Message: message,
			Details: detail,
		},
	})
}
