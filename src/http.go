package main

import (
    "bytes"
    "crypto/subtle"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "regexp"
    "strconv"
    "strings"
    "time"

    . "github.com/hjjg200/go-act"
)

func (srv *Server) startHttpServer() error {

    var err error
    srv.httpListener, err = net.Listen("tcp", "127.0.0.1:0") // Random port
    if err != nil {
        return err
    }

    certFile   := srv.config.HttpCertFilePath
    keyFile    := srv.config.HttpKeyFilePath
    httpServer := &http.Server{
        Addr: srv.HttpAddr(),
        Handler: srv,
    }
    srv.populateHttpRouter()

    // Password
    for i := range srv.config.HttpUsers {
        usr := &srv.config.HttpUsers[i]
        if usr.Password == "" {
            plainPwd := RandomAlphaNum(13)
            usr.Password = fmt.Sprintf("%x", Sha256Sum([]byte(plainPwd)))
            EventLogger.Warnln(
                "Setting a Random Password for",
                usr.Name + ":",
                plainPwd,
            )
        }
    }

    // TLS
    if certFile != "" && keyFile != "" {
        return httpServer.ServeTLS(srv.httpListener, certFile, keyFile)
    }

    return httpServer.Serve(srv.httpListener)
}

func (srv *Server) HttpAddr() string {
    return srv.httpListener.Addr().String()
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

    //
    defer func() {
        rc := recover()
        if rc != nil {
            w.WriteHeader(500)
            EventLogger.Warnln(rc)
        }
    }()

    // Auth
    w.Header().Set("WWW-Authenticate", "Basic realm=\"\"")
    hun, hplainPwd, ok := r.BasicAuth()
    
    var usr HttpUser
    for _, each := range srv.config.HttpUsers {
        if each.Name == hun {
            usr = each
            break
        }
    }
    authTest := subtle.ConstantTimeCompare(
        []byte(usr.Password), []byte(fmt.Sprintf("%x", Sha256Sum([]byte(hplainPwd)))),
    )
    if !ok || usr.Name == "" || authTest != 1 {
        w.WriteHeader(401)
        return
    }

    srv.httpRouter.Serve(HttpContext{
        Request: r, Writer: w, User: usr,
    })

}

func(srv *Server) populateHttpRouter() {

    hr := &httpRouter{
        routes: make(map[*regexp.Regexp] map[string] func(HttpContext)),
    }
    srv.httpRouter = hr

    // Functions
    type staticCache struct {
        name string
        modTime time.Time
        bytes []byte
    }
    staticCacheMap := make(map[string] staticCache)
    cacheHolder    := together.NewHoldGroup()
    serveStatic    := func(hctx HttpContext) {

        defer func() {
            r := recover()
            if r != nil {
                hctx.Writer.WriteHeader(404)
                EventLogger.Warnln(r)
            }
        }()

        fp        := hctx.Request.URL.Path[1:]
        cache, ok := staticCacheMap[fp]
        if !ok {func() {

            // Mutex
            cacheHolder.HoldAt(fp)
            defer cacheHolder.UnholdAt(fp)

            // Check again since a cache could have been created
            _, ok = staticCacheMap[fp]
            if ok { return }

            Assert(strings.HasPrefix(fp, "static/"), "Static file path must start with static/")
            Assert(strings.Contains("/" + fp, "/../") == false, "File path must not have .. in it")

            f, err  := os.OpenFile(fp, os.O_RDONLY, 0644); Try(err)
            st, err := f.Stat(); Try(err)
            buf     := bytes.NewBuffer(nil)

            io.Copy(buf, f)
            cache = staticCache{
                name: f.Name(),
                modTime: st.ModTime(),
                bytes: buf.Bytes(),
            }

            f.Close()
            staticCacheMap[fp] = cache
            EventLogger.Infoln("Cached a static file:", fp)

        }()}

        http.ServeContent(
            hctx.Writer, hctx.Request, cache.name, cache.modTime, bytes.NewReader(cache.bytes),
        )

    }

    // Static files
    hr.Get("/(index.html)?", func(hctx HttpContext) {
        hctx.Request.URL.Path = "/static/index.html"
        serveStatic(hctx)
    })
    hr.Get("/static/(.+)", serveStatic)

    // API
    srv.registerAPIV1()

}

func(srv *Server) registerAPIV1() {

    hr := srv.httpRouter

    // API Version 1

    const (
        apiName = "api/v1"
        prefix  = "/" + apiName + "/"
    )

    // Helpers
    respond := func(hctx HttpContext, key string, val interface{}) {

        w := hctx.Writer
        w.Header().Set("Content-Type", "application/json")

        obj    := map[string] interface{}{key: val}
        j, err := json.MarshalIndent(obj, "", "  ")

        if err != nil {
            w.WriteHeader(500)
            w.Write([]byte("{}"))
            return
        }

        w.Write(j)

    }
    isPermitted := func(hctx HttpContext, key string, params ...string) bool {
        args := []string{ apiName, hctx.Request.Method, key }
        args = append(args, params...)
        return hctx.User.IsPermitted(args...)
    }
    formatRgx := func(key string, argc int) string {
        return prefix + key + strings.Repeat("/([^/]+)", argc)
    }
    assertStatus := func(ok bool, st int) {
        if !ok { panic(st) }
    }
    catchStatus := func(hctx HttpContext) {
        r := recover()
        if st, ok := r.(int); r != nil && ok {
            hctx.Writer.WriteHeader(st)
        }
    }

    // clientInfoMap
    keyClientInfoMap := "clientInfoMap"
    rgxClientInfoMap := formatRgx(keyClientInfoMap, 0)
    hr.Get(rgxClientInfoMap, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Permission
        assertStatus(isPermitted(hctx, keyClientInfoMap), 403)

        ret  := make(ClientInfoMap)
        iMap := srv.clientConfig.InfoMap
        for clId, clInfo := range iMap {
            if !isPermitted(hctx, keyClientInfoMap, clId) {
                continue
            }
            ret[clId] = clInfo
        }

        // Respond
        respond(hctx, keyClientInfoMap, ret)
    })

    // clientRule
    keyClientRule := "clientRule"
    rgxClientRule := formatRgx(keyClientRule, 1)
    hr.Get(rgxClientRule, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Vars
        clId := hctx.Matches[1]
        // Permission
        assertStatus(isPermitted(hctx, keyClientRule, clId), 403)

        clInfo, ok := srv.clientConfig.InfoMap[clId]
        assertStatus(ok, 400)
        clRule := srv.clientConfig.RuleMap.Get(clInfo.Tags)

        // Respond
        respond(hctx, keyClientRule, clRule)
    })

    // clientItemStatus
    keyClItStat := "clientItemStatus"
    rgxClItStat := formatRgx(keyClItStat, 1)
    hr.Get(rgxClItStat, func(hctx HttpContext) {

        defer catchStatus(hctx)

        // Vars
        ret  := make(ClientItemStatusMap)
        clId := hctx.Matches[1]

        // Permission
        assertStatus(isPermitted(hctx, keyClItStat, clId), 403)

        mdMap, ok :=  srv.clientMonitorDataMap[clId]
        assertStatus(ok, 400)

        for mKey, mData := range mdMap {
            if !isPermitted(hctx, keyClItStat, clId, mKey) {
                continue
            }

            // Config
            mCfg, ok := srv.getMonitorConfig(clId, mKey)
            if !ok {
                EventLogger.Warnln("Monitor config for", mKey, "was not found")
            }

            le   := mData[len(mData) - 1]
            ret[mKey] = ClientItemStatus{
                Timestamp: le.Timestamp,
                Value:     le.Value,
                Status:    mCfg.StatusOf(le.Value),
                Constant:  mCfg.Constant,
            }
        }

        // Respond
        respond(hctx, keyClItStat, ret)
    })

    // monitorConfig
    keyMcfg := "monitorConfig"
    rgxMcfg := formatRgx(keyMcfg, 2)
    hr.Get(rgxMcfg, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Vars
        clId := hctx.Matches[1]
        mKey := hctx.Matches[2]

        // Permission
        assertStatus(isPermitted(hctx, keyMcfg, clId, mKey), 403)

        // Get Config
        cfg, ok := srv.getMonitorConfig(clId, mKey)
        assertStatus(ok, 400)

        // Respond
        respond(hctx, keyMcfg, cfg)

    })

    // monitorDataBoundaries
    keyMdb := "monitorDataBoundaries"
    rgxMdb := formatRgx(keyMdb, 1)
    hr.Get(rgxMdb, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Vars
        w    := hctx.Writer
        clId := hctx.Matches[1]
        // Permission
        assertStatus(isPermitted(hctx, keyMdb, clId), 403)

        box, ok := srv.clientMonitorDataTableBox[clId]
        assertStatus(ok, 400)

        // Respond
        w.Header().Set("Content-Type", "text/csv")
        bds := box.Boundaries
        rd  := bytes.NewReader(bds)
        io.Copy(w, rd)
    })

    // monitorDataTable
    keyMdt := "monitorDataTable"
    rgxMdt := formatRgx(keyMdt, 2)
    hr.Get(rgxMdt, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Vars
        w          := hctx.Writer
        clId, mKey := hctx.Matches[1], hctx.Matches[2]
        // Permission
        assertStatus(isPermitted(hctx, keyMdt, clId, mKey), 403)

        box, ok  := srv.clientMonitorDataTableBox[clId]
        assertStatus(ok, 400)

        mdt, ok := box.DataMap[mKey]
        assertStatus(ok, 400)

        // Respond
        w.Header().Set("Content-Type", "text/csv")
        rd := bytes.NewReader(mdt)
        io.Copy(w, rd)
    })
    hr.Delete(rgxMdt, func(hctx HttpContext) {
        
    })

    // webConfig
    keyWebCfg := "webConfig"
    rgxWebCfg := formatRgx(keyWebCfg, 0)
    hr.Get(rgxWebCfg, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Permission
        assertStatus(isPermitted(hctx, keyWebCfg), 403)

        // Respond
        respond(hctx, keyWebCfg, srv.config.Web)
    })

    // version
    keyVersion := "version"
    rgxVersion := prefix + keyVersion
    hr.Get(rgxVersion, func(hctx HttpContext) {
        defer catchStatus(hctx)

        // Permission
        assertStatus(isPermitted(hctx, keyVersion), 403)

        // Respond
        respond(hctx, keyVersion, Version)
    })

}

// ROUTER ---

type httpRouter struct {
    // [regexp] => [method] => route
    routes map[*regexp.Regexp] map[string] func(HttpContext)
}

var httpRouteRegexps = make(map[string] *regexp.Regexp)

type HttpRequest struct {
    Body *http.Request
    Writer http.ResponseWriter
    Matches []string
}

type HttpContext struct {
    Request *http.Request
    Writer http.ResponseWriter
    User HttpUser
    Matches []string
}

func(hr *httpRouter) Serve(hctx HttpContext) {
    r := hctx.Request
    for rgx, handlers := range hr.routes {
        matches := rgx.FindStringSubmatch(r.URL.Path)
        if matches == nil || matches[0] != r.URL.Path {
            continue
        }
        hctx.Matches = matches
        if handler, ok := handlers[r.Method]; ok {
            handler(hctx)
        }
        return
    }
}

func(hr *httpRouter) addRoute(m string, rstr string, h func(HttpContext)) error {

    // Find cached regexp
    rgx, ok := httpRouteRegexps[rstr]
    if !ok {
        var err error
        rgx, err = regexp.Compile(rstr)
        if err != nil {
            return err
        }
        // Cache regex
        httpRouteRegexps[rstr] = rgx
    }

    // Check existence
    if _, ok := hr.routes[rgx]; !ok {
        hr.routes[rgx] = make(map[string] func(HttpContext))
    }

    // Add a rotue
    m = strings.ToUpper(m) // e.g.) get => GET
    hr.routes[rgx][m] = h

    return nil

}

func(hr *httpRouter) Get(rstr string, h func(HttpContext)) { hr.addRoute("GET", rstr, h) }
func(hr *httpRouter) Head(rstr string, h func(HttpContext)) { hr.addRoute("HEAD", rstr, h) }
func(hr *httpRouter) Post(rstr string, h func(HttpContext)) { hr.addRoute("POST", rstr, h) }
func(hr *httpRouter) Put(rstr string, h func(HttpContext)) { hr.addRoute("PUT", rstr, h) }
func(hr *httpRouter) Delete(rstr string, h func(HttpContext)) { hr.addRoute("DELETE", rstr, h) }
func(hr *httpRouter) Options(rstr string, h func(HttpContext)) { hr.addRoute("OPTIONS", rstr, h) }
func(hr *httpRouter) Patch(rstr string, h func(HttpContext)) { hr.addRoute("PATCH", rstr, h) }


// USER ---

type HttpUser struct {
    Name string `json:"name"`
    Password string `json:"password"` // sha256 lowercase
    Permissions []string `json:"permissions"`
    pNode *PermissionNode
}

func(usr *HttpUser) IsPermitted(nodes ...string) bool {
    if usr.pNode == nil {
        usr.pNode = NewPermissionNode(usr.Permissions)
    }
    return usr.pNode.IsPermitted(nodes...)
}

type PermissionNode map[string] *PermissionNode

const (
    permissionSeparator = "."
    permissionWildcard = "*"
)

func permissionSplit(s string) []string {
    var (
        token string
        ret = make([]string, 0)
        quoted = false
        escaped = false
    )

    for i, c := range s {
        switch {
        case c == '\\':
            if escaped {
                token += string(c)
                escaped = false
            } else {
                escaped = true
            }
        case c == '"':
            if quoted {
                if escaped {
                    token += string(c)
                    escaped = false
                } else {
                    quoted = false
                }
            } else {
                quoted = true
            }
        case string(c) == permissionSeparator:
            if quoted {
                token += string(c)
            } else {
                ret = append(ret, token)
                token = ""
            }
        case i == len(s) - 1:
            token += string(c)
            ret = append(ret, token)
        default:
            token += string(c)
        }
    }
    return ret
}

func permissionJoin(p []string) string {
    for i := range p {
        if strings.Contains(p[i], "\"") {
            p[i] = strconv.Quote(p[i])
        }
    }
    return strings.Join(p, permissionSeparator)
}

func NewPermissionNode(p []string) *PermissionNode {
    n := make(PermissionNode)
    for _, l := range p { n.Add(l) }
    return &n
}

func(pn *PermissionNode) Add(p string) {
    nodes := permissionSplit(p)
    root  := strings.ToLower(nodes[0])
    rest  := permissionJoin(nodes[1:])

    child, ok := (*pn)[root]
    if !ok {
        child = NewPermissionNode(nil)
        (*pn)[root] = child
    }

    if len(nodes) > 1 {
        child.Add(rest)
    }
}

func(pn PermissionNode) IsPermitted(nodes ...string) bool {
    root    := strings.ToLower(nodes[0])
    _, wild := pn[permissionWildcard]
    
    switch {
    case wild:
        return true
    case len(nodes) == 1:
        _, ok := pn[root]
        return ok
    case len(nodes) > 1:
        if child, ok := pn[root]; ok {
            return child.IsPermitted(nodes[1:]...)
        }
    }
    return false
}