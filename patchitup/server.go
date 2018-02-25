package patchitup

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var ServerDataFolder string

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
	r.POST("/patch/:username/:filename", handlerPostPatch) // post a patch
	r.GET("/patch/:username/:filename", handlerGetPatch)   // patch a file
	r.GET("/list/:username/:filename", handlerListPatches) // list patches
	r.GET("/hash/:username/:filename", handlerFileHash)    // get the hash of a file
	log.Infof("Running at http://0.0.0.0:" + port)
	err = r.Run(":" + port)
	return
}

func handlerGetPatch(c *gin.Context) {
	patch, message, err := func(c *gin.Context) (patch string, message string, err error) {
		username := c.Param("username")
		filename := c.Param("filename")

		p := New(username)
		// p.SetDataFolder(folder)
		patch, err = p.LoadPatch(filename)
		if err != nil {
			return
		}
		return
	}(c)

	if err != nil {
		err = errors.Wrap(err, "problem getting patch")
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
		Patch:   patch,
	}
	c.JSON(http.StatusOK, sr)
}

func handlerListPatches(c *gin.Context) {
	username := c.Param("username")
	filename := c.Param("filename")

	p := New(username)
	// p.SetDataFolder(folder)
	patches, err := p.getPatches(filename)
	message := fmt.Sprintf("got %d patches", len(patches))
	if err != nil {
		err = errors.Wrap(err, "problem getting patches")
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
		Patches: patches,
	}
	c.JSON(http.StatusOK, sr)
}

func handlerFileHash(c *gin.Context) {
	username := c.Param("username")
	filename := c.Param("filename")

	p := New(username)
	// p.SetDataFolder(folder)
	message, err := p.LatestHash(filename)
	if err != nil {
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
	c.JSON(http.StatusOK, sr)
}

func handlerPostPatch(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		username := c.Param("username")
		filename := c.Param("filename")

		var sr serverRequest
		err = c.ShouldBindJSON(&sr)
		if err != nil {
			return
		}
		if len(sr.Patch) == 0 {
			err = errors.New("no patch supplied")
			return
		}
		if len(username) == 0 {
			err = errors.New("no username supplied")
			return
		}
		if len(filename) == 0 {
			err = errors.New("no filename supplied")
			return
		}

		p := New(username)
		// p.SetDataFolder(folder)
		err = p.SavePatch(filename, sr.Patch)
		return
	}(c)
	if err != nil {
		message = err.Error()
	}

	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
	c.JSON(http.StatusOK, sr)
}

// func handlerGetPatch(c *gin.Context) {
// 	username := c.Param("username")
// 	hash := c.Param("hash")
// 	log.Debug(username, hash)
// 	c.String(http.StatusOK, "OK")
// }

// func handlerListPatches(c *gin.Context) {
// 	patches, message, err := func(c *gin.Context) (patches []patchFile, message string, err error) {
// 		var sr serverRequest
// 		err = c.ShouldBindJSON(&sr)
// 		if err != nil {
// 			return
// 		}
// 		log.Infof("%s/%s upload: %s", sr.Username, sr.Filename, humanize.Bytes(uint64(c.Request.ContentLength)))

// 		p := New("", sr.Username)
// 		patches, err = p.getPatches(sr.Filename)
// 		if err != nil {
// 			return
// 		}
// 		message = "got patches"
// 		return
// 	}(c)
// 	if err != nil {
// 		message = err.Error()
// 	}

// 	sr := serverResponse{
// 		Message: message,
// 		Success: err == nil,
// 		Patches: patches,
// 	}
// 	bSR, _ := json.Marshal(sr)
// 	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
// 	c.JSON(http.StatusOK, sr)
// }

// func handlerPostPatch(c *gin.Context) {
// 	message, err := func(c *gin.Context) (message string, err error) {
// 		var sr serverRequest
// 		err = c.ShouldBindJSON(&sr)
// 		if err != nil {
// 			return
// 		}
// 		if len(sr.Patch) == 0 {
// 			err = errors.New("no patch supplied")
// 			return
// 		}
// 		log.Infof("%s/%s upload: %s", sr.Username, sr.Filename, humanize.Bytes(uint64(c.Request.ContentLength)))

// 		// create cache directory
// 		pathToCacheServer := path.Join(utils.UserHomeDir(), ".patchitup", sr.Username)
// 		if !utils.Exists(pathToCacheServer) {
// 			os.MkdirAll(pathToCacheServer, 0755)
// 		}
// 		pathToFile := path.Join(pathToCacheServer, sr.Filename)
// 		err = ioutil.WriteFile(pathToFile, []byte(sr.Patch), 0755)
// 		if err == nil {
// 			message = "applied patch"
// 		}
// 		return
// 	}(c)
// 	if err != nil {
// 		message = err.Error()
// 	}

// 	sr := serverResponse{
// 		Message: message,
// 		Success: err == nil,
// 	}
// 	bSR, _ := json.Marshal(sr)
// 	log.Infof("download: %s", humanize.Bytes(uint64(len(bSR))))
// 	c.JSON(http.StatusOK, sr)
// }

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
