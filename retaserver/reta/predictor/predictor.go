package predictor

import (
	"bytes"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"appengine"

	"reta/errors"
)

type Predictor struct {
	beginDate                 time.Time
	endDate                   time.Time
	trainingDatasetPercentage int
	testingDatasetPercentage  int
	iteration                 int
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

func (p *Predictor) SetIteration(num int) {
	p.iteration = num
}

//1. Get all user data from begin to end dates
//2. Slice it using percentage
//3. Use training data to create model using prediction method
//4. Use testing data to test prediction
//5. Return model and prediction as HTML
func (p *Predictor) RunPrediction(w http.ResponseWriter, c appengine.Context) string {
	//Initialize HTML result string
	var buffer bytes.Buffer

	//Header
	buffer.WriteString("<header>")
	buffer.WriteString("<h2>Logistic Regression Model for Day-1 Retention</h2>")
	buffer.WriteString("<span>Model created from ")
	buffer.WriteString(p.beginDate.String())
	buffer.WriteString(" to ")
	buffer.WriteString(p.endDate.String())
	buffer.WriteString("</span></header>")

	//Get playerinfo
	var playerinfos []PlayerInfo
	retented, err := GetPlayerInformation(c, p.beginDate, p.endDate, &playerinfos)
	if err != nil {
		return err.Error()
	}

	c.Debugf("Total Retented Player:\n%v\n", retented)

	//Create regression instance
	var regress Regression
	regress.EnableDebugMode(c)

	//Init
	regress.Initialize(6)
	//regress.Initialize(2)

	//Set variable names
	regress.SetObservedName("Day 1 Retention")
	regress.SetVariableName(0, "Tutorial Momentum")
	regress.SetVariableName(1, "Level Momentum")
	regress.SetVariableName(2, "Gameplay Consumed")
	regress.SetVariableName(3, "Social Activity")
	regress.SetVariableName(4, "Progression")
	regress.SetVariableName(5, "Level")
	//regress.SetVariableName(0, "Tutorial Momentum")
	//regress.SetVariableName(1, "Gameplay Consumed")

	//Calculate number of data
	totalDataset := len(playerinfos)

	//Shuffle so we get good result
	trainingRetented := int(float64(p.trainingDatasetPercentage) / 100.0 * float64(retented))
	trainingNotRetented := int(float64(p.trainingDatasetPercentage) / 100.0 * float64(totalDataset-retented))
	testRetented := retented - trainingRetented
	testNotRetented := (totalDataset - retented) - trainingNotRetented

	c.Debugf("Training Retented:\n%v\n", trainingRetented)
	c.Debugf("Training Not-Retented:\n%v\n", trainingNotRetented)
	c.Debugf("Testing Retented:\n%v\n", testRetented)
	c.Debugf("Testing Not-Retented:\n%v\n", testNotRetented)

	trainingInfos := make([]PlayerInfo, trainingRetented+trainingNotRetented)
	testInfos := make([]PlayerInfo, testRetented+testNotRetented)

	trainingIndex := 0
	testIndex := 0

	for i := 0; i < totalDataset; i++ {
		if playerinfos[i].Day1Retention {
			if trainingRetented > 0 {
				trainingInfos[trainingIndex] = playerinfos[i]
				trainingIndex += 1
				trainingRetented -= 1
			} else {
				testInfos[testIndex] = playerinfos[i]
				testIndex += 1
				testRetented -= 1
			}
		} else {
			if trainingNotRetented > 0 {
				trainingInfos[trainingIndex] = playerinfos[i]
				trainingIndex += 1
				trainingNotRetented -= 1
			} else {
				testInfos[testIndex] = playerinfos[i]
				testIndex += 1
				testNotRetented -= 1
			}
		}
	}

	for i := range trainingInfos {
		j := rand.Intn(i + 1)
		trainingInfos[i], trainingInfos[j] = trainingInfos[j], trainingInfos[i]
	}

	for i := range testInfos {
		j := rand.Intn(i + 1)
		testInfos[i], testInfos[j] = testInfos[j], testInfos[i]
	}

	trainingDataNum := len(trainingInfos)
	testDataNum := len(testInfos)

	c.Debugf("Total Dataset:\n%v\n", totalDataset)
	c.Debugf("Training Data:\n%v\n", trainingDataNum)
	c.Debugf("Testing Data:\n%v\n", testDataNum)

	//Dataset
	buffer.WriteString("<div>Total Dataset: ")
	buffer.WriteString(strconv.FormatInt(int64(totalDataset), 10))
	buffer.WriteString("</div>")
	buffer.WriteString("<div>Training vs Testing: ")
	buffer.WriteString(strconv.FormatInt(int64(trainingDataNum), 10))
	buffer.WriteString(" vs ")
	buffer.WriteString(strconv.FormatInt(int64(testDataNum), 10))
	buffer.WriteString("</div>")
	buffer.WriteString("<br/>")

	//Add training data
	for i := 0; i < trainingDataNum; i++ {
		//Convert retention to float
		var retented float64
		if trainingInfos[i].Day1Retention {
			retented = 1.0
		} else {
			retented = 0.0
		}

		//Convert metric to float
		tutorialMomentum := trainingInfos[i].TutorialMomentum
		levelMomentum := trainingInfos[i].LevelMomentum
		gameplayConsumed := float64(trainingInfos[i].GameplayConsumed)
		socialActivity := float64(trainingInfos[i].SocialActivities)
		progression := trainingInfos[i].Progression
		level := float64(trainingInfos[i].Level)

		//Create and add datapoints
		datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, levelMomentum, gameplayConsumed, socialActivity, progression, level}}
		//datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, gameplayConsumed}}
		err = regress.AddDataPoint(datapoint)
		if err != nil {
			return err.Error()
		}
	}

	//Generate logistic regression model
	err = regress.GenerateModel(p.iteration)
	if err != nil {
		return err.Error()
	}

	//Keep generated model
	model := regress.StringHTML()
	buffer.WriteString(model)

	//Add testing data
	testDatapoint := make([]DataPoint, testDataNum)
	for i := 0; i < testDataNum; i++ {
		//Convert retention to float
		var retented float64
		if testInfos[i].Day1Retention {
			retented = 1.0
		} else {
			retented = 0.0
		}

		//Convert metric to float
		tutorialMomentum := testInfos[i].TutorialMomentum
		levelMomentum := testInfos[i].LevelMomentum
		gameplayConsumed := float64(testInfos[i].GameplayConsumed)
		socialActivity := float64(testInfos[i].SocialActivities)
		progression := testInfos[i].Progression
		level := float64(testInfos[i].Level)

		//Create and add datapoints
		datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, levelMomentum, gameplayConsumed, socialActivity, progression, level}}
		//datapoint := DataPoint{Result: retented, Variables: []float64{tutorialMomentum, gameplayConsumed}}
		testDatapoint[i] = datapoint
	}

	//Test prediction
	var prediction float64
	prediction, err = regress.TestModel(testDatapoint)
	if err != nil {
		return err.Error()
	}

	buffer.WriteString("<br/><div><h3>Prediction result percentage (cross-validation with testing data): ")
	buffer.WriteString(strconv.FormatFloat(prediction, 'f', 2, 64))
	buffer.WriteString(" </div>")

	return buffer.String()
}
