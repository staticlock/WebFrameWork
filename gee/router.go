package gee

import (
	"net/http"
	"strings"
)

// import (
//
//	"log"
//	"net/http"
//
// )
//
//	type router struct {
//		handlers map[string]HandlerFunc
//	}
//
//	func newRouter() *router {
//		return &router{
//			handlers: make(map[string]HandlerFunc),
//		}
//	}
//
//	func (r *router) addRoute(method string, path string, handler HandlerFunc) {
//		log.Printf("Route %4s - %s", method, path)
//		key := method + "-" + path
//		r.handlers[key] = handler
//	}
//
//	func (r *router) handle(c *Context) {
//		key := c.Method + "-" + c.Path
//		if handler, ok := r.handlers[key]; ok {
//			handler(c)
//		} else {
//			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
//		}
//	}
type router struct {
	roots   map[string]*node
	handler map[string]HandlerFunc // 路径和处理函数映射
}

func newRouter() *router {
	return &router{
		roots:   make(map[string]*node),
		handler: make(map[string]HandlerFunc),
	}
}

// Only one * is allowed 解析出路径的每一部分
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// 添加路由
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern //作为处理函数和请求方法+url映射的key
	_, ok := r.roots[method]      // 看看有没有该方法请求
	// 为第一次不同的请求创建Tire树
	if !ok {
		r.roots[method] = &node{}
	}
	r.handler[key] = handler
	r.roots[method].insert(pattern, parts, 0)
}
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	rootNode, ok := r.roots[method]
	if !ok {
		return nil, nil
	}
	n := rootNode.search(searchParts, 0)
	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}
	return nil, nil
}
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil { // 找到路由
		c.Params = params
		key := c.Method + "-" + n.pattern
		c.handlers = append(c.handlers, r.handler[key])
	} else { // 路由错误
		c.handlers = append(c.handlers, func(context *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	// 启动所有处理函数
	c.Next()
}
