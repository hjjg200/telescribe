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
    if srv.config.HttpPassword == "" {
        plainPwd := RandomAlphaNum(13)
        srv.config.HttpPassword = fmt.Sprintf("%x", Sha256Sum([]byte(plainPwd)))
        Logger.Warnln("Empty HTTP Password! Setting a Random Password:", plainPwd)
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
    un  := srv.config.HttpUsername
    pwd := srv.config.HttpPassword
    w.Header().Set("WWW-Authenticate", "Basic realm=\"\"")
    hun, hplainPwd, ok := r.BasicAuth()
    authTest := func() int {
        return subtle.ConstantTimeCompare(
            []byte(pwd), []byte(fmt.Sprintf("%x", Sha256Sum([]byte(hplainPwd)))),
       )
    }
    if !ok || un != hun || authTest() != 1 {
        w.WriteHeader(401)
        return
    }

    srv.httpRouter.ServeHTTP(w, r)

}

////////////////
//-- Router --//
////////////////

type httpRouter struct {
    // [regexp] => [method] => route
    routes map[*regexp.Regexp] map[string] func(HttpRequest)
}

var httpRouteRegexps = make(map[string] *regexp.Regexp)

type HttpRequest struct {
    Body *http.Request
    Writer http.ResponseWriter
    Matches []string
}

func(srv *Server) populateHttpRouter() {

    hr := &httpRouter{
        routes: make(map[*regexp.Regexp] map[string] func(HttpRequest)),
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
    staticCacheMu  := sync.Mutex{}
    serveStatic := func(req HttpRequest) {

        defer func() {
            r := recover()
            if r != nil {
                req.Writer.WriteHeader(404)
                Logger.Warnln(r)
            }
        }()

        fp := req.Body.URL.Path[1:]
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
            staticCacheMu.Lock()
            staticCacheMap[fp] = cache
            staticCacheMu.Unlock()
            Logger.Infoln("Cached a static file:", fp)
        }
        http.ServeContent(req.Writer, req.Body, cache.name, cache.modTime, bytes.NewReader(cache.bytes))

    }

    hr.Get("/(index.html)?", func(req HttpRequest) {
        req.Body.URL.Path = "/static/index.html"
        serveStatic(req)
    })
    hr.Get("/static/(.+)", serveStatic)
    hr.Get("/version", func(req HttpRequest) {
        w := req.Writer
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprint(w, Version)
    })
    hr.Get("/options.json", func(req HttpRequest) {
        w := req.Writer
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("Content-Type", "application/json")
        enc := json.NewEncoder(req.Writer)
        enc.Encode(srv.config.Web)
    })
    hr.Get("/abstract.json", func(req HttpRequest) {
        w := req.Writer
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

    parseMdtBox := func(req HttpRequest) (string, string) {
        return req.Matches[1], req.Matches[2]
    }
    hr.Get(rgxMdtBox, func(req HttpRequest) {
        w := req.Writer
        fullName, key := parseMdtBox(req)

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
    hr.Delete(rgxMdtBox, func(req HttpRequest) {
        w := req.Writer
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

    // API Test
    arn := "api/test"
    ar := NewAPIRouter(arn)
    hr.Fallback("/" + arn + "/.+", func(req HttpRequest) {
        ar.Serve(req)
    })
    ar.Get("action1", func(arq APIRequest) {
        arq.Data(map[string] interface{} {
            "data1": "abcd",
            "arg1": arq.Args[0],
        })
    })

}

func(hr *httpRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    for rgx, handlers := range hr.routes {
        matches := rgx.FindStringSubmatch(r.URL.Path)
        if matches == nil || matches[0] != r.URL.Path {
            continue
        }
        hrq := HttpRequest{
            Body: r,
            Writer: w,
            Matches: matches,
        }
        if handler, ok := handlers[r.Method]; ok {
            handler(hrq)
        } else if fallback, ok := handlers[""]; ok {
            fallback(hrq)
        }
        return
    }
}

func(hr *httpRouter) addRoute(m string, rstr string, h func(HttpRequest)) error {

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
        hr.routes[rgx] = make(map[string] func(HttpRequest))
    }

    // Add a rotue
    m = strings.ToUpper(m) // e.g.) get => GET
    hr.routes[rgx][m] = h

    return nil

}

func(hr *httpRouter) Fallback(rstr string, h func(HttpRequest)) { hr.addRoute("", rstr, h) }
func(hr *httpRouter) Get(rstr string, h func(HttpRequest)) { hr.addRoute("GET", rstr, h) }
func(hr *httpRouter) Head(rstr string, h func(HttpRequest)) { hr.addRoute("HEAD", rstr, h) }
func(hr *httpRouter) Post(rstr string, h func(HttpRequest)) { hr.addRoute("POST", rstr, h) }
func(hr *httpRouter) Put(rstr string, h func(HttpRequest)) { hr.addRoute("PUT", rstr, h) }
func(hr *httpRouter) Delete(rstr string, h func(HttpRequest)) { hr.addRoute("DELETE", rstr, h) }
func(hr *httpRouter) Options(rstr string, h func(HttpRequest)) { hr.addRoute("OPTIONS", rstr, h) }
func(hr *httpRouter) Patch(rstr string, h func(HttpRequest)) { hr.addRoute("PATCH", rstr, h) }

