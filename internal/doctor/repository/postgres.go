package repository

import (
	"context"
	"errors"

	"med-go/internal/doctor/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, doctor model.Doctor) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO doctors (id, full_name, specialization, email)
		VALUES ($1, $2, $3, $4)
	`, doctor.ID, doctor.FullName, doctor.Specialization, doctor.Email)
	if isUniqueViolation(err) {
		return ErrDoctorEmailAlreadyExists
	}

	return err
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Doctor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, full_name, specialization, email
		FROM doctors
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doctors := make([]model.Doctor, 0)
	for rows.Next() {
		var doctor model.Doctor
		if err := rows.Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email); err != nil {
			return nil, err
		}
		doctors = append(doctors, doctor)
	}

	return doctors, rows.Err()
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	var doctor model.Doctor
	err := r.pool.QueryRow(ctx, `
		SELECT id, full_name, specialization, email
		FROM doctors
		WHERE id = $1
	`, id).Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Doctor{}, ErrDoctorNotFound
	}
	if err != nil {
		return model.Doctor{}, err
	}

	return doctor, nil
}

func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM doctors WHERE email = $1)
	`, email).Scan(&exists)

	return exists, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
