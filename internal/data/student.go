package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wasid786/schoolManagementSystem/internal/validator"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type Student struct {
	ID        int      `json:"id"`
	SchoolID  string   `json:"school_id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Password  password `json:"password"`
	Class     string   `json:"class"`
	CreatedAt string   `json:"created_at"`
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}
func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}
func Validatestudent(v *validator.Validator, student *Student) {
	v.Check(student.Name != "", "name", "must be provided")
	v.Check(len(student.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, student.Email)

	if student.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *student.Password.plaintext)
	}

	if student.Password.hash == nil {
		panic("missing password hash for student")
	}
}

type StudentModel struct {
	DB *sql.DB
}

func (m *StudentModel) Insert(student *Student) error {
	query := `INSERT INTO students (school_id, name, email, password, class) VALUES (?, ?, ?, ?, ?)`

	args := []interface{}{student.SchoolID, student.Name, student.Email, student.Password.hash, student.Class}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 && strings.Contains(mysqlErr.Message, "email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	student.ID = int(id)

	return nil
}

func (m StudentModel) GetByEmail(email string) (*Student, error) {
	query := `SELECT id,school_id, name, email, password, class,created_at FROM students WHERE email = ?`
	var student Student

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&student.ID,
		&student.SchoolID,
		&student.Name,
		&student.Email,
		&student.Password.hash,
		&student.Class,
		&student.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &student, nil
}

func (m StudentModel) Get(school_id string) (*Student, error) {
	query := `SELECT id,school_id, name, email, password, class,created_at FROM students WHERE school_id = ?`
	var student Student

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, school_id).Scan(
		&student.ID,
		&student.SchoolID,
		&student.Name,
		&student.Email,
		&student.Password.hash,
		&student.Class,
		&student.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &student, nil
}

func (m StudentModel) Update(student *Student) error {
	query := `UPDATE students
			  SET name = ?, email = ?, password = ?, class = ?
			  WHERE school_id = ?`

	args := []interface{}{
		student.Name,
		student.Email,
		student.Password.hash,
		student.Class,
		student.SchoolID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "student_email_key" {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrEditConflict
	}

	return nil
}

func (m StudentModel) GetForToken(tokenScope, tokenPlaintext string) (*Student, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
SELECT student.id,student.school_id, student.created_at, student.name, student.email, student.password, student.class FROM students
INNER JOIN tokens
ON student.id = tokens.student_id
WHERE tokens.hash = ?
AND tokens.scope = ?
AND tokens.expiry > ?`

	args := []interface{}{tokenHash[:], tokenScope, time.Now()}
	var student Student
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Execute the query, scanning the return values into a student struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&student.ID,
		&student.SchoolID,
		&student.CreatedAt,
		&student.Name,
		&student.Email,
		&student.Password.hash,
		&student.Class,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Return the matching student.
	return &student, nil
}
