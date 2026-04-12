package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"med-go/internal/appointment/model"
)

type MongoRepository struct {
	collection *mongo.Collection
}

type appointmentDocument struct {
	ID          string       `bson:"_id"`
	Title       string       `bson:"title"`
	Description string       `bson:"description"`
	DoctorID    string       `bson:"doctor_id"`
	Status      model.Status `bson:"status"`
	CreatedAt   time.Time    `bson:"created_at"`
	UpdatedAt   time.Time    `bson:"updated_at"`
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: database.Collection("appointments"),
	}
}

func (r *MongoRepository) Create(ctx context.Context, appointment model.Appointment) error {
	_, err := r.collection.InsertOne(ctx, appointmentToDocument(appointment))
	return err
}

func (r *MongoRepository) List(ctx context.Context) ([]model.Appointment, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []appointmentDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	appointments := make([]model.Appointment, 0, len(documents))
	for _, document := range documents {
		appointments = append(appointments, documentToAppointment(document))
	}

	return appointments, nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	var document appointmentDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Appointment{}, ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	return documentToAppointment(document), nil
}

func (r *MongoRepository) Update(ctx context.Context, appointment model.Appointment) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: appointment.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "title", Value: appointment.Title},
			{Key: "description", Value: appointment.Description},
			{Key: "doctor_id", Value: appointment.DoctorID},
			{Key: "status", Value: appointment.Status},
			{Key: "created_at", Value: appointment.CreatedAt},
			{Key: "updated_at", Value: appointment.UpdatedAt},
		}}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrAppointmentNotFound
	}

	return nil
}

func appointmentToDocument(appointment model.Appointment) appointmentDocument {
	return appointmentDocument{
		ID:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorID:    appointment.DoctorID,
		Status:      appointment.Status,
		CreatedAt:   appointment.CreatedAt,
		UpdatedAt:   appointment.UpdatedAt,
	}
}

func documentToAppointment(document appointmentDocument) model.Appointment {
	return model.Appointment{
		ID:          document.ID,
		Title:       document.Title,
		Description: document.Description,
		DoctorID:    document.DoctorID,
		Status:      document.Status,
		CreatedAt:   document.CreatedAt,
		UpdatedAt:   document.UpdatedAt,
	}
}
