package reta

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/connector", connectorHandler)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Reta Server | ONLINE")
}

func connectorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if (err == nil) {
			formData := r.PostForm
			fmt.Fprint(w, formData)
		}
	} else {
		fmt.Fprint(w, "Reta Server | Connector Module | ONLINE")
	}
}