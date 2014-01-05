package reta

import (
	"fmt"
	"net/http"

	"appengine"

	"reta/db"
)

func init() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/connector", connectorHandler)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Reta Server | ONLINE\n")
	fmt.Fprintln(w, "Latest events\n")

	c := appengine.NewContext(r)
	db.ListActivities(w, c, 15)
}

func connectorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		c := appengine.NewContext(r)
		formData := r.PostForm

		err = db.SubmitActivity(c, formData.Get("userid"), formData.Get("appversion"), formData.Get("data"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		fmt.Fprint(w, "DATA_SENT_SUCCESS")
	} else {
		fmt.Fprint(w, "Reta Server | Connector Module | ONLINE")
	}
}
