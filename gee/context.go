package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// H :使用interface{},是因为所有的数据都实现了空接口
// 给map[string]interface{}起了一个别名gee.H，构建JSON数据时，显得更简洁。
type H map[string]interface{}

// Context 使用 Context来封装响应和请求对象
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// 中间件业务 , 最后一个放着本次请求的处理函数（从路由的handler中取出来放进去）
	handlers []HandlerFunc
	// 记录中间件走到哪里了
	index  int
	engine *Engine
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    r,
		Path:   r.URL.Path,
		Method: r.Method,
		index:  -1,
	}
}
func (c *Context) Next() {
	c.index++
	lens := len(c.handlers)
	for ; c.index < lens; c.index++ {
		c.handlers[c.index](c)
	}
}
func (c *Context) End() {
	lens := len(c.handlers)
	c.index = lens
	for ; c.index < lens; c.index++ {
		c.handlers[c.index] = nil
	}
	c.Fail(500, "Request Error")
}
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// PostForm 得到请求体中的表单数据
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query 得到URL中的参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// Status 设置响应的状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 设置响应头数据
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 响应字符串数据
func (c *Context) String(code int, format string, values ...any) {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON 响应JSON数据
func (c *Context) JSON(code int, obj interface{}) error {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Status(code)
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = c.Writer.Write(jsonBytes)
	return err
}

// Data 响应字节数据
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// HTML 响应html
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}

// Fail 响应错误
func (c *Context) Fail(code int, msg string) {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	c.Writer.Write([]byte(msg))
}
