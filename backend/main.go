package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	// Load .env file for local development.
	// On Render, environment variables are set in the dashboard instead,
	// so it's fine if this fails there — we just log it, not crash.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// Load the Aiven CA certificate.
	// In production (Render), it comes from an env var since there's no
	// local file on the server. Locally, we fall back to the certs/ca.pem file.
	var caCert []byte
	if certEnv := os.Getenv("DB_CA_CERT"); certEnv != "" {
		caCert = []byte(certEnv)
	} else {
		fileCert, ferr := os.ReadFile("certs/ca.pem")
		if ferr != nil {
			log.Fatalf("failed to read CA certificate: %v", ferr)
		}
		caCert = fileCert
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("failed to add CA certificate to certificate pool")
	}

	if err := mysql.RegisterTLSConfig("aiven", &tls.Config{
		RootCAs: rootCertPool,
	}); err != nil {
		log.Fatalf("failed to register TLS config: %v", err)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?tls=aiven&parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	log.Println("Connected to database successfully")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/enquiries", corsMiddleware(enquiriesRouter))
	http.HandleFunc("/enquiries/", corsMiddleware(enquiryByIDRouter))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func enquiriesRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createEnquiryHandler(w, r)
	case http.MethodGet:
		listEnquiriesHandler(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func enquiryByIDRouter(w http.ResponseWriter, r *http.Request) {
	// Expected path: /enquiries/{id}/status
	path := strings.TrimPrefix(r.URL.Path, "/enquiries/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 || parts[1] != "status" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	id := parts[0]
	updateStatusHandler(w, r, id)
}