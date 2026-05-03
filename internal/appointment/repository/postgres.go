package repository

import (
	"context"
	"errors"
	"time"

	"med-go/internal/appointment/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, appointment model.Appointment) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO appointments (id, title, description, doctor_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, appointment.ID, appointment.Title, appointment.Description, appointment.DoctorID, appointment.Status, appointment.CreatedAt, appointment.UpdatedAt)

	return err
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Appointment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, description, doctor_id, status, created_at, updated_at
		FROM appointments
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appointments := make([]model.Appointment, 0)
	for rows.Next() {
		appointment, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		appointments = append(appointments, appointment)
	}

	return appointments, rows.Err()
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	appointment, err := scanAppointment(r.pool.QueryRow(ctx, `
		SELECT id, title, description, doctor_id, status, created_at, updated_at
		FROM appointments
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Appointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return model.Appointment{}, err
	}

	return appointment, nil
}

func (r *PostgresRepository) Update(ctx context.Context, appointment model.Appointment) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE appointments
		SET title = $2,
			description = $3,
			doctor_id = $4,
			status = $5,
			created_at = $6,
			updated_at = $7
		WHERE id = $1
	`, appointment.ID, appointment.Title, appointment.Description, appointment.DoctorID, appointment.Status, appointment.CreatedAt, appointment.UpdatedAt)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrAppointmentNotFound
	}

	return nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status model.Status, updatedAt time.Time) (model.Appointment, model.Status, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Appointment{}, "", err
	}
	defer tx.Rollback(ctx)

	appointment, err := scanAppointment(tx.QueryRow(ctx, `
		SELECT id, title, description, doctor_id, status, created_at, updated_at
		FROM appointments
		WHERE id = $1
		FOR UPDATE
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Appointment{}, "", ErrAppointmentNotFound
	}
	if err != nil {
		return model.Appointment{}, "", err
	}

	oldStatus := appointment.Status
	appointment.Status = status
	appointment.UpdatedAt = updatedAt

	_, err = tx.Exec(ctx, `
		UPDATE appointments
		SET status = $2,
			updated_at = $3
		WHERE id = $1
	`, id, appointment.Status, appointment.UpdatedAt)
	if err != nil {
		return model.Appointment{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Appointment{}, "", err
	}

	return appointment, oldStatus, nil
}

type appointmentScanner interface {
	Scan(dest ...any) error
}

func scanAppointment(scanner appointmentScanner) (model.Appointment, error) {
	var appointment model.Appointment
	if err := scanner.Scan(
		&appointment.ID,
		&appointment.Title,
		&appointment.Description,
		&appointment.DoctorID,
		&appointment.Status,
		&appointment.CreatedAt,
		&appointment.UpdatedAt,
	); err != nil {
		return model.Appointment{}, err
	}

	return appointment, nil
}
