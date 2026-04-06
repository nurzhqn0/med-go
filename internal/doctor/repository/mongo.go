package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"med-go/internal/doctor/model"
)

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: database.Collection("doctors"),
	}
}

func (r *MongoRepository) Create(ctx context.Context, doctor model.Doctor) error {
	_, err := r.collection.InsertOne(ctx, doctor)
	return err
}

func (r *MongoRepository) List(ctx context.Context) ([]model.Doctor, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var doctors []model.Doctor
	if err := cursor.All(ctx, &doctors); err != nil {
		return nil, err
	}

	return doctors, nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	var doctor model.Doctor
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doctor)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Doctor{}, ErrDoctorNotFound
		}

		return model.Doctor{}, err
	}

	return doctor, nil
}
