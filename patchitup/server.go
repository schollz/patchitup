package patchitup

import (
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
	r.GET("/patch", handlerGetPatch)   // patch a file
	r.POST("/patch", handlerPostPatch) // post a patch
	r.GET("/list", handlerListPatches) // list patches
	r.GET("/hash", handlerFileHash)    // get the hash of a file
	log.Infof("Running at http://0.0.0.0:" + port)
	err = r.Run(":" + port)
	return
}

func authenticate(c *gin.Context) (p *Patchitup, sr serverRequest, err error) {
	err = c.ShouldBindJSON(&sr)
	if err != nil {
		return
	}
	p, err = New(Configuration{
		PublicKey:  sr.PublicKey,
		Signature:  sr.Signature,
		PathToFile: sr.Filename,
	})
	return
}

func handlerGetPatch(c *gin.Context) {
	patch, message, err := func(c *gin.Context) (patch patchFile, message string, err error) {
		p, sr, err := authenticate(c)
		if err != nil {
			return
		}

		patch, err = p.loadPatch(sr.Patch.EpochTime, sr.Patch.Hash)
		if err != nil {
			return
		}
		message = "got patch"
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

func handlerPostPatch(c *gin.Context) {
	patch, message, err := func(c *gin.Context) (patch patchFile, message string, err error) {
		p, sr, err := authenticate(c)
		if err != nil {
			return
		}

		err = p.savePatch(sr.Patch)
		if err != nil {
			return
		}
		message = "saved patch"
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
	patches, message, err := func(c *gin.Context) (patches []patchFile, message string, err error) {
		p, _, err := authenticate(c)
		if err != nil {
			return
		}

		patches, err = p.getPatches()
		if err != nil {
			return
		}
		message = "listed patches"
		return
	}(c)

	if err != nil {
		err = errors.Wrap(err, "problem saving patch")
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
		Patches: patches,
	}
	log.Debug(sr)
	c.JSON(http.StatusOK, sr)
}

func handlerFileHash(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		p, _, err := authenticate(c)
		if err != nil {
			return
		}

		message, err = p.latestHash()
		if err != nil {
			return
		}
		return
	}(c)

	if err != nil {
		err = errors.Wrap(err, "problem getting patches")
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
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
