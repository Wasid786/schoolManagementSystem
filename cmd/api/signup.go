package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Wasid786/schoolManagementSystem/internal"
	"github.com/Wasid786/schoolManagementSystem/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	db := internal.Dbconnect()
	defer db.Close()

	var existingUser string
	err := db.QueryRow("SELECT email FROM users WHERE email = ?", user.Email).Scan(&existingUser)
	switch {
	case err == sql.ErrNoRows:
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "unable to create account", http.StatusInternalServerError)
			return
		}
		if user.Role == "" {
			user.Role = "student"

		}

		_, err = db.Exec("INSERT INTO users (username, email, password, role) VALUES (?, ?, ?,?)", user.Username, user.Email, hashedPassword, user.Role)

		if err != nil {
			http.Error(w, "server unable to create user", http.StatusInternalServerError)
			fmt.Println("Insert error:", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("user has been created"))

	case err != nil:
		http.Error(w, "server db query error", http.StatusInternalServerError)
		fmt.Println("Query error:", err)
		return

	default:
		http.Error(w, "user already exists", http.StatusBadRequest)
	}
}
