package main

import (
	"fabric-byzantine/server"
	"fabric-byzantine/server/helpers"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var logger = helpers.GetLogger()

func timerTask() {
	c := time.Tick(5 * time.Second)
	for {
		<-c
		go server.GetSdkProvider().InvokeCC("mychannel1", "token", "transfer", [][]byte{[]byte("fab"), []byte("alice"), []byte("bob"), []byte("1"), []byte("true")})
	}
}

func main() {
	go server.GetSdkProvider().BlockListener("mychannel1")
	go timerTask()

	router := gin.Default()
	router.GET("/query", func(c *gin.Context) {
		data, err := server.GetSdkProvider().QueryCC("mychannel1", "token", "balance", [][]byte{[]byte("fab"), []byte("alice")})
		if err != nil {
			logger.Error("query err: %v", err)
			c.JSON(http.StatusOK, err)
		} else {
			c.JSON(http.StatusOK, data)
		}
	})

	router.POST("/invoke", func(c *gin.Context) {
		message := c.PostForm("message")

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
		})
	})

	_ = router.Run(":8080")
}
