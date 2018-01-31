package main

import (
	"log"
	"time"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/devices"
)

type TemperatureChangePacket struct {
	SensorID string
	Value    float64
}

var sensorTags = []string{"24:71:89:C0:23:80", "CC:78:AB:7F:72:84"}

const adapterID = "hci0"

func main() {
	var err error

	if err = api.TurnOnAdapter(adapterID); err != nil {
		panic(err)
	}

	if err = api.TurnOnBluetooth(); err != nil {
		panic(err)
	}

	log.Println("Discovery on")

	if err := api.StartDiscoveryOn(adapterID); err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 5)

	err = api.StopDiscoveryOn(adapterID)
	if err != nil {
		panic(err)
	}

	var sts = make([]*devices.SensorTag, len(sensorTags))

	for i := 0; i < len(sensorTags); i++ {
		var tagAddress = sensorTags[i]
		log.Println(tagAddress, "Getting Device by Address")

		dev, err := api.GetDeviceByAddress(tagAddress)
		if err != nil {
			panic(err)
		}

		if dev == nil {
			panic("Device not found")
		}

		log.Println(tagAddress, "Got device, connecting")

		err = dev.Connect()
		if err != nil {
			panic(err)
		}

		log.Println(tagAddress, "Creating NewSensorTag")

		sensorTag, err := devices.NewSensorTag(dev)
		if err != nil {
			panic(err)
		}

		sts[i] = sensorTag
	}

	for {
		for i := 0; i < len(sts); i++ {
			log.Println("READING", sensorTags[i])
			var temp = readTemperature(sensorTags[i], sts[i])
			// go nb.TemperatureChange(TemperatureChangePacket{
			// SensorID: sensorTags[i],
			// Value:    temp,
			// })
			log.Println(TemperatureChangePacket{
				SensorID: sensorTags[i],
				Value:    temp,
			})
		}

		time.Sleep(1 * time.Second)

	}
}

func readTemperature(id string, sensorTag *devices.SensorTag) float64 {
	var err error
	ie, err := sensorTag.Temperature.IsEnabled()

	if sensorTag.Connect() != nil {
		log.Println("XXX", err)
		return -88

	}

	if !ie {
		if err = sensorTag.Temperature.Enable(); err != nil {
			log.Println("XXX", err)
			return -99
		}
	}

	temp, err := sensorTag.Temperature.Read()
	if err != nil {
		panic(err)
	}
	log.Printf("Temperature [%s] %.2f°", id, temp)
	return temp
}
