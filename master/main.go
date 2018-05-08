package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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

	// Create docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal("Cannot create docker client")
	}
	options := types.ContainerListOptions{All: true}

	for {
		ver, err := cli.ServerVersion(ctx)
		if err == nil {
			fmt.Println(ver.Version)
			break
		}
		time.Sleep(1 * time.Second)
	}
	// Pull using images
	timeout := time.Duration(1 * time.Second)
	if _, err := net.DialTimeout("tcp", "hub.docker.com:80", timeout); err != nil {
		log.Println("Site unreachable, error: ", err)
	} else {
		for _, v := range lang.Language {
			res, err := cli.ImagePull(ctx, v.DockerImage, types.ImagePullOptions{})
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(os.Stdout, res)
		}
	}

	// Start routing
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.String(http.StatusOK, "pong")
	})
	r.GET("/language", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.JSON(http.StatusOK, gin.H{
			"languages": lang.Language,
		})
	})
	r.POST("/build", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		var query Build
		/*runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()*/
		if err := c.BindJSON(&query); err == nil {
			if _, ok := lang.Language[query.Language]; !ok {
				fmt.Println("Unsupported language: " + query.Language)
				c.String(http.StatusBadRequest, "Unsupported language: "+query.Language)
				return
			}

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
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}
			fp, err := os.OpenFile("/tmp/compiler/"+runningHash+"/"+lang.Language[query.Language].CodeFile, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}
			defer fp.Close()
			writer := bufio.NewWriter(fp)
			_, err = writer.WriteString(query.Code)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}
			writer.Flush()

			if len(lang.Language[query.Language].BuildCmd) == 0 {
				//c.String(http.StatusCreated, "This language hasn't build command and saved")
				c.String(http.StatusCreated, "")
				return
			}

			// Create container
			// TODO: Limit container spec
			fmt.Println("Create container")
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:           lang.Language[query.Language].DockerImage,
				WorkingDir:      "/workspace",
				Cmd:             lang.Language[query.Language].BuildCmd,
				NetworkDisabled: true,
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
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}

			// Start container
			err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}

			// Flow log of Stdout
			out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}
			rd := bufio.NewReader(out)
			c.Stream(func(w io.Writer) bool {
				line, _, err := rd.ReadLine()
				if len(line) >= 8 {
					w.Write(line[8:])
				} else {
					w.Write(line)
				}
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
			c.String(http.StatusBadRequest, err.Error())
		}
	})
	r.POST("/run", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		var query Run
		if err := c.BindJSON(&query); err == nil {
			// Make hash
			fmt.Println("Make hash")
			h := md5.New()
			io.WriteString(h, query.Language)
			io.WriteString(h, query.Code)
			runningHash := hex.EncodeToString(h.Sum(nil))
			fmt.Println("runningHash: " + runningHash)

			// Check exist of source code and builded image
			_, err = os.Stat("/tmp/compiler/" + runningHash + "/" + lang.Language[query.Language].CodeFile)
			if err != nil {
				// Check this language requires build command
				if len(lang.Language[query.Language].BuildCmd) == 0 {
					// Save code
					fmt.Println("Save code")
					if err := os.MkdirAll("/tmp/compiler/"+runningHash, 0755); err != nil {
						c.String(http.StatusInternalServerError, err.Error())
						fmt.Println(err.Error())
						return
					}
					fp, err := os.OpenFile("/tmp/compiler/"+runningHash+"/"+lang.Language[query.Language].CodeFile, os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						c.String(http.StatusInternalServerError, err.Error())
						fmt.Println(err.Error())
						return
					}
					defer fp.Close()
					writer := bufio.NewWriter(fp)
					_, err = writer.WriteString(query.Code)
					if err != nil {
						c.String(http.StatusInternalServerError, err.Error())
						fmt.Println(err.Error())
						return
					}
					writer.Flush()
				} else {
					c.String(http.StatusBadRequest, "Shoud /build before /run")
				}
			}

			// Create container
			// TODO: Limit container spec
			fmt.Println("Create container")
			fmt.Printf("%v\n", lang.Language[query.Language].RunCmd)
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:           lang.Language[query.Language].DockerImage,
				WorkingDir:      "/workspace",
				Cmd:             lang.Language[query.Language].RunCmd,
				NetworkDisabled: true,
				AttachStdin:     true,
				AttachStdout:    true,
				AttachStderr:    true,
				OpenStdin:       true,
				StdinOnce:       true,
				Tty:             false,
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
				c.String(http.StatusInternalServerError, err.Error())
				fmt.Println(err.Error())
				return
			}

			// Attach container
			stdin, err := cli.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
				Stream: true,
				Stdin:  true,
			})
			defer stdin.Close()
			if err != nil {
				fmt.Println(err.Error())
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			stdout, err := cli.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
				Stream: true,
				Stdout: true,
			})
			defer stdout.Close()
			if err != nil {
				fmt.Println(err.Error())
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			// Start container
			err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
			if err != nil {
				fmt.Println(err.Error())
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			// Put to Stdin
			fmt.Println(query.Stdin)
			stdin.Conn.Write([]byte(query.Stdin))
			stdin.CloseWrite()

			// Flow log of Stdout
			rd := bufio.NewReader(stdout.Reader)
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
			c.String(http.StatusBadRequest, err.Error())
		}
	})
	r.GET("/node", func(c *gin.Context) {
		containers, err := cli.ContainerList(ctx, options)
		if err != nil {
			log.Print(err)
			c.String(http.StatusInternalServerError, err.Error())
		}
		c.JSON(http.StatusOK, gin.H{
			"containers": containers,
		})
	})
	r.Run()
}
