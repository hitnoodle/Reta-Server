package predictor

import (
	"fmt"
	"net/http"
	"time"

	"appengine"

	"reta/errors"
)

type Predictor struct {
	beginDate                 time.Time
	endDate                   time.Time
	trainingDatasetPercentage int
	testingDatasetPercentage  int
	predictionMethod          PredictionMethod
}

type PredictionMethod struct {
	Method     string
	Parameters []string
}

func (p *Predictor) SetInputDates(begin time.Time, end time.Time) {
	p.beginDate = begin
	p.endDate = end
}

func (p *Predictor) SetDatasetPercentage(training int, testing int) error {
	if training+testing == 100 {
		p.trainingDatasetPercentage = training
		p.testingDatasetPercentage = testing
	} else {
		return errors.New("Sum of training and testing dataset must equal 100")
	}

	return nil
}

func (p *Predictor) SetMethod(method string, parameters []string) {
	p.predictionMethod = PredictionMethod{method, parameters}
}

//1. Get all user data from begin to end dates
//2. Slice it using percentage
//3. Use training data to create model using prediction method
//4. Use testing data to test prediction
func (p *Predictor) RunPrediction(w http.ResponseWriter, c appengine.Context) {
	var regress Regression
	regress.Initialize(6)
	regress.SetObservedName("Day 1 Retention")
	regress.SetVariableName(0, "Tutorial Momentum")
	regress.SetVariableName(1, "Level Momentum")
	regress.SetVariableName(2, "Gameplay Consumed")
	regress.SetVariableName(3, "Social Activity")
	regress.SetVariableName(4, "Progression")
	regress.SetVariableName(5, "Level")

	for i := 0; i < 80; i++ {

	}

	regress.GenerateModel()
	model := regress.PrintModel()

	inputs := make([]DataPoint, 20)
	for i := 0; i < 20; i++ {

	}

	prediction := regress.PredictInputs(&inputs)

	fmt.Fprintf(w, model)
	fmt.Fprintf(w, "\n\nPrediction result percentage (cross-validation with testing data): %.2f", prediction)
}
