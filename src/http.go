package main

import (
    "crypto/subtle"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    netUrl "net/url"
    "os"
    "path"
    "strconv"
    "strings"

    . "github.com/hjjg200/go-act"
)

type GDCMonitorData struct {
    Format string `json:"format"`
    Status int `json:"status"`
    LastValue float64 `json:"lastValue"`
}
type GDCClient struct {
    MonitorData map[string] GDCMonitorData `json:"monitorData"`
}
type GDCOptions struct {
    GapThresholdTime int `json:"gapThresholdTime"`
    Durations []int `json:"durations"`
}
type GraphDataCompositeV2 struct {
    Clients map[string] GDCClient `json:"clients"`
    Options GDCOptions `json:"options"`
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

    switch {
    default:
        serveStatic(url)
        return
    case url == "/monitorData.csv":

        Logger.Debugln("monitorData start")
        w.Header().Set("content-type", "text/csv")
        fmt.Fprint(w, "FullName,Key,Timestamp,Value\n")
        for fullName, mdMap := range srv.graphDataComposite.ClientMonitorData {
            fullName = strconv.Quote(fullName)
            for key, md := range mdMap {
                key = strconv.Quote(key)
                for _, elem := range md {
                    fmt.Fprintf(w, "%s,%s,%d,%f\n", fullName, key, elem.Timestamp, elem.Value)
                }
            }
        }

    case strings.HasPrefix(url, "/monitorData/"):

        Logger.Debugln("monitorData start")
        split := strings.Split(url[len("/monitorData/"):], "/")
        Assert(len(split) == 2, "Wrong monitor data url")
        fullName, err := netUrl.QueryUnescape(split[0))
        Try(err)
        key, err := netUrl.QueryUnescape(split[1])
        Try(err)

        mds, ok := srv.graphDataComposite.ClientMonitorData[fullName][key]
        Assert(ok, "Monitor data not found")

        //
        gtht := srv.config.GapThresholdTime * 60
        lt := mds[0].Timestamp
        w.Header().Set("content-type", "text/csv")
        fmt.Fprint(w, "Timestamp,Value\n")
        for _, mde := range mds {
            if mde.Timestamp - lt > gtht {
                avg := (float64(mde.Timestamp) + float64(lt)) / 2.0
                fmt.Fprintf(w, "%f,NaN", avg)
            }

            lt = mde.Timestamp
            fmt.Fprint(w, "%d,%f\n", mde.Timestamp, mde.Value)
        }
        Logger.Debugln("monitorData end")

    case url == "/graphDataCompositeV2.json":

        // Clients
        clients := make(map[string] GDCClient)
        for fullName, mdsMap := range srv.clientMonitorData {
            gdcMd := make(map[string] GDCMonitorData)
            for key, mds := range mdsMap {
                mi := srv.getMonitorInfo(fullName, key)
                vl := mds[len(mds) - 1].Value
                st := mi.StatusOf(vl)
                gdcMd[key] = GDCMonitorData{
                    Status: st,
                    Format: mi.Format,
                    LastValue: vl,
                }
            }
            clients[fullName] = GDCClient{
                MonitorData: gdcMd,
            }
        }

        // Form
        gdc := GraphDataCompositeV2{
            Clients: clients,
            Options: GDCOptions{
                GapThresholdTime: srv.config.GapThresholdTime,
                Durations: []int{3*3600, 12*3600, 3*86400, 7*86400, 30*86400},
            },
        }

        // Write
        w.Header().Set("Content-Type", "application/json")
        enc := json.NewEncoder(w)
        Try(enc.Encode(gdc))

    case url == "/":
        serveStatic("/static/index.html")
    
    case url == "/graphDataComposite.json":
        w.Header().Set("Content-Type", "application/json")
        w.Write(srv.graphDataCompositeJson)
    case url == "/clientMonitorStatus.json":
        cms := make(map[string] map[string] MonitorStatusSliceElem)
        for fullName, mdsMap := range srv.clientMonitorData {
            cms[fullName] = make(map[string] MonitorStatusSliceElem)
            for key, mds := range mdsMap {

                // TODO Avg interval
                mi := srv.getMonitorInfo(fullName, key)
                Logger.Debugln(mi, fullName, key)
                if len(mds) == 0 {
                    continue
                }
                last := mds[len(mds) - 1]
                cms[fullName][key] = MonitorStatusSliceElem{
                    Timestamp: last.Timestamp,
                    Value: last.Value,
                    Status: mi.StatusOf(last.Value),
                }
            }
        }

        w.Header().Set("Content-Type", "application/json")
        enc := json.NewEncoder(w)
        enc.Encode(cms)
    }
}
