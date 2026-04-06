package model

type Doctor struct {
	ID             string `json:"id" bson:"_id"`
	FullName       string `json:"full_name" bson:"full_name"`
	Specialization string `json:"specialization" bson:"specialization"`
	Email          string `json:"email" bson:"email"`
}
