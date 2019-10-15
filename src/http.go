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

type MonitorDataTables map[string] MDTClient
const MDTPrefix = "/monitorDataTables/"
type MDTClient struct {
    Timestamps []byte
    MonitorDataSlices map[string] []byte
}

type Abstract struct {
    Clients map[string] ABSClient `json:"clients"`
}
type ABSClient struct {
    Csv ABSCsv `json:"csv"`
    MonitorInfos map[string] MonitorInfo `json:"monitorInfos"`
    Latest map[string] ABSLatest `json:"latest"`
}
type ABSCsv struct {
    Timestamps string `json:"timestamps"`
    MonitorDataSlices map[string] string `json:"monitorDataSlices"`
}
type ABSLatest struct {
    Timestamp int64 `json:"timestamp"`
    Status int `json:"status"`
    Value float64 `json:"value"`
}

type GraphOptions struct {
    GapThresholdTime int `json:"gapThresholdTime"` // Two points whose time difference is greater than <threshold> seconds are considered as not connected, thus as having a gap in between
    Durations []int `json:"durations"`
    FormatNumber string `json:"format.number"`
    FormatDateLong string `json:"format.date.long"`
    FormatDateShort string `json:"format.date.short"`
}

func (srv *Server) startHttpServer() error {
    var err error
    srv.httpListener, err = net.Listen("tcp", "127.0.0.1:0") // Random port
    if err != nil {
        return err
    }

    certFile := srv.config.HttpCertFilePath
    keyFile := srv.config.HttpKeyFilePath
    httpServer := &http.Server{
        Addr: srv.HttpAddr(),
        Handler: srv,
    }

    // Password
    pwd := srv.config.HttpPassword
    if pwd == "" {
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
    un := srv.config.HttpUsername
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

    serveStatic := func(u string) {
        if strings.HasPrefix(u, "/static/") {
            fp := u[1:]
            f, err := os.OpenFile(fp, os.O_RDONLY, 0644)
            if err != nil {
                w.WriteHeader(404)
                return
            }
            st, err := f.Stat()
            if err != nil {
                w.WriteHeader(404)
                return
            }
            http.ServeContent(w, r, f.Name(), st.ModTime(), f)
            f.Close()
            return
        }
        w.WriteHeader(404)
    }

    url := r.URL.Path
    stripPrefix := func (s string) bool {
        if strings.HasPrefix(url, s) {
            url = url[len(s):]
            return true
        }
        return false
    }

    switch {
    default:
        serveStatic(url)
        return
    case url == "/":
        serveStatic("/static/index.html")
    case url == "/abstract.json":

        w.Header().Set("content-type", "application/json")
        enc := json.NewEncoder(w)
        clients := make(map[string] ABSClient)
        for fullName, mdsMap := range srv.clientMonitorData {
            mis     := srv.getMonitorInfos(fullName)
            efn     := netUrl.QueryEscape(fullName)
            csvTss  := MDTPrefix + efn + "/_timestamps.csv"
            csvMdss := make(map[string] string)
            latest  := make(map[string] ABSLatest)
            for key, mds := range mdsMap {
                mi := mis[key]
                le := mds[len(mds) - 1]
                csvMdss[key] = MDTPrefix + efn + "/" + netUrl.QueryEscape(key) + ".csv"
                latest[key]  = ABSLatest{
                    Timestamp: le.Timestamp,
                    Status: mi.StatusOf(le.Value),
                    Value: le.Value,
                }
            }
            //
            clients[fullName] = ABSClient{
                MonitorInfos: mis,
                Csv: ABSCsv{
                    Timestamps: csvTss,
                    MonitorDataSlices: csvMdss,
                },
                Latest: latest,
            }
        }
        abs := Abstract{
            Clients: clients,
        }
        enc.Encode(abs)

    case url == "/graphOptions.json":

        w.Header().Set("content-type", "application/json")
        enc := json.NewEncoder(w)
        enc.Encode(srv.config.Graph)

    case stripPrefix(MDTPrefix):

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

        switch key {
        case "_timestamps":
            tss := srv.monitorDataTables[fullName].Timestamps
            rd := bytes.NewReader(tss)
            io.Copy(w, rd)
        default:
            mds, ok := srv.monitorDataTables[fullName].MonitorDataSlices[key]
            Assert(ok, "Monitor data not found")
            rd := bytes.NewReader(mds)
            io.Copy(w, rd)
        }

    }
}
