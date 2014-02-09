package reta

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"appengine"

	"reta/db"
	"reta/predictor"
)

func init() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/dataset", datasetHandler)
	http.HandleFunc("/predict", predictHandler)
	http.HandleFunc("/about", aboutHandler)

	http.HandleFunc("/connector", connectorHandler)
}

var homeTemplate = template.Must(template.ParseFiles("reta/templates/index.html"))

func rootHandler(w http.ResponseWriter, r *http.Request) {
	err := homeTemplate.Execute(w, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func datasetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Reta Server\n")
	fmt.Fprintln(w, "Home | [Dataset] | Predict | About\n\n")

	fmt.Fprintln(w, "ERROR_MODULE_UNIMPLEMENTED\n")
}

func predictHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Reta Server\n")
	fmt.Fprintln(w, "Home | Dataset | [Predict] | About\n\n")

	layout := "01/02/2006"
	beginning, _ := time.Parse(layout, "01/01/2013")
	today := time.Now().Format(layout)

	fmt.Fprintln(w, "Model Generation\n")
	fmt.Fprintln(w, "From 01/01/2014 to", today)
	fmt.Fprintln(w, "Training - Test Dataset Percentage: 80% - 20%")
	fmt.Fprintln(w, "Method: Logistic Regression")
	fmt.Fprintln(w, "Technique: Iteratively Reweighted Least Squares | Newton-Raphson")

	fmt.Fprintln(w, "\n[TEST]")

	c := appengine.NewContext(r)

	var predict predictor.Predictor
	predict.SetInputDates(beginning, time.Now())
	predict.SetDatasetPercentage(80, 20)
	predict.SetMethod("Linear Regression", nil)
	predict.RunPrediction(w, c)
}

var aboutTemplate = template.Must(template.ParseFiles("reta/templates/about.html"))

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	err := aboutTemplate.Execute(w, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func connectorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Fprint(w, "YOU_DONT_BELONG_HERE")
		return
	}

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
}
