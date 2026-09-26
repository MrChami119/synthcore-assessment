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
		caCert = normalizePEM(certEnv)
	} else {
		fileCert, ferr := os.ReadFile("certs/ca.pem")
		if ferr != nil {
			log.Fatalf("failed to read CA certificate: %v", ferr)
		}
		caCert = fileCert
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("failed to add CA certificate to certificate pool: DB_CA_CERT is set but is not valid PEM (need -----BEGIN CERTIFICATE----- with real newlines, not literal \\n)")
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

// normalizePEM makes CA certs stored in environment variables parseable.
// Hosting dashboards often keep literal \n sequences, wrap the value in quotes,
// or paste the PEM as a single line.
func normalizePEM(cert string) []byte {
	cert = strings.TrimSpace(cert)
	if len(cert) >= 2 {
		if (cert[0] == '"' && cert[len(cert)-1] == '"') || (cert[0] == '\'' && cert[len(cert)-1] == '\'') {
			cert = strings.TrimSpace(cert[1 : len(cert)-1])
		}
	}

	cert = strings.ReplaceAll(cert, `\r\n`, "\n")
	cert = strings.ReplaceAll(cert, `\n`, "\n")
	cert = strings.ReplaceAll(cert, "\r\n", "\n")

	const begin = "-----BEGIN CERTIFICATE-----"
	const end = "-----END CERTIFICATE-----"
	if strings.Contains(cert, begin) && !strings.Contains(cert, "\n") {
		var rebuilt strings.Builder
		rest := cert
		for {
			start := strings.Index(rest, begin)
			if start < 0 {
				break
			}
			rest = rest[start+len(begin):]
			endIdx := strings.Index(rest, end)
			if endIdx < 0 {
				break
			}
			body := strings.ReplaceAll(strings.TrimSpace(rest[:endIdx]), " ", "")
			rebuilt.WriteString(begin)
			rebuilt.WriteByte('\n')
			rebuilt.WriteString(body)
			rebuilt.WriteByte('\n')
			rebuilt.WriteString(end)
			rebuilt.WriteByte('\n')
			rest = rest[endIdx+len(end):]
		}
		if rebuilt.Len() > 0 {
			return []byte(rebuilt.String())
		}
	}

	if !strings.HasSuffix(cert, "\n") {
		cert += "\n"
	}
	return []byte(cert)
}