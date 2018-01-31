package main

import (
	"log"
	"testing"
)

var rawValues = []float64{
	1000,
	1000,
	1000,
	1000,
	1000,
}

func TestCalculateAlert(t *testing.T) {
	dayTriggered, triggeredVal := calculateAlert("", rawValues, 3)
	log.Println(dayTriggered, triggeredVal)
	t.Fail()
}

func TestCalculateLR(t *testing.T) {
	dayTriggered, triggeredVal := calculateLR("", rawValues, 3)
	log.Println(dayTriggered, triggeredVal)
	t.Fail()
}
