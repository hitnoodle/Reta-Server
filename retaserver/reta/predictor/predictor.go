package predictor

import (
	"fmt"
	"math"
	"math/rand"
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
		return errors.New("Error: Sum of training and testing dataset must equal 100")
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
	//Create regression instance
	var regress Regression
	regress.EnableDebugMode(c)

	regress.Initialize(6)
	regress.SetObservedName("Day 1 Retention")
	regress.SetVariableName(0, "Tutorial Momentum")
	regress.SetVariableName(1, "Level Momentum")
	regress.SetVariableName(2, "Gameplay Consumed")
	regress.SetVariableName(3, "Social Activity")
	regress.SetVariableName(4, "Progression")
	regress.SetVariableName(5, "Level")

	//Hardcode coefficients for testing regression
	testC := []float64{-3.4, 3.2, -5.2, -10.0, -2.3, 4.1, 4.0}
	totalDataset := 100.0
	random := rand.New(rand.NewSource(123))

	fmt.Fprintf(w, "\nHardcoded Test Function:%v", testC)
	fmt.Fprintf(w, "\nTotal Dataset: %v\n", totalDataset)

	trainingDataNum := int(0.8 * totalDataset)
	for i := 0; i < trainingDataNum; i++ {
		//Randomize training data
		intercept := 1.0
		tutorialMomentum := random.Float64() * 3     //Between 0 - 3 minutes per tutorial
		levelMomentum := random.Float64() * 10       //Between 0 - 10 minutes per level
		gameplayConsumed := float64(random.Intn(50)) //Between 0 - 50
		socialActivity := float64(random.Intn(20))   //Between 0 - 20
		progression := random.Float64() * 80         //Between 0 - 80% in the first day
		level := float64(random.Intn(20))            //Between Level 0 - 20 in the first day

		//Calculate training results with hardcode functions above
		z := (testC[0] * intercept) + (testC[1] * tutorialMomentum) + (testC[2] * levelMomentum) + (testC[3] * gameplayConsumed) + (testC[4] * socialActivity) + (testC[5] * progression) + (testC[6] * level)
		p := 1.0 / (1.0 + math.Exp(-z))

		var retented float64
		if p < 0.5 {
			retented = 0.0
		} else {
			retented = 1.0
		}

		//Create and add datapoints
		datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, levelMomentum, gameplayConsumed, socialActivity, progression, level}}
		regress.AddDataPoint(datapoint)
	}

	err := regress.GenerateModel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	//Print generated model
	model := regress.String()
	fmt.Fprintln(w, "\n[TEST RESULT]")
	fmt.Fprintf(w, "\n%s", model)

	//Randomize testing data
	testDataNum := int(0.2 * totalDataset)
	testDatapoint := make([]DataPoint, testDataNum)
	for i := 0; i < testDataNum; i++ {
		//Randomize training data
		intercept := 1.0
		tutorialMomentum := random.Float64()         //Between 0 - 1 minutes per tutorial
		levelMomentum := random.Float64() * 10       //Between 0 - 10 minutes per level
		gameplayConsumed := float64(random.Intn(50)) //Between 0 - 50
		socialActivity := float64(random.Intn(20))   //Between 0 - 20
		progression := random.Float64() * 80         //Between 0 - 80% in the first day
		level := float64(random.Intn(20))            //Between Level 0 - 20 in the first day

		//Calculate training results with hardcode functions above
		z := (testC[0] * intercept) + (testC[1] * tutorialMomentum) + (testC[2] * levelMomentum) + (testC[3] * gameplayConsumed) + (testC[4] * socialActivity) + (testC[5] * progression) + (testC[6] * level)
		p := 1.0 / (1.0 - math.Exp(-z))

		var retented float64
		if p < 0.5 {
			retented = 0.0
		} else {
			retented = 1.0
		}

		//Create and add datapoints
		datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, levelMomentum, gameplayConsumed, socialActivity, progression, level}}
		testDatapoint[i] = datapoint
	}

	var prediction float64
	prediction, err = regress.TestModel(testDatapoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "\n\nPrediction result percentage (cross-validation with testing data): %.2f", prediction)
}
