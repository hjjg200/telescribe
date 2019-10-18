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
    "path"
    "strings"

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
    Timestamp int64
    Value float64
    Status int
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

    // Methods
    serveStatic := func(u string) {
        defer func() {
            r := recover()
            if r != nil {
                w.WriteHeader(404)
            }
        }()
        Assert(strings.HasPrefix(u, "/static/"), "Wrong static path")
        fp := u[1:]
        f, err := os.OpenFile(fp, os.O_RDONLY, 0644)
        Try(err)
        defer f.Close()
        st, err := f.Stat()
        Try(err)
        http.ServeContent(w, r, f.Name(), st.ModTime(), f)
    }

    url := r.URL.Path
    stripPrefix := func (s string) bool {
        if strings.HasPrefix(url, s) {
            url = url[len(s):]
            return true
        }
        return false
    }

    // Routes
    const (
        prefixMdtBox = "/monitorDataTableBox/"
    )
    switch {
    default:
        serveStatic(url)
        return
    case url == "/":
        serveStatic("/static/index.html")
    case url == "/abstract.json":

        w.Header().Set("content-type", "application/json")
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

    case url == "/options.json":

        w.Header().Set("content-type", "application/json")
        enc := json.NewEncoder(w)
        enc.Encode(srv.config.Web)

    case stripPrefix(prefixMdtBox):

        split := strings.Split(url, "/")
        Assert(len(split) == 2, "Wrong monitor data url")
        fullName, err := netUrl.QueryUnescape(split[0])
        Try(err)
        base, err     := netUrl.QueryUnescape(split[1])
        Try(err)

        // CSV
        Assert(path.Ext(base) == ".csv", "Non-csv request")
        key := base[:len(base) - 4]
        w.Header().Set("content-type", "text/csv")

        //
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

    }
}
