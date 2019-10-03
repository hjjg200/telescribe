package main

import (
    "crypto/subtle"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "strings"
)

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

    switch url {
        // TODO http
        // TODO fatal status webhook
    default:
        serveStatic(url)
        return
    case "/":
        serveStatic("/static/index.html")
    case "/graphDataComposite.json":
        w.Header().Set("Content-Type", "application/json")
        w.Write(srv.graphDataCompositeJson)
    case "/clientMonitorStatus.json":
        cms := make(map[string] map[string] MonitorStatusSliceElem)
        for fullName, mdsMap := range srv.clientMonitorData {
            cms[fullName] = make(map[string] MonitorStatusSliceElem)
            for key, mds := range mdsMap {

                // TODO Avg interval
                mi := srv.getMonitorInfo(fullName, key)
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
