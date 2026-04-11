package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/pay", paymentPage)
	mux.HandleFunc("/success", paymentSuccess)
	mux.HandleFunc("/fail", paymentFail)

	fmt.Println("Fake Payment Provider running on :8081")

	err := http.ListenAndServe(":8081", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func GenerateHMAC(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func sendCallback(paymentID string, status string) error {
	payload := fmt.Sprintf(`{
		"payment_id": "%s",
		"status" :"%s"
	}`, paymentID, status)

	// create signature
	sign := GenerateHMAC(payload, "ini-rahasia")

	req, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/payments/callback",
		strings.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", sign)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func paymentSuccess(w http.ResponseWriter, r *http.Request) {
	paymentID := r.FormValue("payment_id")

	err := sendCallback(paymentID, "success")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Payment Success! Callback sent.")
}

func paymentFail(w http.ResponseWriter, r *http.Request) {

	paymentID := r.FormValue("payment_id")

	err := sendCallback(paymentID, "failed")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Payment Failed! Callback sent.")
}

func paymentPage(w http.ResponseWriter, r *http.Request) {

	paymentID := r.URL.Query().Get("payment_id")

	html := fmt.Sprintf(`
		<html>
		<body>
			<h1>Fake Payment Gateway</h1>
			<p>Payment ID: %s</p>

			<form action="/success" method="POST">
				<input type="hidden" name="payment_id" value="%s"/>
				<button type="submit">Pay</button>
			</form>

			<form action="/fail" method="POST">
				<input type="hidden" name="payment_id" value="%s"/>
				<button type="submit">Fail</button>
			</form>
		</body>
		</html>
	`, paymentID, paymentID, paymentID)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
