package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"med-go/internal/appointment/model"
)

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: database.Collection("appointments"),
	}
}

func (r *MongoRepository) Create(ctx context.Context, appointment model.Appointment) error {
	_, err := r.collection.InsertOne(ctx, appointment)
	return err
}

func (r *MongoRepository) List(ctx context.Context) ([]model.Appointment, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var appointments []model.Appointment
	if err := cursor.All(ctx, &appointments); err != nil {
		return nil, err
	}

	return appointments, nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	var appointment model.Appointment
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&appointment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Appointment{}, ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	return appointment, nil
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
