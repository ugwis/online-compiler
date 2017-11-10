package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
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

type Language struct {
	Name        string   `yaml:"name"`
	DockerImage string   `yaml:"docker_image"`
	BuildCmd    []string `yaml:"build_cmd"`
	RunCmd      []string `yaml:"run_cmd"`
	CodeFile    string   `yaml:"code_file"`
}

type Languages struct {
	Language map[string]Language `yaml:"language"`
}

func main() {
	ctx := context.Background()

	// Read languges setttings
	buf, err := ioutil.ReadFile("./languages.yaml")
	if err != nil {
		fmt.Println(err.Error())
	}
	var lang Languages
	err = yaml.Unmarshal(buf, &lang)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", lang)

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal("Docker client is not connected.")
	}
	options := types.ContainerListOptions{All: true}

	res, err := cli.ImagePull(ctx, "bash", types.ImagePullOptions{})
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, res)

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.POST("/build", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		var query Build
		/*runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()*/
		if err := c.BindJSON(&query); err == nil {

			// Make hash
			fmt.Println("Make hash")
			h := md5.New()
			io.WriteString(h, query.Language)
			io.WriteString(h, query.Code)
			runningHash := hex.EncodeToString(h.Sum(nil))
			fmt.Println("runningHash: " + runningHash)

			// Save code
			fmt.Println("Save code")
			if err := os.MkdirAll("/tmp/compiler/"+runningHash, 0755); err != nil {
				fmt.Println(err.Error())
				return
			}
			// TODO fix file ext
			fp, err := os.OpenFile("/tmp/compiler/"+runningHash+"/main.sh", os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer fp.Close()
			writer := bufio.NewWriter(fp)
			_, err = writer.WriteString(query.Code)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			writer.Flush()

			// Create container
			// TODO: Limit container spec
			fmt.Println("Create container")
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:      lang.Language[query.Language].DockerImage,
				WorkingDir: "/workspace",
				Cmd:        lang.Language[query.Language].RunCmd,
			}, &container.HostConfig{
				Mounts: []mount.Mount{
					mount.Mount{
						Type:   mount.TypeBind,
						Source: "/tmp/compiler/" + runningHash,
						Target: "/workspace",
					},
				},
				AutoRemove: true,
			}, nil, "")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				fmt.Println(err.Error())
				return
			}

			// Start container
			err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				fmt.Println(err.Error())
				return
			}

			// Flow log of Stdout
			out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, Follow: true})
			if err != nil {
				fmt.Println(err.Error())
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			rd := bufio.NewReader(out)
			c.Stream(func(w io.Writer) bool {
				line, _, err := rd.ReadLine()
				w.Write(line)
				w.Write([]byte("\n"))
				if err == io.EOF {
					return false
				} else if err != nil {
					fmt.Println(err.Error())
					return false
				}
				return true
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	})
	r.GET("/node", func(c *gin.Context) {
		containers, err := cli.ContainerList(ctx, options)
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
