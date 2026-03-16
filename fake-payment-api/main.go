package main

import (
	"fmt"
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

func sendCallback(paymentID string, status string) error {
	payload := fmt.Sprintf(`{
		"payment_id": "%s",
		"status" :"%s"
	}`, paymentID, status)

	resp, err := http.Post(
		"http://localhost:8080/payments/callback",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

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
