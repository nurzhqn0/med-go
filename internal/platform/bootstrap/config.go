package bootstrap

type Config struct {
	MongoURI            string
	MongoDatabaseName   string
	DoctorAddress       string
	AppointmentAddress  string
	DoctorServiceTarget string
}

func LoadConfig() Config {
	return Config{
		MongoURI:            GetEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDatabaseName:   GetEnv("MONGODB_DATABASE", "med_go"),
		DoctorAddress:       GetEnv("DOCTOR_SERVICE_ADDR", ":8081"),
		AppointmentAddress:  GetEnv("APPOINTMENT_SERVICE_ADDR", ":8082"),
		DoctorServiceTarget: GetEnv("DOCTOR_SERVICE_GRPC_TARGET", "127.0.0.1:8081"),
	}
}
