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

type doctorDocument struct {
	ID             string `bson:"_id"`
	FullName       string `bson:"full_name"`
	Specialization string `bson:"specialization"`
	Email          string `bson:"email"`
}

func NewMongoRepository(ctx context.Context, database *mongo.Database) (*MongoRepository, error) {
	repo := &MongoRepository{
		collection: database.Collection("doctors"),
	}

	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *MongoRepository) Create(ctx context.Context, doctor model.Doctor) error {
	_, err := r.collection.InsertOne(ctx, doctorToDocument(doctor))
	if mongo.IsDuplicateKeyError(err) {
		return ErrDoctorEmailAlreadyExists
	}

	return err
}

func (r *MongoRepository) List(ctx context.Context) ([]model.Doctor, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []doctorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	doctors := make([]model.Doctor, 0, len(documents))
	for _, document := range documents {
		doctors = append(doctors, documentToDoctor(document))
	}

	return doctors, nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	var document doctorDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Doctor{}, ErrDoctorNotFound
		}

		return model.Doctor{}, err
	}

	return documentToDoctor(document), nil
}

func (r *MongoRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	err := r.collection.FindOne(ctx, bson.D{{Key: "email", Value: email}}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}

	return false, err
}

func (r *MongoRepository) ensureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_doctor_email"),
	})

	return err
}

func doctorToDocument(doctor model.Doctor) doctorDocument {
	return doctorDocument{
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}

func documentToDoctor(document doctorDocument) model.Doctor {
	return model.Doctor{
		ID:             document.ID,
		FullName:       document.FullName,
		Specialization: document.Specialization,
		Email:          document.Email,
	}
}
