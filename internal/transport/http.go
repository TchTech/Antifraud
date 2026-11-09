package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	external "antifraud/internal/api/external"
	api "antifraud/internal/api/riskapi"
	"antifraud/internal/core"
	"github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"
)

type Risk struct{ Service *core.Service }

func reply(c *gin.Context, v any, e error) {
	if e != nil {
		status := 503
		if errors.Is(e, core.ErrConflict) {
			status = 409
		}
		c.JSON(status, gin.H{"error": http.StatusText(status)})
		return
	}
	c.JSON(200, v)
}
func (r Risk) Evaluate(c *gin.Context) {
	var b api.Evaluation
	if e := c.ShouldBindJSON(&b); e != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	v, e := r.Service.Evaluate(c.Request.Context(), b)
	reply(c, v, e)
}
func (r Risk) Confirm(c *gin.Context) {
	var b api.Confirmation
	if e := c.ShouldBindJSON(&b); e != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	v, e := r.Service.Confirm(c.Request.Context(), b)
	reply(c, v, e)
}
func (r Risk) State(c *gin.Context, account api.StateParamsAccount) {
	v, e := r.Service.State(c.Request.Context(), string(account))
	reply(c, v, e)
}
func base() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery(), func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10); c.Next() })
	return g
}
func RiskRouter(s *core.Service) *gin.Engine {
	g := base()
	spec, e := api.GetSwagger()
	if e != nil {
		panic(e)
	}
	g.Use(middleware.OapiRequestValidator(spec))
	api.RegisterHandlers(g, Risk{s})
	return g
}

type Gateway struct {
	URL    string
	Client *http.Client
}

func (g Gateway) proxy(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 1500*time.Millisecond)
	defer cancel()
	var body []byte
	var e error
	if c.Request.Body != nil {
		body, e = io.ReadAll(c.Request.Body)
		if e != nil {
			c.JSON(400, gin.H{"error": "body"})
			return
		}
	}
	req, e := http.NewRequestWithContext(ctx, c.Request.Method, strings.TrimRight(g.URL, "/")+c.Request.URL.RequestURI(), bytes.NewReader(body))
	if e != nil {
		reply(c, nil, e)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := g.Client.Do(req)
	if e != nil {
		reply(c, nil, e)
		return
	}
	defer resp.Body.Close()
	out, e := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if e != nil || !json.Valid(out) {
		c.JSON(503, gin.H{"error": "invalid scorer response"})
		return
	}
	c.Data(resp.StatusCode, "application/json", out)
}
func (g Gateway) Evaluate(c *gin.Context)                                   { g.proxy(c) }
func (g Gateway) Confirm(c *gin.Context)                                    { g.proxy(c) }
func (g Gateway) State(c *gin.Context, account external.StateParamsAccount) { g.proxy(c) }
func GatewayRouter(url string) *gin.Engine {
	g := base()
	spec, e := external.GetSwagger()
	if e != nil {
		panic(e)
	}
	g.Use(middleware.OapiRequestValidator(spec))
	external.RegisterHandlers(g, Gateway{url, &http.Client{Timeout: 1500 * time.Millisecond}})
	return g
}
