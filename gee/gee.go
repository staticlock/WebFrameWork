package gee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type HandlerFunc func(*Context)

// Logger 默认日志中间件
func Logger() HandlerFunc {
	return func(c *Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()
		// Calculate resolution time
		log.Printf("我是默认日志:[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

// Engine 引擎,也就是启动器 实现了下面的接口，替代原先的 Handler
/*
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
*/
type Engine struct {
	*RouterGroup  // 引擎默认的分组 ，就是 / 用户不用给 / 分组。框架已经提供了
	router        *router
	groups        []*RouterGroup     // 存储所有的分组
	htmlTemplates *template.Template // 将所有的模板加载进内存
	funcMap       template.FuncMap   // 所有的自定义模板渲染函数。 本质是 map[string]any
}
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc // support middleware
	engine      *Engine       // all groups share an Engine instance
}

// New is the constructor of gee.Engine
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}
func (e *Engine) DefaultConfiguration() {
	e.UseMiddleware(Logger(), Recovery())
}
func (e *Engine) Group(prefix string) *RouterGroup {
	engine := e.engine
	newGroup := &RouterGroup{
		prefix: e.prefix + prefix, // 此时的 r.prefix应该是 ""
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// SetFuncMap 给用户分别提供了设置自定义渲染函数funcMap
func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

// LoadHTMLGlob 加载模板的方法。
func (e *Engine) LoadHTMLGlob(pattern string) {
	e.htmlTemplates = template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
}

//	func (e *Engine) addRoute(method string, path string, handler HandlerFunc) {
//		e.router.addRoute(method, path, handler)
//	}
func (r *RouterGroup) addRoute(method string, path string, handler HandlerFunc) {
	pattern := r.prefix + path // 如果不执行上面的Group 此时r.prefix为 ""
	log.Printf("Route %s-%s", method, pattern)
	r.engine.router.addRoute(method, pattern, handler)
}

func (r *RouterGroup) Get(path string, fn HandlerFunc) {
	r.addRoute("GET", path, fn) // 记得GET大写
}
func (r *RouterGroup) Post(path string, fn HandlerFunc) {
	r.addRoute("POST", path, fn) // 记得POST大写
}
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

// UseMiddleware 为某一组添加中间件
func (r *RouterGroup) UseMiddleware(middlewares ...HandlerFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}
func (r *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(r.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.StatusCode = http.StatusOK
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// Static serve static files
func (r *RouterGroup) Static(relativePath string, root string) {
	handler := r.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	// Register GET handlers
	r.Get(urlPattern, handler)
}

// 我们的引擎实现了 ServeHTTP ，然后将 Engine 扔进 http.ListenAndServe(addr, e) 中，会替代原先的 Handler，走我们自己的处理路由方法
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 每一个请求创建一个新的 Context 所有请求都是隔离的
	c := newContext(w, req)
	c.engine = e
	// 遍历所有的分组
	for _, group := range e.groups {
		if strings.HasPrefix(c.Path, group.prefix) {
			c.handlers = append(c.handlers, group.middlewares...)
		}
	}
	// 处理路由
	e.router.handle(c)
}
