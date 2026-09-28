package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Enquiry mirrors one row in the enquiries table.
type Enquiry struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Company   string    `json:"company"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Very small email format check — not exhaustive RFC 5322, just enough
// to catch obvious mistakes like a missing "@".
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

var validStatuses = map[string]bool{
	"New":         true,
	"In Progress": true,
	"Closed":      true,
}

func createEnquiryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input Enquiry
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Trim whitespace so " " doesn't pass as valid input.
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Company = strings.TrimSpace(input.Company)
	input.Message = strings.TrimSpace(input.Message)

	// Validation
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if input.Email == "" || !emailRegex.MatchString(input.Email) {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if input.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if len(input.Name) > 255 || len(input.Email) > 255 || len(input.Company) > 255 {
		writeError(w, http.StatusBadRequest, "one or more fields exceed the maximum length")
		return
	}

	result, err := db.Exec(
		`INSERT INTO enquiries (name, email, company, message) VALUES (?, ?, ?, ?)`,
		input.Name, input.Email, input.Company, input.Message,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save enquiry")
		return
	}

	id, _ := result.LastInsertId()
	input.ID = int(id)
	input.Status = "New"
	
	go sendEnquiryNotification(input)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func listEnquiriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rows, err := db.Query(
		`SELECT id, name, email, company, message, status, created_at
		 FROM enquiries ORDER BY created_at DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch enquiries")
		return
	}
	defer rows.Close()

	enquiries := []Enquiry{}
	for rows.Next() {
		var e Enquiry
		var company sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.Email, &company, &e.Message, &e.Status, &e.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read enquiry row")
			return
		}
		e.Company = company.String
		enquiries = append(enquiries, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(enquiries)
}

func updateStatusHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !validStatuses[input.Status] {
		writeError(w, http.StatusBadRequest, "status must be one of: New, In Progress, Closed")
		return
	}

	result, err := db.Exec(`UPDATE enquiries SET status = ? WHERE id = ?`, input.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "enquiry not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": input.Status})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}