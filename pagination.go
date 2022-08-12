package goat

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pageable defines a request for a page of data
type Pageable struct {
	Index int `json:"index"`
	Size  int `json:"size"`
}

// Page defines a page of data
type Page struct {
	Pageable
	TotalElements int   `json:"totalElements"`
	TotalPages    int   `json:"totalPages"`
	Content       []any `json:"content"`
}

func NewPage() Page {
	p := Page{}
	p.Content = make([]any, 0)
	return p
}

func GetPageable(c *gin.Context) Pageable {
	p := Pageable{}
	p.Index, _ = strconv.Atoi(c.DefaultQuery("page", "0"))
	p.Size, _ = strconv.Atoi(c.DefaultQuery("size", "50"))
	return p
}

func (p *Page) Compute() {
	if p.Size == 0 {
		p.TotalPages = 0
	} else {
		p.TotalPages = int(math.Ceil(float64(p.TotalElements) / float64(p.Size)))
	}
}
