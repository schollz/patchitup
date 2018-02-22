package patchitup

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func init() {
	os.MkdirAll(path.Join(UserHomeDir(), ".patchitup", "server"), 0755)
}

// Run will run the main program
func Run(port string) (err error) {
	defer log.Flush()
	// setup gin server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleWareHandler(), gin.Recovery())
	r.HEAD("/", func(c *gin.Context) { // handler for the uptime robot
		c.String(http.StatusOK, "OK")
	})
	r.POST("/lines", handlerLines)
	log.Infof("Running at http://0.0.0.0:" + port)
	err = r.Run(":" + port)
	return
}

func handlerLines(c *gin.Context) {
	lines, message, err := func(c *gin.Context) (lines map[string][]int, message string, err error) {
		lines = make(map[string][]int)
		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		// create cache directory
		if !Exists(path.Join(pathToCacheServer, sr.Username)) {
			os.MkdirAll(path.Join(pathToCacheServer, sr.Username), 0755)
		}
		pathToFile := path.Join(pathToCacheServer, sr.Username, sr.Filename)
		if !Exists(pathToFile) {
			message = "created new file"
			newFile, err2 := os.Create(pathToFile)
			if err2 != nil {
				err = errors.Wrap(err2, "problem creating file")
				return
			}
			newFile.Close()
			return
		}

		// file exists, read it line by line
		file, err := os.Open(pathToFile)
		if err != nil {
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			h := HashSHA256(scanner.Bytes())
			if _, ok := lines[h]; !ok {
				lines[h] = []int{}
			}
			lines[h] = append(lines[h], lineNumber)
		}
		message = "wrote lines"
		return
	}(c)
	if err != nil {
		message = err.Error()
	}

	response := serverResponse{
		Message:         message,
		Success:         err == nil,
		HashLinenumbers: lines,
	}
	bResponse, err := json.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("response: %+v", response)
	c.Data(http.StatusOK, "application/json", bResponse)
}

func middleWareHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		// Add base headers
		addCORS(c)
		// Run next function
		c.Next()
		// Log request
		log.Infof("%v %v %v %s", c.Request.RemoteAddr, c.Request.Method, c.Request.URL, time.Since(t))
	}
}

func addCORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
}

func contentType(filename string) string {
	switch {
	case strings.Contains(filename, ".css"):
		return "text/css"
	case strings.Contains(filename, ".jpg"):
		return "image/jpeg"
	case strings.Contains(filename, ".png"):
		return "image/png"
	case strings.Contains(filename, ".js"):
		return "application/javascript"
	case strings.Contains(filename, ".xml"):
		return "application/xml"
	}
	return "text/html"
}
