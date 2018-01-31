package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	logging "github.com/op/go-logging"
	debug "github.com/tj/go-debug"

	client "github.com/influxdata/influxdb/client/v2"
)

var logger = logging.MustGetLogger("main")
var dbg = debug.Debug("bluez:main")

var adapterID = "hci0"

var tagAddresses = []string{"24:71:89:C0:23:80"} // , "CC:78:AB:7F:72:84"}

const avgRateOfChange = 0.4 // 0.04C per minute, 1.0C every 20 mins
const minAlarm = 500
const maxAlarm = 1500

//SensorTagTemperatureExample example of reading temperature from a TI sensortag

func webserver(broker *SSEBroker) {
	var r = gin.Default()
	r.LoadHTMLGlob("templates/*.html")

	// cl, err := client.NewHTTPClient(client.HTTPConfig{
	// 	Addr: "http://localhost:8086",
	// 	// Username: username,
	// 	// Password: password,
	// })

	// if err != nil {
	// 	panic(err)
	// }
	var local = false
	if local {
		go func() {
			files, _ := filepath.Glob("data/*.json")

			var rs = make([]reading, len(files))

			for i := 0; i < len(files); i++ {
				log.Println(i)
				fc, _ := ioutil.ReadFile(files[i])
				var rd reading
				json.Unmarshal(fc, &rd)

				rs[i] = rd
			}

			sort.Sort(ByPublishedAt(rs))

			for i := 0; i < len(rs); i++ {
				broker.NewReading(rs[i])

				time.Sleep(time.Millisecond * 500)
			}
		}()
	}

	var i float64 = 0
	var locker = sync.RWMutex{}

	go func() {
		for {
			time.Sleep(time.Second * 60)
			locker.Lock()
			i = 0
			locker.Unlock()
		}
	}()

	go func() {
		rand.Seed(time.Now().UnixNano())

		var mbound = 1000
		// var maxbound = 1500
		var badRounds = 15.0 / 0.3
		var goodRounds = badRounds * 9

		for {
			locker.RLock()
			if i > badRounds && i < goodRounds {
				if int(i)%10 > 0 {
					broker.NewReading(reading{
						PublishedAt: time.Now(),
						Data:        strconv.Itoa(1500 + rand.Intn(1000)),
					})
				} else {
					broker.NewReading(reading{
						PublishedAt: time.Now(),
						Data:        strconv.Itoa(rand.Intn(800)),
					})
				}
			} else {
				broker.NewReading(reading{
					PublishedAt: time.Now(),
					Data:        strconv.Itoa(800 + rand.Intn(700)),
				})
			}
			locker.RUnlock()

			time.Sleep(time.Millisecond * 75)
			locker.Lock()
			i++
			locker.Unlock()

		}

		log.Println(mbound)
	}()

	r.Static("/static", ".")

	r.GET("/", func(c *gin.Context) {

		// q := fmt.Sprintf("SELECT DISTINCT  FROM %s LIMIT %d", "cpu_usage", 10)
		// res, err := queryDB(cl, q)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// for i, row := range res[0].Series[0].Values {
		// 	log.Println(i, row)
		// }

		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"Sensors": nil,
		})
	})

	r.POST("/", func(c *gin.Context) {
		var err error
		var rv reading
		if err = c.BindJSON(&rv); err != nil {
			c.AbortWithStatus(500)
			return
		}

		// var fn = "data/x_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".json"

		// ioutil.WriteFile(fn, rv, 0700)

		broker.NewReading(rv)

		c.String(http.StatusAccepted, "text/plain", "ok")
		return
	})

	r.OPTIONS("/t", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.AbortWithStatus(http.StatusOK)
	})

	r.GET("/t", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Access-Control-Allow-Origin", "*")

		var tempChangeCh = make(chan []byte)

		broker.AddClient(tempChangeCh)
		defer broker.RemoveClient(tempChangeCh)

		notify := c.Writer.(http.CloseNotifier).CloseNotify()

		go func() {
			<-notify
			broker.RemoveClient(tempChangeCh)
		}()

		// var readings = make(map[uint32][]reading)
		var historicValues = make([]float64, 0)

		var lastReading *reading

		for {
			var thisVal = <-tempChangeCh

			var tc reading
			json.Unmarshal(thisVal, &tc)

			if lastReading == nil {
				lastReading = &tc
			}

			log.Println(tc.PublishedAt, lastReading.PublishedAt, tc.PublishedAt.Before(lastReading.PublishedAt))

			if tc.PublishedAt.Before(lastReading.PublishedAt) {
				continue
			}

			lastReading = &tc

			// if _, ok := readings[tc.SensorID]; !ok {
			// 	readings[tc.SensorID] = make([]reading, 0)
			// }

			// for i := 0; i < len(readings[tc.SensorID]); i++ {
			d, _ := strconv.ParseFloat(tc.Data, 64)

			historicValues = append(historicValues, d)
			// }

			tc.MinAlarm = 800
			tc.MaxAlarm = 1500

			// if len(historicValues) > 15 {
			// log.Println(historicValues)
			ma := calculateMA("", historicValues, 30)
			// _, lr := calculateLR("", historicValues, 30)

			formatted, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", ma), 64)
			tc.CE = formatted
			tc.TE = 1000
			tc.MRE = 900
			// tc.TUF = lr

			// }

			m, _ := json.Marshal(tc)
			log.Println("thisVal", string(m))

			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(m))))
			c.Writer.Flush()
		}
	})

	log.Fatal(r.Run("0.0.0.0:80"))
}

// queryDB convenience function to query the database
func queryDB(clnt client.Client, cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: "hvac",
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}

func main() {
	var broker = NewSSEBroker()

	webserver(broker)
	// readSensors("localhost", broker)
}
