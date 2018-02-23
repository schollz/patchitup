package patchitup

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Run will run the main program
func Run(port string) (err error) {
	os.MkdirAll(path.Join(UserHomeDir(), ".patchitup", "server"), 0755)

	defer log.Flush()
	// setup gin server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleWareHandler(), gin.Recovery())
	r.HEAD("/", func(c *gin.Context) { // handler for the uptime robot
		c.String(http.StatusOK, "OK")
	})
	r.POST("/lineNumbers", handlerLineNumbers) // returns hash and line numbers
	r.POST("/lineText", handlerLineText)       // returns hash and line text
	r.POST("/patch", handlerPatch)             // patch a file
	r.POST("/fileHash", handlerFileHash)       // get the hash of a file
	log.Infof("Running at http://0.0.0.0:" + port)
	err = r.Run(":" + port)
	return
}

func handlerFileHash(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		log.Infof("%s/%s upload: %s", sr.Username, sr.Filename, humanize.Bytes(uint64(c.Request.ContentLength)))

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

		message, err = Filemd5Sum(pathToFile)
		return
	}(c)
	if err != nil {
		message = err.Error()
	}

	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
	bSR, _ := json.Marshal(sr)
	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
	c.JSON(http.StatusOK, sr)
}

func handlerPatch(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		if len(sr.Patch) == 0 {
			err = errors.New("no patch supplied")
			return
		}
		log.Infof("%s/%s upload: %s", sr.Username, sr.Filename, humanize.Bytes(uint64(c.Request.ContentLength)))

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

		err = patchFile(pathToFile, sr.Patch)
		if err == nil {
			message = "applied patch"
		}
		return
	}(c)
	if err != nil {
		message = err.Error()
	}

	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
	bSR, _ := json.Marshal(sr)
	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
	c.JSON(http.StatusOK, sr)
}

func handlerLineText(c *gin.Context) {
	lines, message, err := func(c *gin.Context) (lines map[string][]byte, message string, err error) {
		lines = make(map[string][]byte)
		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		log.Infof("%s/%s upload: %s", sr.Username, sr.Filename, humanize.Bytes(uint64(c.Request.ContentLength)))

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
		allLines, err := getHashLines(pathToFile)
		if err != nil {
			return
		}

		// filter out by only the ones needed
		for line := range allLines {
			if _, ok := sr.MissingLines[line]; ok {
				lines[line] = allLines[line]
			}
		}
		message = "wrote lines"
		return
	}(c)
	if err != nil {
		message = err.Error()
	}

	sr := serverResponse{
		Message:      message,
		Success:      err == nil,
		HashLineText: lines,
	}
	bSR, _ := json.Marshal(sr)
	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
	c.JSON(http.StatusOK, sr)
}

func handlerLineNumbers(c *gin.Context) {
	lines, message, err := func(c *gin.Context) (lines map[string][]int, message string, err error) {
		lines = make(map[string][]int)
		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		log.Infof("%s/%s upload: %d", sr.Username, sr.Filename, c.Request.ContentLength)

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
		lines, err = getHashLineNumbers(pathToFile)
		if err != nil {
			return
		}
		message = "wrote lines"
		return
	}(c)
	if err != nil {
		message = err.Error()
	}
	sr := serverResponse{
		Message:         message,
		Success:         err == nil,
		HashLinenumbers: lines,
	}
	bSR, _ := json.Marshal(sr)
	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
	c.JSON(http.StatusOK, sr)
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
