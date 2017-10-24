package main

import (
	"log"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

type Build struct {
	Code     string `form:"code"`
	Language string `form:"language"`
}

type Run struct {
	Code     string `form:"code"`
	Language string `form:"language"`
	Stdin    string `form:"stdin"`
}

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal("Docker client is not connected.")
	}
	options := types.ContainerListOptions{All: true}
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.POST("/build", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/node", func(c *gin.Context) {
		containers, err := cli.ContainerList(context.Background(), options)
		if err != nil {
			log.Print(err)
			c.JSON(500, gin.H{
				"error": "Does not permit to fetch container list",
			})
		}
		c.JSON(200, gin.H{
			"containers": containers,
		})
	})
	r.Run()
}
