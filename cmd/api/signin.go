package main

import (
	"encoding/json"
	"net/http"

	"github.com/Wasid786/schoolManagementSystem/internal"
	"github.com/Wasid786/schoolManagementSystem/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) signinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		panic(err)
	}
	dbs := internal.Dbconnect()
	defer dbs.Close()
	var newpassword string
	err = dbs.QueryRow("select password from users where email=?", user.Email).Scan(&newpassword)
	switch {
	case err != nil:
		http.Error(w, "unauthorized no user with this email", http.StatusUnauthorized)
		return
	default:
		err := bcrypt.CompareHashAndPassword([]byte(newpassword), []byte(user.Password))
		if err != nil {
			http.Error(w, "unathorized password wrong", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("welcome" + user.Username))
	}

}
