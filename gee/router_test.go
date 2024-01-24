package gee

import (
	"fmt"
	"testing"
)

func TestParsePattern(t *testing.T) {
	result := parsePattern("/")
	fmt.Println(result)
	newRouter().addRoute("GET", "/", func(c *Context) {

	})
}
