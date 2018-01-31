// Copyright 2014 Dirk Jablonowski. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
weatherstation is a example how to program the weatherstation kit in go (golang).

The idea is, that this example application searches for the bricklets and print
out their informations. A LCD 20x4 Bricklet is needed for printing out something.

The program could take two parameters.
One for the connection address. It defaults to localhost:4223.
With the other parameter you could trigger that the output will be printed to
the console too. Default is false (no output on console).

You need the bricker api code.
  go get github.com/dirkjabl/bricker
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dirkjabl/bricker"
	"github.com/dirkjabl/bricker/connector/buffered"
	"github.com/dirkjabl/bricker/device"
	"github.com/dirkjabl/bricker/device/bricklet/moisture"
	"github.com/dirkjabl/bricker/device/bricklet/temperature"
	"github.com/dirkjabl/bricker/device/enumerate"
	"github.com/dirkjabl/bricker/device/identity"
)

const (
	cn                   = "ws" // connectorname
	blTemperature uint16 = 216  // Temperature bricklet device identifer
	blMoisture    uint16 = 232  // Temperature bricklet device identifer
)

type reading struct {
	Hostname    string
	SensorID    uint32
	SensorType  uint16
	Reading     interface{}
	MinAlarm    float64
	MaxAlarm    float64
	Data        string    `json:"data"`
	Event       string    `json:"event"`
	PublishedAt time.Time `json:"published_at"`

	Alarm string `json:"alarm"`

	CE  float64 `json:"ce"`
	TE  float64 `json:"te"`
	MRE float64 `json:"mre"`
	TUF float64 `json:"tuf"`
}

type ByPublishedAt []reading

func (a ByPublishedAt) Len() int           { return len(a) }
func (a ByPublishedAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPublishedAt) Less(i, j int) bool { return a[i].PublishedAt.Before(a[j].PublishedAt) }

// bricklet type for remember important data
type bricklet struct {
	has          bool           // if the bricklet exists
	sub          *device.Device // subscriber
	brickletType uint16         // bricklet type
	uid          uint32         // uid
}

// Data structur, remembers which bricklet exists, what for a address to use and the bricker.
var conf struct {
	addr          string               // address of the stack
	brick         *bricker.Bricker     // bricker
	showOnConsole bool                 // show output from the LCD on the console, too
	bricklets     map[uint32]*bricklet // Map with all supportet bricklets

	brickletLock sync.RWMutex
}

// main routine, will startup.
func readSensors(hostnamePlus string, sseBroker *SSEBroker) {
	// need the address of the stack with the weatherstation kit bricklets.
	var addr = flag.String("addr", "localhost:4223",
		"address of the brickd, default is localhost:4223")
	var soc = flag.Bool("console", false,
		"show output from lcd on the console, too: default false")
	flag.Parse()

	// create map for the bricklets
	conf.brickletLock.Lock()
	conf.bricklets = make(map[uint32]*bricklet)
	conf.brickletLock.Unlock()

	// conf.bricklets = map[uint16]*bricklet{
	// 	bl_lcd: &bricklet{has: false, cb: workLcd}, // LCD 20x4
	// 	// bl_humidity:     &bricklet{has: false, cb: workHumidity},     // humidity
	// 	// bl_barometer:    &bricklet{has: false, cb: workBarometer},    // barometer
	// 	// bl_ambientlight: &bricklet{has: false, cb: workAmbientlight}, // ambient light
	// 	blTemperature: &bricklet{has: false, cb: workTemp}, // temperature
	// }

	// remember the flags
	conf.addr = *addr
	conf.showOnConsole = *soc

	// Create a bricker object
	conf.brick = bricker.New()
	defer conf.brick.Done() // later for stopping the bricker

	var conn *buffered.ConnectorBuffered
	var err error

	for conn == nil {
		// create a connection to a real brick stack
		conn, err = buffered.New(conf.addr, 20, 10)
		if err != nil { // no connection
			fmt.Printf("No connection, sleeping for 5: %s\n", err.Error())
			time.Sleep(5 * time.Second)
		}
	}
	log.Println(">>>>>>>>>>>>>>> Connected to BrickD")
	defer conn.Done() // later for stopping current connection

	// attach the connector to the bricker
	err = conf.brick.Attach(conn, cn) // ws is the name for this connection
	if err != nil {                   // no bricker, no fun
		fmt.Printf("Could not attach connection to bricker: %s\n", err.Error())
		return
	}
	log.Println(">>>>>>>>>>>>>>> Attached to BrickD")
	defer conf.brick.Release(cn) // later to release connection from bricker

	var wg = sync.WaitGroup{}
	wg.Add(1)

	// Look out for the hardware(bricklets) inside the given stack
	// hw := make(chan *enumerate.Enumeration, 4)
	en := enumerate.Enumerate("Enumerate", false,
		func(r device.Resulter, err error) {
			if err == nil && r != nil { // only if no error occur
				if v, ok := r.(*enumerate.Enumeration); ok {
					// log.Println("V", v)
					// hw <- v
					hardwareidentify(v, hostnamePlus, sseBroker)
				}
			}
		})

	// attach enumeration subscriber to the bricker
	err = conf.brick.Subscribe(en, cn)

	wg.Wait()

	// go on with the program, waiting for a key
	// fmt.Printf("Press return for stop.\n")
	// _, _ = bufio.NewReader(os.Stdin).ReadByte()
	// if conf.bricklets[bl_lcd].has {
	// _ = lcd20x4.BacklightOffFuture(conf.brick, cn, conf.bricklets[bl_lcd].uid)
	// }
}

// This handler identify the founded hardware and if possible
// it starts or stops a handler/callback to read out the sensors or to display.
func hardwareidentify(value *enumerate.Enumeration, hostnamePlus string, sseBroker *SSEBroker) {
	var uid uint32
	if value.EnumerationType != enumerate.EnumerationTypeDisconneted {
		// exists and is active
		uid = value.IntUid()
	} else {
		uid = 0
	}
	// log.Println(value, value.DeviceIdentifer, "uid", uid)
	conf.brickletLock.Lock()
	var b = bricklet{
		has:          true,
		uid:          uid,
		brickletType: value.DeviceIdentifer,
	}
	conf.bricklets[uid] = &b

	switch b.brickletType {
	case blTemperature:
		b.sub = identity.GetIdentity("", b.uid, nilHandler)

		go pollTemperature(&b, cn, hostnamePlus, sseBroker)
	case blMoisture:
		b.sub = identity.GetIdentity("", b.uid, nilHandler)

		go pollMoisture(&b, cn, hostnamePlus, sseBroker)
	default:
		log.Println("Unknown type", b.brickletType, b.uid)
	}

	conf.brickletLock.Unlock()
}

func pollTemperature(b *bricklet, connectorName, hostnamePlus string, sseBroker *SSEBroker) {
	var ticker = time.Tick(time.Millisecond * 1500)
	for {
		select {
		case <-ticker:
			conf.brickletLock.Lock()
			var st = time.Now()
			temp := temperature.GetTemperatureFuture(conf.brick, cn, b.uid)
			if temp != nil { // only if a result exists, it is a pointer(!)
				fmt.Printf("Temperature (%d): %02.02f °C (%s)\n", b.uid, temp.Float64(), time.Now().Sub(st))
			}
			conf.brickletLock.Unlock()

			sseBroker.NewReading(reading{
				Hostname:   hostnamePlus,
				SensorID:   b.uid,
				SensorType: b.brickletType,
				Reading:    temp.Float64(),
			})

		}
	}
}

func pollMoisture(b *bricklet, connectorName, hostnamePlus string, sseBroker *SSEBroker) {
	var ticker = time.Tick(time.Millisecond * 1500)
	for {
		select {
		case <-ticker:
			conf.brickletLock.Lock()
			var st = time.Now()
			m := moisture.GetMoistureValueFuture(conf.brick, cn, b.uid)
			if m != nil { // only if a result exists, it is a pointer(!)
				fmt.Printf("Moisture (%d): %d (%s)\n", b.uid, m.Value, time.Now().Sub(st))
			}
			conf.brickletLock.Unlock()

			sseBroker.NewReading(reading{
				Hostname:   hostnamePlus,
				SensorID:   b.uid,
				SensorType: b.brickletType,
				Reading:    m.Value,
			})
		}
	}
}

func nilHandler(r device.Resulter, err error) {
}
