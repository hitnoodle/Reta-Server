package predictor

import (
	"bytes"
	"github.com/skelterjohn/go.matrix"
	"math"
	"strconv"

	"appengine"

	"reta/errors"
)

//Logistic regression model
//
//References:
// - https://github.com/sajari/regression
// - http://crsouza.blogspot.com/2010/02/logistic-regression-in-c.html
// - http://msdn.microsoft.com/en-us/magazine/jj618304.aspx

type DataPoint struct {
	Result    float64
	Variables []float64
}

type Model struct {
	Coefficients             []float64
	StandardErrors           []float64
	WaldStatistics           []float64
	OddsRatio                []float64
	LowerConfidenceIntervals []float64
	UpperConfidenceIntervals []float64
	LogLikelihood            float64
	Deviance                 float64
	ChiSquare                float64
}

type Regression struct {
	initialized   bool        //Does the regression instance ready to generate model
	inputs        int         //How many coefficient are there
	observedName  string      //Name of the observed variable
	variableNames []string    //Name of each independent variables
	dataPoints    []DataPoint //Datapoints used for training
	model         Model       //Regression model from training

	debugMode    bool
	debugContext appengine.Context
}

func (r *Regression) Initialize(input int) {
	r.inputs = input
	r.variableNames = make([]string, r.inputs)
}

func (r *Regression) EnableDebugMode(c appengine.Context) {
	r.debugMode = true
	r.debugContext = c
}

func (r *Regression) SetObservedName(observed string) {
	r.observedName = observed
}

func (r *Regression) SetVariableName(index int, variable string) {
	if index > -1 && index < len(r.variableNames) {
		r.variableNames[index] = variable
	}
}

func (r *Regression) AddDataPoint(data DataPoint) error {
	numVariables := len(r.variableNames)
	if len(data.Variables) != numVariables {
		return errors.New("Error: Number of variables in the data != in the model")
	}

	r.dataPoints = append(r.dataPoints, data)
	r.initialized = true

	return nil
}

func (r *Regression) GenerateModel(iteration int) error {
	if !r.initialized {
		return errors.New("Error: Need some data to perform regression")
	}

	numData := len(r.dataPoints)
	numVariables := len(r.variableNames)

	if numData <= numVariables {
		return errors.New("Error: Datapoints must exceed variables")
	}

	//Create training data matrix for observed (result) and (independent) variables
	trainingObserved := matrix.Zeros(numData, 1)
	trainingVariables := matrix.Zeros(numData, numVariables+1)

	//Copy data to matrix
	for i := 0; i < numData; i++ {
		trainingObserved.Set(i, 0, r.dataPoints[i].Result)
		for j := 0; j < numVariables+1; j++ {
			if j == 0 {
				trainingVariables.Set(i, 0, 1)
			} else {
				trainingVariables.Set(i, j, r.dataPoints[i].Variables[j-1])
			}
		}
	}

	if r.debugMode {
		r.debugContext.Infof("\n---------- VARIABLES ---------\n%s", trainingVariables.String())
		r.debugContext.Infof("\n-------- OBSERVED -------\n%s", trainingObserved.String())
	}

	//Initialize model arrays
	r.model.StandardErrors = make([]float64, numVariables+1)

	//Newton-Raphson algorithm stop condition
	maxIteration := iteration
	epsilon := 0.01      //Stop if all coefficients change less than this | Algorithm has converged
	jumpFactor := 1000.0 //Stop if any new coefficients jumps too much | Algorithm spinning out of control

	//Use Newton-Raphson to find coefficients that best fit training data
	err := r.computeBestCoefficients(trainingVariables, trainingObserved, maxIteration, epsilon, jumpFactor)
	if err != nil {
		return err
	}

	//Compute odds ratio of generated coefficients
	err = r.computeOddsRatio()
	if err != nil {
		return err
	}

	//Compute pValue of wald statistics from the generated coefficients
	err = r.computeWaldStatistic()
	if err != nil {
		return err
	}

	//Compute confidence interval of generated coefficients
	err = r.computeConfidenceInterval()
	if err != nil {
		return err
	}

	//Compute log likelihood of the model
	err = r.computeLogLikelihood()
	if err != nil {
		return err
	}

	//Compute deviance of the model
	r.computeDeviance()

	//Compute chi-square value
	r.computeChiSquare()

	return nil
}

//Use the Newton-Raphson technique to estimate logistic regression beta parameters: b[t] = b[t-1] + inv(X'W[t-1]X)X'(Y - p[t-1])
//- xTrainingVector is a design matrix of predictor variables where the first column is augmented with all 1.0 to represent dummy x values for the b0 constant
//- yTrainingVector is a column vector of binary (0.0 or 1.0) dependent variables
//- maxIterations is the maximum number of times to iterate in the algorithm. A value of 1000 is reasonable.
//- epsilon is a closeness parameter: if all new b[i] values after an iteration are within epsilon of
// 	the old b[i] values, we assume the algorithm has converged and we return. A value like 0.001 is often reasonable.
//- jumpFactor stops the algorithm if any new beta value is jumpFactor times greater than the old value. A value of 1000.0 seems reasonable.
//
//The return is a column vector of the beta estimates: b[0] is the constant, b[1] for x1, etc.
//
//Note: There is a lot that can go wrong here. The algorithm involves finding a matrx inverse (see MatrixInverse) which will throw
//if the inverse cannot be computed. The Newton-Raphson algorithm can generate beta values that tend towards infinity.
//If anything bad happens the return is the best beta values known at the time (which could be all 0.0 values but not null).
func (r *Regression) computeBestCoefficients(xTrainingVector matrix.Matrix, yTrainingVector matrix.Matrix, maxIteration int, epsilon float64, jumpFactor float64) error {
	//Error checking for the matrix length first
	xRows := xTrainingVector.Rows()
	xCols := xTrainingVector.Cols()

	if xRows != yTrainingVector.Rows() {
		return errors.New("Error: Training vectors are not compatible to generate model")
	}

	//Initial coefficients
	coeffVector := matrix.Zeros(xCols, 1)
	for i := 0; i < xCols; i++ {
		coeffVector.Set(i, 0, 0.0)
	}

	if r.debugMode {
		r.debugContext.Infof("\nInitial coefficients vector:\n%s", coeffVector.String())
	}

	//Current best coefficients
	var bestCoeffVector matrix.Matrix
	bestCoeffVector = coeffVector.Copy()

	//A column vector of the probabilities of each row using the b[i] values and the x[i] values
	pVector, err := r.constructProbVector(xTrainingVector, coeffVector)
	if err != nil {
		return err
	}

	if r.debugMode {
		r.debugContext.Infof("\nInitial probabilities vector:\n%s", pVector.String())
	}

	//Check MSE of the prediction
	mse, err := r.meanSquaredError(pVector, yTrainingVector)
	if err != nil {
		return err
	}

	if r.debugMode {
		r.debugContext.Infof("\nInitial MSE:\n%f", mse)
	}

	//How many times are the new betas worse (i.e., give worse MSE) than the current betas
	timesWorse := 0
	for i := 0; i < maxIteration; i++ {
		//Generate new beta values using Newton-Raphson. Could return null
		newCoeffVector := r.constructNewCoefficientsVector(coeffVector, xTrainingVector, yTrainingVector, pVector)
		if newCoeffVector == nil {
			if r.debugMode {
				r.debugContext.Infof("\nNew coefficients vector is null | current product cannot be inverted -- stopping")
			}
			break
		}

		if r.debugMode {
			r.debugContext.Infof("\nNew calculated coefficients vector:\n%s", newCoeffVector.String())
		}

		//We are done because of no significant change in beta[]
		if r.noChange(coeffVector, newCoeffVector, epsilon) == true {
			if r.debugMode {
				r.debugContext.Infof("\nNo significant change between old beta values and new beta values -- stopping")
			}
			break
		}

		//Any new beta more than jumpFactor times greater than old?
		if r.outOfControl(coeffVector, newCoeffVector, jumpFactor) == true {
			if r.debugMode {
				r.debugContext.Infof("\nThe new coefficients vector has at least one value which changed by a factor of %s -- stopping", jumpFactor)
			}
			break
		}

		pVector, err = r.constructProbVector(xTrainingVector, newCoeffVector)
		if err != nil {
			return err
		}

		if r.debugMode {
			r.debugContext.Infof("\nNew calculated probabilities vector:\n%s", pVector.String())
		}

		newMSE, err := r.meanSquaredError(pVector, yTrainingVector)
		if err != nil {
			return err
		}

		if r.debugMode {
			r.debugContext.Infof("\nNew calculated MSE:\n%f", newMSE)
		}

		if newMSE > mse {
			//Update counter if newMSE is worst than the current one
			timesWorse += 1
			if timesWorse > 4 {
				if r.debugMode {
					r.debugContext.Infof("\nThe new coefficients vector produced worse predictions even after modification four times in a row -- stopping")
				}
				break
			}

			if r.debugMode {
				r.debugContext.Infof("\nThe new coefficients vector has produced probabilities which give worse predictions -- modifying new coefficients to halfway between old and new")
			}

			//Update current: old becomes not the new but halfway between new and old
			for k := 0; k < coeffVector.Rows(); k++ {
				val := coeffVector.Get(k, 0)
				newVal := newCoeffVector.Get(k, 0)
				coeffVector.Set(k, 0, (val+newVal)/2.0)
			}

			//Update current SSD
			mse = newMSE
		} else {
			if r.debugMode {
				r.debugContext.Infof("\nThe new coefficients vector has produced probabilities which give better predictions -- updating")
			}

			coeffVector = newCoeffVector.DenseMatrix() //Update best
			bestCoeffVector = coeffVector.Copy()       //Update current
			mse = newMSE                               //Update current MSE
			timesWorse = 0                             //Reset counter
		}

		if r.debugMode && i == maxIteration-1 {
			r.debugContext.Infof("\nExceeded max iterations -- stopping")
		}
	}

	if r.debugMode {
		r.debugContext.Infof("\nBest coefficients vector:\n%s", bestCoeffVector.String())
	}

	//Done, put coefficients data from matrix to arrays
	length := bestCoeffVector.Rows()
	r.model.Coefficients = make([]float64, length)
	for i := 0; i < length; i++ {
		r.model.Coefficients[i] = bestCoeffVector.Get(i, 0)
	}

	return nil
}

//p = 1 / (1 + exp(-z)) where z = b0x0 + b1x1 + b2x2 + b3x3 + . . .
//Suppose X is 10 x 4 (cols are: x0 = const. 1.0, x1, x2, x3)
//Then b would be a 4 x 1 (col vecror)
//Then result of X times b is (10x4)(4x1) = (10x1) column vector
func (r *Regression) constructProbVector(xMatrix matrix.Matrix, bVector matrix.Matrix) (matrix.Matrix, error) {
	times, err := xMatrix.Times(bVector)
	if err != nil {
		return nil, err
	}

	length := times.Rows()
	for i := 0; i < length; i++ {
		prob := 1.0 / (1.0 + math.Exp(-times.Get(i, 0)))
		times.Set(i, 0, prob)
	}

	return times, nil
}

//How good are the predictions? (using an already-calculated prob vector)
//Note: it is possible that a model with better (lower) MSE than a second model could give worse predictive accuracy.
func (r *Regression) meanSquaredError(pVector matrix.Matrix, yVector matrix.Matrix) (mse float64, err error) {
	pRows := pVector.Rows()
	yRows := yVector.Rows()

	if pRows != yRows {
		err = errors.New("Error: The prob vector and the y vector are not compatible")
		return 0.0, err
	}

	if pRows == 0 {
		return 0.0, nil
	}

	result := 0.0
	for i := 0; i < pRows; i++ {
		result += (pVector.Get(i, 0) - yVector.Get(i, 0)) * (pVector.Get(i, 0) - yVector.Get(i, 0))
	}
	mse = result / float64(pRows)

	return mse, nil
}

//This is the heart of the Newton-Raphson technique
// b[t] = b[t-1] + inv(X'W[t-1]X)X'(y - p[t-1])
//
// b[t] is the new (time t) b column vector
// b[t-1] is the old (time t-1) vector
// X' is the transpose of the X matrix of x data (1.0, age, sex, chol)
// W[t-1] is the old weight matrix
// y is the column vector of binary dependent variable data
// p[t-1] is the old column probability vector (computed as 1.0 / (1.0 + exp(-z) where z = b0x0 + b1x1 + . . .)

//Note: W[t-1] is nxn which could be huge so instead of computing b[t] = b[t-1] + inv(X'W[t-1]X)X'(y - p[t-1]),
//compute b[t] = b[t-1] + inv(X'X~)X'(y - p[t-1]) where X~ is W[t-1]X computed directly.
//The idea is that the vast majority of W[t-1] cells are 0.0 and so can be ignored
func (r *Regression) constructNewCoefficientsVector(oldBVector matrix.Matrix, xMatrix matrix.Matrix, yVector matrix.Matrix, oldPVector matrix.Matrix) matrix.Matrix {
	Xt := matrix.Transpose(xMatrix)                // X'
	A, err := r.computeXtilde(oldPVector, xMatrix) // WX

	if err != nil {
		return nil
	}

	B := matrix.Product(Xt, A) // X'WX
	C := matrix.Inverse(B)     // inv(X'WX)

	// Computing the inverse can blow up easily
	if C == nil {
		return nil
	}

	//Save standard error
	errLength := len(r.model.StandardErrors)
	for i := 0; i < errLength; i++ {
		r.model.StandardErrors[i] = math.Sqrt(C.Get(i, i))
	}

	D := matrix.Product(C, Xt)                   // inv(X'WX)X'
	YP := matrix.Difference(yVector, oldPVector) // y-p
	E := matrix.Product(D, YP)                   // inv(X'WX)X'(y-p)
	result := matrix.Sum(oldBVector, E)          // b + inv(X'WX)X'(y-p)

	return result
}

//Note: W[t-1] is nxn which could be huge so instead of computing b[t] = b[t-1] + inv(X'W[t-1]X)X'(y - p[t-1]) directly
//we compute the W[t-1]X part, without the use of W.
//
//Since W is derived from the prob vector and W has non-0.0 elements only on the diagonal we can avoid a ton of work
//by using the prob vector directly and not computing W at all.
//
//Some of the research papers refer to the product W[t-1]X as X~ hence the name of this method.
//Ex: if xMatrix is 10x4 then W would be 10x10 so WX would be 10x4 -- the same size as X
func (r *Regression) computeXtilde(pVector matrix.Matrix, xMatrix matrix.Matrix) (matrix.Matrix, error) {
	pRows := pVector.Rows()
	xRows := xMatrix.Rows()
	xCols := xMatrix.Cols()

	if pRows != xRows {
		return nil, errors.New("The pVector and xMatrix are not compatible in computeXtilde")
	}

	//We are not doing matrix multiplication. the p column vector sort of lays on top of each matrix column.
	result := matrix.Zeros(pRows, xCols)
	for i := 0; i < pRows; i++ {
		for j := 0; j < xCols; j++ {
			pVal := pVector.Get(i, 0)
			xVal := xMatrix.Get(i, j)
			result.Set(i, j, pVal*(1.0-pVal)*xVal) //Note the p(1-p)
		}
	}

	return result, nil
}

func (r *Regression) noChange(oldBVector matrix.Matrix, newBVector matrix.Matrix, epsilon float64) bool {
	length := oldBVector.Rows()
	for i := 0; i < length; i++ {
		oldVal := oldBVector.Get(i, 0)
		newVal := newBVector.Get(i, 0)

		//We have at least one change
		if math.Abs(oldVal-newVal) > epsilon {
			return false
		}
	}

	//No change
	return true
}

func (r *Regression) outOfControl(oldBVector matrix.Matrix, newBVector matrix.Matrix, jumpFactor float64) bool {
	length := oldBVector.Rows()
	for i := 0; i < length; i++ {
		//Anything goes if old value is 0.0
		oldVal := oldBVector.Get(i, 0)
		if oldVal == 0.0 {
			return false
		}

		newVal := newBVector.Get(i, 0)

		//Too big of a change
		if math.Abs(oldVal-newVal)/math.Abs(oldVal) > jumpFactor {
			return true
		}
	}

	//Still in control
	return false
}

//Odds ratio is e^coefficient
func (r *Regression) computeOddsRatio() error {
	length := len(r.model.Coefficients)
	if length == 0 {
		return errors.New("Error: Coefficients in models are not generated yet")
	}

	r.model.OddsRatio = make([]float64, length)
	for i := 0; i < length; i++ {
		odds := math.Exp(r.model.Coefficients[i])
		r.model.OddsRatio[i] = odds
	}

	return nil
}

//-16325072
//1.08577462845905
//p-Value for wald statistics is coefficient / standard error
func (r *Regression) computeWaldStatistic() error {
	length := len(r.model.Coefficients)
	if length == 0 {
		return errors.New("Error: Coefficients in models are not generated yet")
	}

	//Assume standard error exists when coefficient is already computed
	r.model.WaldStatistics = make([]float64, length)
	for i := 0; i < length; i++ {
		pvalue := r.model.Coefficients[i] / r.model.StandardErrors[i]
		r.model.WaldStatistics[i] = pvalue
	}

	return nil

}

//lower = coefficient - 1.96 * standard error, upper = coefficient + 1.96 * standard error
func (r *Regression) computeConfidenceInterval() error {
	length := len(r.model.Coefficients)
	if length == 0 {
		return errors.New("Error: Coefficients in models are not generated yet")
	}

	//Assume standard error exists when coefficient is already computed
	r.model.LowerConfidenceIntervals = make([]float64, length)
	r.model.UpperConfidenceIntervals = make([]float64, length)

	for i := 0; i < length; i++ {
		offset := 1.96 * r.model.StandardErrors[i]
		r.model.LowerConfidenceIntervals[i] = r.model.Coefficients[i] - offset
		r.model.UpperConfidenceIntervals[i] = r.model.Coefficients[i] + offset
	}

	return nil
}

//Log likelihood ln LF = TotalAddition[(Yi * ln Pi) + (1 - Yi) * ln (1 - Pi)]
func (r *Regression) computeLogLikelihood() error {
	numData := len(r.dataPoints)
	numVariables := len(r.dataPoints[0].Variables)

	//Create test data matrix for observed (result) and (independent) variables
	testObserved := matrix.Zeros(numData, 1)
	testVariables := matrix.Zeros(numData, numVariables+1)

	//Copy data to matrix
	for i := 0; i < numData; i++ {
		testObserved.Set(i, 0, r.dataPoints[i].Result)
		for j := 0; j < numVariables+1; j++ {
			if j == 0 {
				testVariables.Set(i, 0, 1)
			} else {
				testVariables.Set(i, j, r.dataPoints[i].Variables[j-1])
			}
		}
	}

	//Error check the coefficient
	coeffLen := len(r.model.Coefficients)
	if coeffLen == 0 {
		return errors.New("Error: Coefficients in models are not generated yet")
	}

	//Create coefficient matrix from already generated model
	bVector := matrix.Zeros(coeffLen, 1)
	for i := 0; i < coeffLen; i++ {
		bVector.Set(i, 0, r.model.Coefficients[i])
	}

	//Error checking again
	xRows := testVariables.Rows()
	xCols := testVariables.Cols()
	yRows := testObserved.Rows()
	bRows := bVector.Rows()
	if xCols != bRows || xRows != yRows {
		return errors.New("Error:Bad dimensions for xMatrix or yVector or bVector in computeLogLikelihood()")
	}

	pVector, err := r.constructProbVector(testVariables, bVector)
	if err != nil {
		return err
	}

	pRows := pVector.Rows()
	if pRows != xRows {
		return errors.New("Error:Unequal rows in prob vector and design matrix in computeLogLikelihood()")
	}

	//Initiate cases
	logLikelihood := 0.0

	for i := 0; i < yRows; i++ {
		pVal := pVector.Get(i, 0)
		observedVal := testObserved.Get(i, 0)

		current := 0.0
		if observedVal == 0.0 {
			current = math.Log(1 - pVal)
		} else if observedVal == 1.0 {
			current = math.Log(pVal)
		}

		if r.debugMode {
			r.debugContext.Infof("\nY is %v and P is %v", observedVal, pVal)
			r.debugContext.Infof("\n(Yi * ln Pi) + (1 - Yi) * ln (1 - Pi): %v", current)
		}

		logLikelihood += current
	}

	if r.debugMode {
		r.debugContext.Infof("\nLog likelihood: %v", logLikelihood)
	}

	//Save the calculated log likelihood result
	r.model.LogLikelihood = logLikelihood

	return nil
}

func (r *Regression) computeDeviance() {
	r.model.Deviance = -2 * r.model.LogLikelihood
}

func (r *Regression) computeChiSquare() {
	//Calculate the baseline model log likelihood
	length := len(r.dataPoints)
	logLikelihoodBase := 0.0
	for i := 0; i < length; i++ {
		//Assume predicted probability of 0.5
		//Current is always ln 0.5 because whether observed is 1 or 0, 1 - 0.5 and 0.5 is the same
		logLikelihoodBase += math.Log(0.5)
	}

	//Calculate baseline deviance
	devianceBase := -2 * logLikelihoodBase
	if r.debugMode {
		r.debugContext.Infof("\nBase is %v and Deviance is %v", devianceBase, r.model.Deviance)
	}

	//Save the difference
	r.model.ChiSquare = devianceBase - r.model.Deviance

	if r.debugMode {
		r.debugContext.Infof("\nLog ChiSquare: %v", r.model.ChiSquare)
	}
}

func (r *Regression) Predict(testData DataPoint) (predicted float64, err error) {
	//Create matrix for independent variables
	numVariables := len(testData.Variables)
	testVariables := matrix.Zeros(1, numVariables+1)

	for i := 0; i < numVariables; i++ {
		if i == 0 {
			testVariables.Set(0, 0, 1)
		} else {
			testVariables.Set(0, i, testData.Variables[i-i])
		}
	}

	//Create coefficient matrix from already generated model
	coeffLen := len(r.model.Coefficients)
	bVector := matrix.Zeros(coeffLen, 1)

	for i := 0; i < coeffLen; i++ {
		bVector.Set(i, 0, r.model.Coefficients[i])
	}

	//Error checking first
	xCols := testVariables.Cols()
	bRows := bVector.Rows()
	if xCols != bRows {
		return 0.0, errors.New("Error:Bad dimensions for xMatrix or bVector in Predict()")
	}

	//Calculate probability vector
	pVector, err := r.constructProbVector(testVariables, bVector)
	if err != nil {
		return 0.0, err
	}

	return pVector.Get(0, 0), nil
}

func (r *Regression) TestModel(testData []DataPoint) (accuracy float64, err error) {
	numData := len(testData)
	numVariables := len(testData[0].Variables)

	//Create test data matrix for observed (result) and (independent) variables
	testObserved := matrix.Zeros(numData, 1)
	testVariables := matrix.Zeros(numData, numVariables+1)

	//Copy data to matrix
	for i := 0; i < numData; i++ {
		testObserved.Set(i, 0, testData[i].Result)
		for j := 0; j < numVariables+1; j++ {
			if j == 0 {
				testVariables.Set(i, 0, 1)
			} else {
				testVariables.Set(i, j, testData[i].Variables[j-1])
			}
		}
	}

	//Create coefficient matrix from already generated model
	coeffLen := len(r.model.Coefficients)
	bVector := matrix.Zeros(coeffLen, 1)
	for i := 0; i < coeffLen; i++ {
		bVector.Set(i, 0, r.model.Coefficients[i])
	}

	//Error checking first
	xRows := testVariables.Rows()
	xCols := testVariables.Cols()
	yRows := testObserved.Rows()
	bRows := bVector.Rows()
	if xCols != bRows || xRows != yRows {
		return 0.0, errors.New("Error:Bad dimensions for xMatrix or yVector or bVector in TestModel()")
	}

	pVector, err := r.constructProbVector(testVariables, bVector)
	if err != nil {
		return 0.0, err
	}

	pRows := pVector.Rows()
	if pRows != xRows {
		return 0.0, errors.New("Error:Unequal rows in prob vector and design matrix in TestModel()")
	}

	//Initiate cases
	numberCasesCorrect := 0
	numberCasesWrong := 0

	for i := 0; i < yRows; i++ {
		pVal := pVector.Get(i, 0)
		observedVal := testObserved.Get(i, 0)

		if r.debugMode {
			r.debugContext.Infof("\nPredicted vs Test Result: %v vs %v\n", pVal, observedVal)
		}

		if pVal >= 0.50 && observedVal == 1.0 {
			numberCasesCorrect += 1
		} else if pVal < 0.50 && observedVal == 0.0 {
			numberCasesCorrect += 1
		} else {
			numberCasesWrong += 1
		}
	}

	//Calculate correct prediction percentage
	total := numberCasesCorrect + numberCasesWrong
	correctPercentage := (100.0 * float64(numberCasesCorrect)) / float64(total)

	if r.debugMode {
		r.debugContext.Infof("\nCorrect case vs Wrong case: %v vs %v\n", numberCasesCorrect, numberCasesWrong)
		r.debugContext.Infof("\nCorrect predicted percentage: %v", correctPercentage)
	}

	if total == 0 {
		return 0.0, nil
	} else {
		return correctPercentage, nil
	}
}

func (r *Regression) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Name|Coefficient|Odds Ratio|Std. Error|p-Value|Lower Confidence|Upper Confidence\n")

	length := len(r.variableNames) + 1
	for i := 0; i < length; i++ {
		index := i - 1

		var variableString string
		if index == -1 {
			variableString = "Intercept"
		} else {
			variableString = r.variableNames[index]
		}

		coeffString := strconv.FormatFloat(r.model.Coefficients[i], 'f', 6, 64)
		oddsRatioString := strconv.FormatFloat(r.model.OddsRatio[i], 'f', 6, 64)
		stdErrString := strconv.FormatFloat(r.model.StandardErrors[i], 'f', 6, 64)
		pValueString := strconv.FormatFloat(r.model.WaldStatistics[i], 'f', 6, 64)
		lowerString := strconv.FormatFloat(r.model.LowerConfidenceIntervals[i], 'f', 6, 64)
		upperString := strconv.FormatFloat(r.model.UpperConfidenceIntervals[i], 'f', 6, 64)

		buffer.WriteString("\n")
		buffer.WriteString(variableString)
		buffer.WriteString("|")
		buffer.WriteString(coeffString)
		buffer.WriteString("|")
		buffer.WriteString(oddsRatioString)
		buffer.WriteString("|")
		buffer.WriteString(stdErrString)
		buffer.WriteString("|")
		buffer.WriteString(pValueString)
		buffer.WriteString("|")
		buffer.WriteString(lowerString)
		buffer.WriteString("|")
		buffer.WriteString(upperString)
		buffer.WriteString("\n")
	}

	logLikelihoodString := strconv.FormatFloat(r.model.LogLikelihood, 'f', 15, 64)
	devianceString := strconv.FormatFloat(r.model.Deviance, 'f', 15, 64)
	chiString := strconv.FormatFloat(r.model.ChiSquare, 'f', 15, 64)

	buffer.WriteString("\n")
	buffer.WriteString("Log Likelihood: ")
	buffer.WriteString(logLikelihoodString)
	buffer.WriteString("\n")
	buffer.WriteString("-2 * Log Likelihood (Deviance): ")
	buffer.WriteString(devianceString)
	buffer.WriteString("\n")
	buffer.WriteString("Chi-Square Goodness of Fit: ")
	buffer.WriteString(chiString)

	buffer.WriteString("\n\n")
	buffer.WriteString("Note: Critical chi-square value for 0.05 at 6 degree of freedom is 12.59158724")

	return buffer.String()
}

func (r *Regression) StringHTML() string {
	//HTML string buffer
	var buffer bytes.Buffer

	//Table header
	buffer.WriteString("<table>")
	buffer.WriteString("<tr>")
	buffer.WriteString("<td>Name</td>")
	buffer.WriteString("<td>Coefficient</td>")
	buffer.WriteString("<td>Odds Ratio</td>")
	buffer.WriteString("<td>Std. Error</td>")
	buffer.WriteString("<td>p-Value</td>")
	buffer.WriteString("<td>Lower Confidence</td>")
	buffer.WriteString("<td>Upper Confidence</td>")
	buffer.WriteString("</tr>")

	//Table attributes
	length := len(r.variableNames) + 1
	for i := 0; i < length; i++ {
		//Decrease one for fun
		index := i - 1

		//Independent variable names
		var variableString string
		if index == -1 {
			variableString = "Intercept"
		} else {
			variableString = r.variableNames[index]
		}

		//Convert attributes to string
		coeffString := strconv.FormatFloat(r.model.Coefficients[i], 'f', 6, 64)
		oddsRatioString := strconv.FormatFloat(r.model.OddsRatio[i], 'f', 6, 64)
		stdErrString := strconv.FormatFloat(r.model.StandardErrors[i], 'f', 6, 64)
		pValueString := strconv.FormatFloat(r.model.WaldStatistics[i], 'f', 6, 64)
		lowerString := strconv.FormatFloat(r.model.LowerConfidenceIntervals[i], 'f', 6, 64)
		upperString := strconv.FormatFloat(r.model.UpperConfidenceIntervals[i], 'f', 6, 64)

		//Header
		buffer.WriteString("<tr>")

		//Model attributes
		buffer.WriteString("<td>")
		buffer.WriteString(variableString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(coeffString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(oddsRatioString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(stdErrString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(pValueString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(lowerString)
		buffer.WriteString("</td>")

		buffer.WriteString("<td>")
		buffer.WriteString(upperString)
		buffer.WriteString("</td>")

		//Footer
		buffer.WriteString("</tr>")
	}

	//End table
	buffer.WriteString("</table>")

	//Calculate model performance
	logLikelihoodString := strconv.FormatFloat(r.model.LogLikelihood, 'f', 15, 64)
	devianceString := strconv.FormatFloat(r.model.Deviance, 'f', 15, 64)
	chiString := strconv.FormatFloat(r.model.ChiSquare, 'f', 15, 64)

	//Model performance
	buffer.WriteString("<div>Log Likelihood: ")
	buffer.WriteString(logLikelihoodString)
	buffer.WriteString("</div>")
	buffer.WriteString("<div>-2 * Log Likelihood (Deviance): ")
	buffer.WriteString(devianceString)
	buffer.WriteString("</div>")
	buffer.WriteString("<div>Chi-Square Goodness of Fit: ")
	buffer.WriteString(chiString)
	buffer.WriteString("</div>")
	buffer.WriteString("<br/>")
	buffer.WriteString("<div>Note: Critical chi-square value for 0.05 at 6 degree of freedom is 12.59158724</div>")

	return buffer.String()
}
