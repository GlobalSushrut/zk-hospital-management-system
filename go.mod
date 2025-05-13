module github.com/telemedicine/zkhealth

go 1.13

require (
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	go.mongodb.org/mongo-driver v1.3.5
)

// Use local module instead of trying to fetch from GitHub
replace github.com/telemedicine/zkhealth => ./
