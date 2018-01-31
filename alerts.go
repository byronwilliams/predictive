package main

import (
	"github.com/ohheydom/linearregression"
)

var minAccepted = 1000.0

func calculateMA(sensorID string, rawValues []float64, windowSize int) float64 {
	/**
	Look for the sensor data for the sensor
	Calculate the average daily temperature efficiency
	*/

	var lastMA float64

	for i := 0; i < len(rawValues)-windowSize; i++ {
		var total float64
		for j := 0; j < windowSize; j++ {
			total = total + rawValues[i+j]
		}

		var avg = total / float64(windowSize)

		lastMA = avg
	}

	return lastMA
}

func calculateAlert(sensorID string, rawValues []float64, windowSize int) (int, float64) {
	/**
	Look for the sensor data for the sensor
	Calculate the average daily temperature efficiency
	*/
	var alertTriggerDay = -1
	var alertTriggerValue float64
	for i := 0; i < len(rawValues)-windowSize; i++ {
		var total float64
		for j := 0; j < windowSize; j++ {
			total = total + rawValues[i+j]
		}

		var avg = total / float64(windowSize)

		if alertTriggerDay == -1 {
			if avg <= minAccepted {
				alertTriggerDay = i
				alertTriggerValue = avg
			}
		}
	}

	return alertTriggerDay, alertTriggerValue
}

func calculateLR(sensorID string, rawValues []float64, windowSize int) (int, float64) {

	for i := 0; i < len(rawValues)-windowSize; i++ {
		var xs = make([][]float64, windowSize)
		var ys = make([]float64, windowSize)

		for j := 0; j < windowSize; j++ {
			xs[j] = make([]float64, 1)
			xs[j][0] = float64(j)
			ys[j] = float64(rawValues[i+j])
		}

		var lr = linearregression.LinearRegression{}
		lr.Fit(xs, ys)

		var yPred = lr.Predict([][]float64{[]float64{rawValues[windowSize+1]}})

		if yPred[0] >= minAccepted {
			return i, yPred[0]
		}
	}

	return 0, 0
}
