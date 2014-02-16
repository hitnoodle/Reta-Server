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
	//Handling interaction with people
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/predict", predictHandler)
	http.HandleFunc("/result", resultHandler)

	http.HandleFunc("/oldresult", oldresultHandler)

	//Handling connection with game
	http.HandleFunc("/connector", connectorHandler)
}

/* Home page */

var homeTemplate = template.Must(template.ParseFiles("reta/templates/index.html"))

func rootHandler(w http.ResponseWriter, r *http.Request) {
	err := homeTemplate.Execute(w, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/* Prediction input page */

var predictTemplate = template.Must(template.ParseFiles("reta/templates/predict.html"))

func predictHandler(w http.ResponseWriter, r *http.Request) {
	err := predictTemplate.Execute(w, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/* Prediction result page */

var resultTemplate = template.Must(template.ParseFiles("reta/templates/result.html"))

func resultHandler(w http.ResponseWriter, r *http.Request) {
	err := resultTemplate.Execute(w, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func oldresultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Reta Server | Prediction Result\n")

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

/* Connection module */

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

	err = db.SubmitEvent(c, formData.Get("userid"), formData.Get("appversion"), formData.Get("data"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(w, "DATA_SENT_SUCCESS")
}
