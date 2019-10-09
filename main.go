package main

import (
	"fabric-byzantine/server"
	"fabric-byzantine/server/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

var logger = helpers.GetLogger()

func main() {
	router := gin.Default()
	router.GET("/query/:channel", func(c *gin.Context) {
		channel := c.Param("channel")
		data, err := server.GetSdkProvider().QueryCC(channel, "token", "balance", [][]byte{[]byte("fab"), []byte("alice")})
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
