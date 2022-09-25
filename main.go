package main

import (
	"encoding/json"
	"log"
	"net/http"

	authentication "k8s.io/api/authentication/v1beta1"
)

var Users []map[string]string = []map[string]string{
	{"token": "wei1-token", "username": "wei2", "uid": "111"},
	{"token": "wei2-token", "username": "wei2", "uid": "222"},
}

func main() {
	http.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		// 00. Decode TokenReview
		var tokenView authentication.TokenReview
		err := decoder.Decode(&tokenView)
		if err != nil {
			log.Fatalln("Error:", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apiVersion": "authentication.k8s.io/v1beta1",
				"kind":       "TokenReview",
				"status": authentication.TokenReviewStatus{
					Authenticated: false,
				},
			})
			return
		}

		// 01. Check user
		for _, item := range Users {
			if item["token"] == tokenView.Spec.Token {
				w.WriteHeader(http.StatusOK)
				trs := authentication.TokenReviewStatus{
					Authenticated: true,
					User: authentication.UserInfo{
						Username: item["username"],
						UID:      item["uid"],
					},
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"apiVersion": "authentication.k8s.io/v1beta1",
					"kind":       "TokenReview",
					"status":     trs,
				})
				log.Println("username:", item["username"])
				return
			}
		}

		// check fail
		log.Println("Error: auth failed")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"apiVersion": "authentication.k8s.io/v1beta1",
			"kind":       "TokenReview",
			"status": authentication.TokenReviewStatus{
				Authenticated: false,
			},
		})
		return
	})
	log.Println("Listen on port: 3000")
	http.ListenAndServe(":3000", nil)
}
