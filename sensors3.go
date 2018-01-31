// +build

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/paypal/gatt"
)

var done = make(chan struct{})
var sensorTags = map[string]bool{
	"24:71:89:C0:23:80": true,
	"CC:78:AB:7F:72:84": true,
}

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {

	if strings.HasPrefix(p.ID(), "CC") || strings.HasPrefix(p.ID(), "24") {
		// fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		// fmt.Println("  Local Name        =", a.LocalName)
		// fmt.Println("  TX Power Level    =", a.TxPowerLevel)
		// fmt.Println("  Manufacturer r =", a.ManufacturerData)
		// fmt.Println("  Service Data      =", a.ServiceData)

		p.Device().Connect(p)
	}
}

func onPeriphConnected(p gatt.Peripheral, err error) {
	fmt.Println("Connected")
	defer p.Device().CancelConnection(p)

	if err := p.SetMTU(500); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}

	for _, s := range ss {
		// msg := "Service: " + s.UUID().String()
		// if len(s.Name()) > 0 {
		// 	msg += " (" + s.Name() + ")"
		// }
		// fmt.Println(msg)

		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
		if err != nil {
			fmt.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			log.Println(c.UUID().String())
			if c.UUID().String() == "2902" {
				log.Println("Notification: 2902")
			} else if strings.HasPrefix(c.UUID().String(), "f000aa01") {
				log.Println("Data AA01*: aa01")
			} else if strings.HasPrefix(c.UUID().String(), "f000aa02") {
				log.Println("Config AA02*: aa02")
			} else {
				continue
			}

			// msg := "  Characteristic  " + c.UUID().String()
			// log.Println(msg)
			// if len(c.Name()) > 0 {
			// 	msg += " (" + c.Name() + ")"
			// }
			// msg += "\n    properties    " + c.Properties().String()
			// fmt.Println(msg)

			// // Read the characteristic, if possible.
			// if (c.Properties() & gatt.CharRead) != 0 {
			// 	b, err := p.ReadCharacteristic(c)
			// 	if err != nil {
			// 		fmt.Printf("Failed to read characteristic, err: %s\n", err)
			// 		continue
			// 	}
			// 	fmt.Printf("    value         %x | %q\n", b, b)
			// }

			// // Discovery descriptors
			ds, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				fmt.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			for _, d := range ds {
				var msg = " >>>>>>>>>>>>>>>>>>>>..  Descriptor      `" + d.UUID().String() + "`"
				log.Println(msg)

				if d.UUID().String() == "2902" {
					p.WriteDescriptor(d, []byte{0x0001})
				}

				// 	if len(d.Name()) > 0 {
				// 		msg += " (" + d.Name() + ")"

				// if strings.HasPrefix(c.UUID().String(), "F000AA02") {
				// 	log.Println("Temp")
				// 	p.WriteDescriptor(d, []byte{0x01})
				// }

				// if strings.HasPrefix(c.UUID().String(), "2902") {
				// 	log.Println("Temp")

			}

			p.SetNotifyValue(c, func(c *gatt.Characteristic, b []byte, e error) {
				log.Printf("Got back %s\n", string(b))
			})

			// }
			// fmt.Println(msg)

			// 	// Read descriptor (could fail, if it's not readable)
			// 	b, err := p.ReadDescriptor(d)
			// 	if err != nil {
			// 		fmt.Printf("Failed to read descriptor, err: %s\n", err)
			// 		continue
			// 	}
			// 	fmt.Printf("    value         %x | %q\n", b, b)
			// }

			// // Subscribe the characteristic, if possible.
			// if (c.Properties() & (gatt.CharNotify | gatt.CharIndicate)) != 0 {
			// 	f := func(c *gatt.Characteristic, b []byte, err error) {
			// 		fmt.Printf("notified: % X | %q\n", b, b)
			// 	}
			// 	if err := p.SetNotifyValue(c, f); err != nil {
			// 		fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
			// 		continue
			// 	}
			// }

		}
	}

	fmt.Printf("Waiting for 5 seconds to get some notifiations, if any.\n")
	time.Sleep(10 * time.Second)
}

func onPeriphDisconnected(p gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
	close(done)
}

func main() {
	d, err := gatt.NewDevice()
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)
	<-done
	fmt.Println("Done")
}
