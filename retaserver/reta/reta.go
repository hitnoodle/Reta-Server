package reta

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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
	//Create appengine context
	c := appengine.NewContext(r)

	//Set dates
	layout := "02/01/2006"
	beginning, _ := time.Parse(layout, r.FormValue("startdate"))
	ending, _ := time.Parse(layout, r.FormValue("enddate"))

	//Set iteration
	iteration, _ := strconv.ParseInt((r.FormValue("iteration")), 10, 32)

	//Run prediction
	var predict predictor.Predictor
	predict.SetInputDates(beginning, ending)
	predict.SetDatasetPercentage(80, 20)
	predict.SetIteration(int(iteration))
	prediction := predict.RunPrediction(w, c)

	//Show prediction result on result page
	err := resultTemplate.Execute(w, template.HTML(prediction))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func oldresultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Reta Server | Prediction Result\n")

	layout := "02/01/2006"
	beginning, _ := time.Parse(layout, "17/02/2014")
	ending, _ := time.Parse(layout, "28/02/2014")

	fmt.Fprintln(w, "Model Generation\n")
	fmt.Fprintln(w, "From 17/02/2014 to 28/02/2014")
	fmt.Fprintln(w, "Training - Test Dataset Percentage: 80% - 20%")
	fmt.Fprintln(w, "Method: Logistic Regression")
	fmt.Fprintln(w, "Technique: Iteratively Reweighted Least Squares | Newton-Raphson")
	fmt.Fprintln(w, "Iteration: 20 times")

	c := appengine.NewContext(r)

	var predict predictor.Predictor
	predict.SetInputDates(beginning, ending)
	predict.SetDatasetPercentage(80, 20)
	predict.SetIteration(20)
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
