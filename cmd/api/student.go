package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wasid786/schoolManagementSystem/internal/data"
	"github.com/Wasid786/schoolManagementSystem/internal/validator"
)

func (app *application) RegisterStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		SchoolID string `json:"school_id"` // or omit if generated
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Class    string `json:"class"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	schoolID := input.SchoolID
	if schoolID == "" {
		schoolID = generateSchoolID() // or however you generate it
	}

	student := &data.Student{
		SchoolID: schoolID,
		Name:     input.Name,
		Email:    input.Email,
		Class:    input.Class,
	}
	err = student.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.Validatestudent(v, student); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Students.Insert(student)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a student with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Tokens.New(int64(student.ID), 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"school_id":       student.SchoolID,
		}
		err = app.mailer.Send(student.Email, "student_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"student": student}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updatestudentPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password       string `json:"password"`
		TokenPlaintext string `json:"token"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := validator.New()
	data.ValidatePasswordPlaintext(v, input.Password)
	data.ValidateTokenPlaintext(v, input.TokenPlaintext)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	student, err := app.models.Students.GetForToken(data.ScopePasswordReset, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired password reset token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = student.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.models.Students.Update(student)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.models.Tokens.DeleteAllForStudent(data.ScopePasswordReset, int64(student.ID))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	env := envelope{"message": "your password was successfully reset "}
	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// func GetStudentHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
// 	schoolID := r.URL.Query().Get("school_id")
// 	// student, err := models.GetStudentByID(db, schoolID)

//		if err != nil {
//			http.Error(w, "Student not found", http.StatusNotFound)
//			return
//		}
//		json.NewEncoder(w).Encode(student)
//	}
func generateSchoolID() string {
	// Example: generate a random 8 character string
	return "SCH" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
