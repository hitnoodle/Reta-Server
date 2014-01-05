package predictor

import (
	"bytes"
)

//Logistic regression model
//
//References:
// - https://github.com/sajari/regression
// - http://crsouza.blogspot.com/2010/02/logistic-regression-in-c.html

type DataPoint struct {
	Result    float64
	Variables []float64
}

type Model struct {
	Coefficients             []float64
	StandardErrors           []float64
	WaldTests                []float64
	OddsRatio                []float64
	LowerConfidenceIntervals []float64
	UpperConfidenceIntervals []float64
	LogLikelihood            float64
	Deviance                 float64
	ChiSquare                float64
}

type Regression struct {
	inputs        int         //How many coefficient are there
	observedName  string      //Name of the observed variable
	variableNames []string    //Name of each independent variables
	dataPoints    []DataPoint //Datapoints used for training
	model         Model       //Regression model from training
}

func (r *Regression) Initialize(input int) {
	r.inputs = input
	r.variableNames = make([]string, r.inputs)
}

func (r *Regression) SetObservedName(observed string) {
	r.observedName = observed
}

func (r *Regression) SetVariableName(index int, variable string) {
	if index > -1 && index < len(r.variableNames) {
		r.variableNames[index] = variable
	}
}

func (r *Regression) AddDataPoint(data DataPoint) {
	r.dataPoints = append(r.dataPoints, data)
}

func (r *Regression) GenerateModel() {

}

func (r *Regression) PrintModel() string {
	var buffer bytes.Buffer
	buffer.WriteString("Name\t\t\tCoefficient\tStd. Error\tp-Value\t\tOdds Ratio\tLower Confidence\tUpper Confidence\n")
	for _, variable := range r.variableNames {
		buffer.WriteString("\n")
		buffer.WriteString(variable)
		buffer.WriteString("\n")
	}

	buffer.WriteString("\n")
	buffer.WriteString("Log Likelihood:\n")
	buffer.WriteString("-2 * Log Likelihood (Deviance):\n")
	buffer.WriteString("Chi-Square Goodness of Fit:\t\t\t")
	buffer.WriteString("P-Value:\n")

	return buffer.String()
}

func (r *Regression) PredictInput(input []float64) float64 {
	return float64(0)
}

func (r *Regression) PredictInputs(inputs *[]DataPoint) float64 {
	return float64(0)
}
