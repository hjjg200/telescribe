package main

import (
    "bytes"
    "crypto/subtle"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    netUrl "net/url"
    "os"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"

    . "github.com/hjjg200/go-act"
)

type WebOptions struct {
    Durations []int `json:"durations"`
    FormatNumber string `json:"format.number"`
    FormatDateLong string `json:"format.date.long"`
    FormatDateShort string `json:"format.date.short"`
}

type WebAbstract struct { // webAbs
    ClientMap map[string/* fullName */] WebAbsClient `json:"clientMap"`
}
type WebAbsClient struct { // absCl
    CsvBox WebAbsCsvBox `json:"csvBox"`
    LatestMap map[string/* key */] WebAbsLatest `json:"latestMap"`
    ConfigMap map[string/* key */] MonitorConfig `json:"configMap"`
}
type WebAbsCsvBox struct { // csvBox
    Boundaries string `json:"boundaries"`
    DataMap map[string/* key */] string `json:"dataMap"`
}
type WebAbsLatest struct { // latest
    Timestamp int64 `json:"timestamp"`
    Value float64 `json:"value"`
    Status int `json:"status"`
}

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
            Logger.Warnln(
                "Setting a Random Password for",
                usr.Name + ":",
                plainPwd,
            )
        }
    }

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
            Logger.Warnln(rc)
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

    // Constants
    const (
        prefixMdtBox = "/monitorDataTableBox/"
        rgxMdtBox = prefixMdtBox + "([^/]+)/([^/]+)\\.csv"
    )

    // Functions
    type staticCache struct {
        name string
        modTime time.Time
        bytes []byte
    }
    staticCacheMap := make(map[string] staticCache)
    serveStatic := func(hctx HttpContext) {

        defer func() {
            r := recover()
            if r != nil {
                hctx.Writer.WriteHeader(404)
                Logger.Warnln(r)
            }
        }()

        fp := hctx.Request.URL.Path[1:]
        cache, ok := staticCacheMap[fp]
        if !ok {
            Assert(strings.HasPrefix(fp, "static/"), "Static file path must start with static/")
            Assert(strings.Contains("/" + fp, "/../") == false, "File path must not have .. in it")
            f, err := os.OpenFile(fp, os.O_RDONLY, 0644); Try(err)
            st, err := f.Stat(); Try(err)
            buf := bytes.NewBuffer(nil)
            io.Copy(buf, f)
            cache = staticCache{
                name: f.Name(),
                modTime: st.ModTime(),
                bytes: buf.Bytes(),
            }
            f.Close()
            staticCacheMap[fp] = cache
            Logger.Infoln("Cached a static file:", fp)
        }
        http.ServeContent(
            hctx.Writer, hctx.Request, cache.name, cache.modTime, bytes.NewReader(cache.bytes),
        )

    }

    hr.Get("/(index.html)?", func(hctx HttpContext) {
        hctx.Request.URL.Path = "/static/index.html"
        serveStatic(hctx)
    })
    hr.Get("/static/(.+)", serveStatic)
    hr.Get("/version", func(hctx HttpContext) {
        w := hctx.Writer
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprint(w, Version)
    })
    hr.Get("/options.json", func(hctx HttpContext) {
        w := hctx.Writer
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("Content-Type", "application/json")
        enc := json.NewEncoder(hctx.Writer)
        enc.Encode(srv.config.Web)
    })
    hr.Get("/abstract.json", func(hctx HttpContext) {
        w := hctx.Writer
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("Content-Type", "application/json")
        enc       := json.NewEncoder(w)
        clientMap := make(map[string/* fullName */] WebAbsClient)
        for fullName, mdMap := range srv.clientMonitorDataMap {
            efn     := netUrl.QueryEscape(fullName)
            csvBds  := prefixMdtBox + efn + "/_boundaries.csv"
            csvMap  := make(map[string/* key */] string)
            latest  := make(map[string/* key */] WebAbsLatest)
            cfgMap  := make(map[string/* key */] MonitorConfig)
            for key, md := range mdMap {
                mCfg := srv.getMonitorConfig(fullName, key)
                le   := md[len(md) - 1]
                csvMap[key] = prefixMdtBox + efn + "/" + netUrl.QueryEscape(key) + ".csv"
                latest[key]  = WebAbsLatest{
                    Timestamp: le.Timestamp,
                    Status:    mCfg.StatusOf(le.Value),
                    Value:     le.Value,
                }
                cfgMap[key] = mCfg
            }
            //
            clientMap[fullName] = WebAbsClient{
                CsvBox: WebAbsCsvBox{
                    Boundaries: csvBds,
                    DataMap: csvMap,
                },
                LatestMap: latest,
                ConfigMap: cfgMap,
            }
        }
        abs := WebAbstract{
            ClientMap: clientMap,
        }
        enc.Encode(abs)
    })

    // DATA RELATED

    parseMdtBox := func(hctx HttpContext) (string, string) {
        return hctx.Matches[1], hctx.Matches[2]
    }
    hr.Get(rgxMdtBox, func(hctx HttpContext) {
        w := hctx.Writer
        fullName, key := parseMdtBox(hctx)

        // CSV
        w.Header().Set("content-type", "text/csv")
        mdtBox := srv.clientMonitorDataTableBox[fullName]
        switch key {
        case "_boundaries":
            bds := mdtBox.Boundaries
            rd  := bytes.NewReader(bds)
            io.Copy(w, rd)
        default:
            mdt, ok := mdtBox.DataMap[key]
            Assert(ok, "Monitor data not found")
            rd := bytes.NewReader(mdt)
            io.Copy(w, rd)
        }
    })
    hr.Delete(rgxMdtBox, func(hctx HttpContext) {
        w := hctx.Writer
        //fullName, key := parseMdtBox(req)

        // JSON Response
        w.Header().Set("content-type", "application/json")
        
        /*
{
    "data": {
        "fullName": "...",
        "key": "..."
    },
    "meta": {
        "action": "deleteMonitorDataTableBox",
        "timestamp": ...,
        "executor": ... // http username
    }
}

{
    "error": {
        "code": 300,
        "message": "not found"
    },
    "meta": {
        "action": "deleteMonitorDataTableBox",
        "timestamp": ...,
        "executor": ... // http username
    }
}
        */
    })

    hr.Get("/test1", func(hctx HttpContext) {
        if !hctx.User.IsPermitted("get", "test1") {
            hctx.Writer.WriteHeader(403)
            return
        }
        pn := NewPermissionNode([]string{
            "action1",
            "action2.*",
            "action3.abc",
            "action3.def",
        })

        wr := func(s ...string) {
            fmt.Fprintln(hctx.Writer, s, pn.IsPermitted(s...))
        }
        wr("action1")
        wr("action1", "a")
        wr("action2")
        wr("action2", "a")
        wr("action3", "aaa")
        wr("action3", "bbb")

        fmt.Fprintln(hctx.Writer, "")
        fmt.Fprintln(hctx.Writer, permissionSplit("abcd.efgh.eghh"))
        fmt.Fprintln(hctx.Writer, permissionSplit("abcd.\"12..34\".eghh"))
        fmt.Fprintln(hctx.Writer, permissionSplit("abcd.\"1.2\\\"3.4\\\"56\".eghh"))
        fmt.Fprintln(hctx.Writer, permissionJoin([]string{"123", "4\"5.6", "67.89"}))

    })


    // API
    srv.registerAPIV1()

}

func(srv *Server) registerAPIV1() {

    hr := srv.httpRouter

    // API Version 1

    const (
        apiName = "api/v1"
        prefix  = "/" + apiName + "/"
        keyMdtBox = "monitorDataTableBox"
        keyClients = "clients"
        keyOptions = "options"
    )

    // Helpers
    isPermitted := func(hctx HttpContext, key string, params ...string) bool {
        return hctx.User.IsPermitted(apiName, hctx.Request.Method, key, params...)
    }

    // monitorDataTableBox
    //
    // GET, DELETE

    rgxMdtBox = prefix + keyMdtBox + "/([^/]+)/([^/]+)"
    hr.Get(rgxMdtBox, func(hctx HttpContext) {
        w := hctx.Writer
        r := hctx.Request
        
        fullName, mdKey := hctx.Matches[1], hctx.Matches[2]
        if !isPermitted(hctx, keyMdtBox, fullName, mdKey) {
            w.WriteHeader(403)
            return
        }

        w.Header().Set("Content-Type", "text/csv")
        mdtBox := srv.clientMonitorDataTableBox[fullName]
        switch mdKey {
        case "_boundaries":
            bds := mdtBox.Boundaries
            rd  := bytes.NewReader(bds)
            io.Copy(w, rd)
        default:
            mdt, ok := mdtBox.DataMap[mdKey]
            Assert(ok, "Monitor data not found")
            rd := bytes.NewReader(mdt)
            io.Copy(w, rd)
        }
    })
    hr.Delete(rgxMdtBox, func(hctx HttpContext) {

    })

    // Clients
    //
    // GET

    rgxClients := prefix + keyClients
    hr.Get(rgxClients, func(hctx HttpContext) {
        if !isPermitted(hctx, keyClients) {
            w.WriteHeader(403)
            return
        }

        
    })

}

////////////////
//-- Router --//
////////////////

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


//////////////
//-- USER --//
//////////////

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
    root  := nodes[0]
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
//  nodes := permissionSplit(p)
    root  := strings.ToLower(nodes[0])
    rest  := permissionJoin(nodes[1:])
    switch {
    case len(nodes) == 1:
        _, ok   := pn[root]
        _, wild := pn[permissionWildcard]
        return ok || wild
    case len(nodes) > 1:
        if child, ok := pn[root]; ok {
            return child.IsPermitted(rest)
        }
    }
    return false
}