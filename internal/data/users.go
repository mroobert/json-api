package data

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mroobert/json-api/internal/database"
	"github.com/mroobert/json-api/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

//go:embed queries/users/create.sql
var createUserSQL string

//go:embed queries/users/read.sql
var readUserSQL string

//go:embed queries/users/update.sql
var updateUserSQL string

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type (

	// User represent an individual user.
	User struct {
		ID        int64     `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Password  password  `json:"-"`
		Activated bool      `json:"activated"`
		Version   int       `json:"-"`
	}

	// NewUser contains information needed to create a new user.
	NewUser struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// UserRepository manages the set of APIs for user database access.
	UserRepository struct {
		DB *pgxpool.Pool
	}

	// password contains the plaintext and hashed versions of the password for a user.
	// The plaintext field is a *pointer* to a string, so that we're able to distinguish between
	// a plaintext password not being present in the struct at all, versus a plaintext password
	// which is the empty string "".
	password struct {
		plaintext *string
		hash      []byte
	}
)

func (u User) Validate(v *validator.Validator) {
	v.Check(u.Name != "", "name", "must be provided")
	v.Check(len(u.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, u.Email)

	if u.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *u.Password.plaintext)
	}

	// If the password hash is ever nil, this will be due to a logic error in our
	// codebase (probably because we forgot to set a password for the user). It's a
	// useful sanity check to include here, but it's not a problem with the data
	// provided by the client. So rather than adding an error to the validation map we
	// raise a panic instead.
	if u.Password.hash == nil {
		panic("missing password hash for user")
	}
}

func (u *User) FromNewUser(input NewUser) {
	u.Name = input.Name
	u.Email = input.Email
	u.Activated = false
}

// Create will insert a new user in the database.
func (r UserRepository) Create(user *User) error {
	args := []any{user.Name, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.DB.QueryRow(ctx, createUserSQL, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == database.UniqueViolation {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

// Read will fetch a user from the database.
// Because we have a UNIQUE constraint on the email column, this SQL query will only
// return one record or none at all.
func (r UserRepository) Read(email string) (*User, error) {
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.DB.QueryRow(ctx, readUserSQL, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

// Update will update a user from the database.
// This operation is implementing optimistic locking.
func (r UserRepository) Update(user *User) error {
	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.DB.QueryRow(ctx, updateUserSQL, args...).Scan(&user.Version)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == database.UniqueViolation {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

// Set calculates the bcrypt hash of a plaintext password, and stores both
// the hash and the plaintext versions in the struct.
func (p *password) Set(plainTextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plainTextPassword
	p.hash = hash

	return nil
}

// Matches checks whether the provided plaintext password matches the
// hashed password stored in the struct.
func (p *password) Matches(plainTextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plainTextPassword))
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
