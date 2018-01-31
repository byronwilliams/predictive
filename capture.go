package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func mainx() {
	r := gin.Default()

	r.POST("/", func(c *gin.Context) {
		var err error
		var rv json.RawMessage
		if err = c.BindJSON(&rv); err != nil {
			c.AbortWithStatus(500)
			return
		}

		var fn = "x_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".json"

		ioutil.WriteFile(fn, rv, 0700)

		c.String(http.StatusAccepted, "text/plain", "ok")
		return
	})

	log.Fatal(r.Run("0.0.0.0:80"))
}
