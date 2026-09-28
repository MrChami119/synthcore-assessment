package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var resendClient = &http.Client{Timeout: 10 * time.Second}

// sendEnquiryNotification emails the admin about a new enquiry via Resend.
// It is best-effort: any failure is logged, never returned, so a broken
// email service can't stop an enquiry from being saved.
func sendEnquiryNotification(e Enquiry) {
	apiKey := os.Getenv("RESEND_API_KEY")
	to := os.Getenv("NOTIFY_EMAIL")
	if apiKey == "" || to == "" {
		log.Println("email notification skipped: RESEND_API_KEY or NOTIFY_EMAIL not set")
		return
	}

	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "Synthcore Enquiries <onboarding@resend.dev>"
	}

	company := e.Company
	if company == "" {
		company = "(not provided)"
	}

	// Everything the visitor typed is untrusted, so escape it before it goes into HTML.
	message := strings.ReplaceAll(html.EscapeString(e.Message), "\n", "<br>")
	body := fmt.Sprintf(
		"<h2>New enquiry #%d</h2>"+
			"<p><strong>Name:</strong> %s</p>"+
			"<p><strong>Email:</strong> %s</p>"+
			"<p><strong>Company:</strong> %s</p>"+
			"<p><strong>Message:</strong></p><p>%s</p>",
		e.ID,
		html.EscapeString(e.Name),
		html.EscapeString(e.Email),
		html.EscapeString(company),
		message,
	)

	// Strip line breaks from the name so it can't mangle the subject line.
	safeName := strings.NewReplacer("\r", " ", "\n", " ").Replace(e.Name)

	payload, err := json.Marshal(map[string]any{
		"from":     from,
		"to":       []string{to},
		"subject":  "New enquiry from " + safeName,
		"html":     body,
		"reply_to": e.Email,
	})
	if err != nil {
		log.Printf("email notification: failed to build payload: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		log.Printf("email notification: failed to build request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// Stops a retry from sending the same email twice.
	req.Header.Set("Idempotency-Key", fmt.Sprintf("enquiry-%d", e.ID))

	resp, err := resendClient.Do(req)
	if err != nil {
		log.Printf("email notification: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("email notification: Resend returned %d: %s", resp.StatusCode, respBody)
		return
	}
	log.Printf("email notification sent for enquiry #%d", e.ID)
}