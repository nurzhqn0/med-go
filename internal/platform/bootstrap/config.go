package bootstrap

type Config struct {
	DoctorAddress          string
	AppointmentAddress     string
	DoctorServiceTarget    string
	DatabaseURL            string
	DoctorDatabaseURL      string
	AppointmentDatabaseURL string
	NATSURL                string
}

func LoadConfig() Config {
	databaseURL := GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/med_go?sslmode=disable")

	return Config{
		DoctorAddress:          GetEnv("DOCTOR_SERVICE_ADDR", ":8081"),
		AppointmentAddress:     GetEnv("APPOINTMENT_SERVICE_ADDR", ":8082"),
		DoctorServiceTarget:    GetEnv("DOCTOR_SERVICE_GRPC_TARGET", "127.0.0.1:8081"),
		DatabaseURL:            databaseURL,
		DoctorDatabaseURL:      GetEnv("DOCTOR_DATABASE_URL", databaseURL),
		AppointmentDatabaseURL: GetEnv("APPOINTMENT_DATABASE_URL", databaseURL),
		NATSURL:                GetEnv("NATS_URL", "nats://localhost:4222"),
	}
}
