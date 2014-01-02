package reta

import (
	"fmt"
	"net/http"

	"appengine"
	"appengine/datastore"
)

type Activity struct {
	Player  string
	Version string
	Data    string
}

func init() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/connector", connectorHandler)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	fmt.Fprintln(w, "Reta Server | ONLINE\n")
	fmt.Fprintln(w, "Latest activity\n")

	q := datastore.NewQuery("Activity").Limit(15)
	for t := q.Run(c); ; {
		var act Activity

		_, err := t.Next(&act)
		if err == datastore.Done {
			break
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Activity=%#v\n\n", act)
	}
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
		activity := Activity{
			Player:  formData.Get("userid"),
			Version: formData.Get("appversion"),
			Data:    formData.Get("data"),
		}

		_, err = datastore.Put(c, datastore.NewIncompleteKey(c, "Activity", nil), &activity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, "DATA_SENT_SUCCESS")
	} else {
		fmt.Fprint(w, "Reta Server | Connector Module | ONLINE")
	}
}
