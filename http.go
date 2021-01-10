package goat

import (
	"net/http"
	"strconv"
)

// HTTPRequest specializes the standard http request
type HTTPRequest struct {
	req *http.Request
}

// NewHTTPRequest create a new HTTPRequest from an http.Request
func NewHTTPRequest(r *http.Request) *HTTPRequest {
	return &HTTPRequest{req: r}
}

// Param get the query parameter with the specified name
func (r *HTTPRequest) Param(name string) string {
	keys, ok := r.req.URL.Query()[name]
	if !ok || len(keys[0]) < 1 {
		return ""
	}

	key := keys[0]
	return key
}

// IntParam get the query parameter with the specified name as int
func (r *HTTPRequest) IntParam(name string, defvalue int) int {
	s := r.Param((name))
	if s == "" {
		return defvalue
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return defvalue
	}

	return value
}

// FloatParam get the query parameter with the specified name as float64
func (r *HTTPRequest) FloatParam(name string, defvalue float64) float64 {
	s := r.Param((name))
	if s == "" {
		return defvalue
	}

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defvalue
	}

	return value
}

// BoolParam get the query parameter with the specified name as bool
func (r *HTTPRequest) BoolParam(name string) bool {
	s := r.Param((name))
	if s == "" {
		return false
	}

	value, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}

	return value
}
