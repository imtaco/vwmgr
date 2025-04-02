package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

var (
	checkIAPFields = map[string]string{
		"/identity/accounts/prelogin": "email",    // prelogin to get parameters
		"/identity/connect/token":     "username", // real login
		// TODO: change pwd
	}
)

type appArgs struct {
	BindAddr    string `long:"bind_addr" env:"BIND_ADDR" default:":8080"`
	UpStreamURL string `long:"up_stream_url" env:"UP_STREAM_URL"`
}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal("err:", err)
	}

	// TODO: basic args validation
	remote, err := url.Parse(args.UpStreamURL)
	if err != nil {
		log.Fatalf("fail to parse upstream url %v", err)
	}

	// disalbe logs
	gin.DefaultWriter = io.Discard
	g := gin.Default()

	// for health check of LB or k8s
	g.GET("/_healthz", func(c *gin.Context) {})

	notAccept := func(c *gin.Context) {
		c.JSON(http.StatusNotAcceptable, gin.H{"error": "illegal operation"})
	}

	g.GET("/login", func(c *gin.Context) {
		// for auto fill email in login form
		email := getIAPHeader(c)
		if email == "" {
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, "/#login?email="+email)
	})
	g.GET("/api/devices/knowndevice", func(c *gin.Context) {
		c.String(http.StatusOK, "false")
	})

	// forbid operations
	g.DELETE("/api/accounts", notAccept)                  // delete account
	g.PUT("/api/accounts/profile", notAccept)             // change name
	g.POST("/api/organizations/:org_id/leave", notAccept) // leave org

	proxy := httputil.NewSingleHostReverseProxy(remote)

	g.NoRoute(func(c *gin.Context) {
		// Verify whether the field matches the IAM email if necessary
		if field, ok := checkIAPFields[c.Request.URL.Path]; ok {
			if (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut) &&
				!checkIAP(c, field) {
				return
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	g.Run(args.BindAddr)
}

func checkIAP(c *gin.Context, field string) bool {
	email := getIAPHeader(c)
	if email == "" {
		return false
	}

	payload, err := parseRequestToMap(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	if v, ok := payload[field]; ok && v != email {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "illegal operation"})
		return false
	}
	return true
}

func getIAPHeader(c *gin.Context) string {
	// Get the IAP user email header
	emailHeader := c.GetHeader("X-Goog-Authenticated-User-Email")
	if emailHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "IAP header missing"})
		return ""
	}

	// Extract the email part: accounts.google.com:email@example.com → email@example.com
	parts := strings.SplitN(emailHeader, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email header format"})
		return ""
	}
	return parts[1]
}

func parseRequestToMap(c *gin.Context) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, errors.Errorf("read body failed: %v", err)
	}
	// reset body for reverse proxy
	bodyReader := bytes.NewReader(bodyBytes)
	c.Request.Body = io.NopCloser(bodyReader)

	switch c.ContentType() {
	case "application/json":
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return nil, nil
		}
	case "application/x-www-form-urlencoded", "multipart/form-data":
		c.Request.ParseForm()
		for key, values := range c.Request.PostForm {
			if len(values) > 1 {
				data[key] = values
			} else {
				data[key] = values[0]
			}
		}
	default:
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Unsupported Content-Type"})
		return nil, nil
	}

	// reuse body for reverse proxy
	bodyReader.Seek(0, io.SeekStart)
	return data, nil
}
