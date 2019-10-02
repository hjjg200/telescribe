package main

import (
    "bufio"
    "bytes"
    "encoding/gob"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "os"
    "path/filepath"
    "strings"
    "time"
    . "github.com/hjjg200/go-act"
)

const (
    monitorDataCacheExt = ".cache"
)

type Server struct {
    config ServerConfig
    httpListener net.Listener
    authFingerprint string

    clientMonitorData map[string] map[string] MonitorDataSlice
    graphDataComposite GrpahDataComposite
    graphDataCompositeJson []byte
    cachedExecutable []byte
}

type ServerConfig struct {
    // General
    AuthPrivateKeyPath string `json:"authPrivateKeyPath"`
        // TODO Log latest.log, 20190102.log ...
    // Http
    HttpUsername string `json:"http.username"`
    HttpPassword string `json:"http.password(sha256)"`
    HttpCertFilePath string `json:"http.certFilePath"` // For TLS
    HttpKeyFilePath string `json:"http.keyFilePath"` // For TLS
    // Monitor
    MonitorDataCacheInterval int `json:"monitor.dataCacheInterval(minutes)"`
    MonitorDataCacheDir string `json:"monitor.dataCacheDir"`
    GapThresholdTime int `json:"monitor.gapThresholdTime(minutes)"` // Two points whose time difference is greater than <threshold> seconds are considered as not connected, thus as having a gap in between
    MaxDataLength int `json:"monitor.maxDataLength"`
    GraphPointThreshold int `json:"monitor.graphThreshold"`
    GraphDecimationInterval int `json:"monitor.graphDecimationInterval(minutes)"`
    GraphGapPercent float64 `json:"monitor.graphGapPercent"`
    GraphMomentJSFormat string `json:"monitor.graphMomentJsFormat"`
    // Network
    Bind string `json:"network.bind"`
    Port int `json:"network.port"`
    Tickrate int `json:"network.tickrate(hz)"` // How many times will the server process connections in a second (Hz)
    // Client Config
    ClientAliases map[string] ClientAliasConfig `json:"client.aliases"`
    ClientRoles map[string] ClientRoleConfig `json:"client.roles"`
}

var DefaultServerConfig = ServerConfig{
    // General
    AuthPrivateKeyPath: "./.serverAuth.priv",
    // Http
    HttpUsername: "user",
    HttpPassword: "",
    HttpCertFilePath: "",
    HttpKeyFilePath: "",
    // Monitor
    MonitorDataCacheInterval: 1,
    MonitorDataCacheDir: "./serverCache.d",
    GapThresholdTime: 30,
    MaxDataLength: 43200, // 30 days for 1-minute interval
    GraphPointThreshold: 500,
    GraphDecimationInterval: 10,
    GraphGapPercent: 5.0,
    GraphMomentJSFormat: "MM/DD HH:mm",
    // Network
    Bind: "127.0.0.1",
    Port: 1226,
    Tickrate: 60,
    // Client Config
    ClientAliases: map[string] ClientAliasConfig{
        "127.0.0.1": {
            Alias: "foo",
            Comment: "This is an example.",
            Role: "example",
        },
    },
    ClientRoles: map[string] ClientRoleConfig{
        "example": {
            MonitorInfos: map[string] MonitorInfo{
                "general-cpuUsage": MonitorInfo{
                    FatalRange: "80:",
                    WarningRange: "50:",
                },
            },
            MonitorInterval: 60,
        },
    },
}

func NewServer() *Server {
    srv := &Server{}
    srv.clientMonitorData = make(map[string] map[string] MonitorDataSlice)
    return srv
}

func (srv *Server) LoadConfig(p string) error {

    // Load Default First
    srv.config = DefaultServerConfig

    f, err := os.OpenFile(p, os.O_RDONLY, 0644)
    if err != nil && !os.IsNotExist(err) {
        return err
    } else if err == nil {
        dec := json.NewDecoder(f)
        err = dec.Decode(&srv.config)
        if err != nil {
            return err
        }
        f.Close()
    }

    // Save config
    f2, err := os.OpenFile(p, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    enc := json.NewEncoder(f2)
    enc.SetIndent("", "  ")
    err = enc.Encode(srv.config)
    if err != nil {
        return err
    }
    f2.Close()
    return nil

}

func (srv *Server) cacheExecutable() error {
    
    f, err := os.OpenFile(executablePath, os.O_RDONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    buf := bytes.NewBuffer(nil)
    io.Copy(buf, f)

    srv.cachedExecutable = buf.Bytes()
    return nil

}

func (srv *Server) checkAuthPrivateKey() error {
    apk := srv.config.AuthPrivateKeyPath
    return LoadAuthPrivateKey(apk)
}

func (srv *Server) Start() (err error) {

    defer Catch(&err)

    // Config
    Try(srv.LoadConfig(flServerConfigPath))
    Logger.Infoln("Loaded Config")

    //
    Try(srv.checkAuthPrivateKey())
    Logger.Infoln("The fingerprint of the authentication public key is:")
    Logger.Infoln(sessionAuthPriv.PublicKey.Fingerprint())

    // Cache Executable
    Try(srv.cacheExecutable())
    Logger.Infoln("Cached Executable for Auto-Update")

    // Read Monitor Data Cache
    Try(srv.readCachedMonitordItems())
    Logger.Infoln("Read the Cached Monitored Items")

    // Ensure Directories
    Try(EnsureDirectory(srv.config.MonitorDataCacheDir))
    Logger.Infoln("Ensured Necessary Directories")

    // Network
    addr := fmt.Sprintf("%s:%d", srv.config.Bind, srv.config.Port)
    ln, err := net.Listen("tcp", addr)
    Try(err)
    Logger.Infoln("Network Configured to Listen at", addr)

    // Flush cache thread
    go func() {
        for {
            time.Sleep(time.Minute * time.Duration(srv.config.MonitorDataCacheInterval))
            goErr := srv.FlushCachedMonitoredItems()
            if goErr != nil {
                Logger.Warnln(goErr)
                continue
            }
            Logger.Infoln("Flushed Client MonitorData Cache")
        }
    }()
    Logger.Infoln("Started Monitor Data Caching Thread")

    // Graph-ready Data Preparing Thread
    go func() {
        // graphClientMonitorData
        for {
            ca := make(map[string] string) // Client aliases
            cmd := make(map[string] map[string] MonitorDataSlice) // Client monitor data
            for host, mdsMap := range srv.clientMonitorData {
                ca[host] = srv.config.ClientAliases[host].Alias
                cmd[host] = make(map[string] MonitorDataSlice)
                for key, mds := range mdsMap {
                    cmd[host][key] = LTTBMonitorDataSlice( // Decimate monitor data
                        mds, srv.config.GraphPointThreshold,
                   )
                }
            }

            srv.graphDataComposite = GrpahDataComposite{
                GapThresholdTime: srv.config.GapThresholdTime,
                GapPercent: srv.config.GraphGapPercent,
                MomentJSFormat: srv.config.GraphMomentJSFormat,
                ClientAliases: ca,
                ClientMonitorData: cmd,
            }
            var err2 error
            srv.graphDataCompositeJson, err2 = json.Marshal(srv.graphDataComposite)
            if err2 != nil {
                Logger.Warnln(err2)
            } else {
                Logger.Infoln("Cached Graph-ready Data")
            }
            time.Sleep(time.Minute * time.Duration(srv.config.GraphDecimationInterval))
        }
    }()
    Logger.Infoln("Started Data Decimation Thread")

    // Http
    go srv.startHttpServer()
    Logger.Infoln("Started HTTP Server")

    // Main
    Logger.Infoln("Successfully Started the Server")
    copyIO := func(dest, src net.Conn) {
        defer src.Close()
        defer dest.Close()
        io.Copy(dest, src)
    }
    for {

        time.Sleep(time.Duration(1000.0 / float64(srv.config.Tickrate)) * time.Millisecond)

        // Connection
        conn, err := ln.Accept()
        if err != nil {
            Logger.Warnln(err)
            continue
        }

        host, _ := HostnameOf(conn)
        rd := bufio.NewReader(conn)
        startLine, err := rd.ReadString('\n') // Start line
        if err != nil {
            Logger.Warnln(err)
            continue
        }

        rest, err := rd.Peek(rd.Buffered()) // Read rest bytes without advancing the reader
        if err != nil {
            Logger.Warnln(err)
            continue
        }

        already := append([]byte(startLine), rest...) // Bytes that are already read

        switch {
        case strings.Contains(startLine, "HTTP"):
            go func() {
                proxy, err2 := net.Dial("tcp", srv.httpListener.Addr().String())
                if err2 != nil {
                    Logger.Warnln(err)
                    return
                }
                startLine = strings.Trim(startLine, "\r\n")
                Logger.Infoln(host, startLine)
                proxy.Write(already)
                go copyIO(proxy, conn)
                go copyIO(conn, proxy)
            }()
        case strings.Contains(startLine, "TELESCRIBE"):
            go func() {
                s := NewSession(conn)
                defer s.Close()
                s.PrependRawInput(bytes.NewReader(already))
                srv.HandleSession(s)
            }()
        default:
            continue
        }

    }

    err = fmt.Errorf("Server terminated")
    return

}

func (srv *Server) readCachedMonitordItems() (err error) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf("%v", r)
        }
    }()

    matches, err := filepath.Glob(srv.config.MonitorDataCacheDir + "/*" + monitorDataCacheExt)
    if err != nil {
        return
    }

    for _, match := range matches {

        f, forErr := os.OpenFile(match, os.O_RDONLY, 0644)
        if forErr != nil {
            Logger.Warnln(forErr)
            continue
        }
        buf := bytes.NewBuffer(nil)
        io.Copy(buf, f)
        f.Close()

        //
        base := filepath.Base(match)
        ext := filepath.Ext(base)
        skeyString := base[:len(base) - len(ext)]
        skey, forErr := hex.DecodeString(skeyString)
        if forErr != nil {
            Logger.Warnln(forErr)
            continue
        }

        //
        var (
            host, key string
       )
        rd := bytes.NewReader(skey)
        dec := gob.NewDecoder(rd)
        dec.Decode(&host)
        dec.Decode(&key)

        //
        mds, forErr := DecompressMonitorDataSlice(buf.Bytes())
        if forErr != nil {
            Logger.Warnln(forErr)
            continue
        }

        //
        _, ok := srv.clientMonitorData[host]
        if !ok {
            srv.clientMonitorData[host] = make(map[string] MonitorDataSlice)
        }
        srv.clientMonitorData[host][key] = mds

    }

    return

}

func (srv *Server) FlushCachedMonitoredItems() (err error) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf("%v", r)
        }
    }()

    for host, mdsMap := range srv.clientMonitorData {
        
        for key, mds := range mdsMap {
            
            skey := bytes.NewBuffer(nil)
            enc := gob.NewEncoder(skey)
            enc.Encode(host)
            enc.Encode(key)
            fn := srv.config.MonitorDataCacheDir + "/" + fmt.Sprintf("%x", skey.Bytes()) + monitorDataCacheExt

            f, forErr := os.OpenFile(fn, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0644)
            if forErr != nil {
                Logger.Warnln(forErr)
                continue
            }

            cmp, forErr := CompressMonitorDataSlice(mds)
            if forErr != nil {
                Logger.Warnln(forErr)
                continue
            }

            f.Write(cmp)
            f.Close()

        }

    }

    return

}

func (srv *Server) RecordMonitorData(host string, md map[string] interface{}) {

    //
    _, ok := srv.clientMonitorData[host]
    if !ok {
        srv.clientMonitorData[host] = make(map[string] MonitorDataSlice)
    }

    //
    initialized := make([]MonitorDataSliceElem, 0)
    timestamp := time.Now().Unix()

    appendValue := func(key string, val float64) {
        short, ok := srv.clientMonitorData[host][key]
        if !ok {
            short = initialized
        }
        if len(short) > srv.config.MaxDataLength {
            // Get MaxLength - 1 items
            start := len(short) - srv.config.MaxDataLength + 1
            short = short[start:]
        }

        srv.clientMonitorData[host][key] = append(
            short,
            MonitorDataSliceElem{
                Value: val,
                Timestamp: timestamp,
            },
       )
    }

    for key, val := range md {
        switch cast := val.(type) {
        case float64:
            appendValue(key, cast)
        case map[string] float64:
            for subKey, subVal := range cast {
                longKey := fmt.Sprintf("%s[%s]", key, subKey)
                appendValue(longKey, subVal)
            }
        }
    }

}

func (srv *Server) getMonitorInfo(host, key string) (MonitorInfo) {

    aBase, aParam, aIdx := ParseMonitorrKey(key)
    var bpMatch MonitorInfo

    alias := srv.config.ClientAliases[host]
    role := srv.config.ClientRoles[alias.Role]
    for b, mi := range role.MonitorInfos {
        bBase, bParam, bIdx := ParseMonitorrKey(b)
        if aBase == bBase && aParam == bParam {
            if aIdx == bIdx {
                // Exact match
                return mi
            }
            bpMatch = mi
        }
    }

    return bpMatch

}

func (srv *Server) HandleSession(s *Session) (err error) {

    defer Catch(&err)

    //
    host, err := s.RemoteHost()
    Try(err)
    alias, whitelisted := srv.config.ClientAliases[host]
    if !whitelisted {
        srvRsp := NewResponse("not-whitelisted")
        s.WriteResponse(srvRsp)
        return fmt.Errorf("%s [non-whitelisted] tried to establish a connection", host)
    }

    role, ok := srv.config.ClientRoles[alias.Role]
    Assert(ok, "Client must have its role")
    Logger.Infoln(host, "connected")
    clRsp, err := s.NextResponse()
    Try(err)

    // Version Check
    ver := clRsp.String("version")
    switch ver {
    case "":
        // Verison empty
        panic("Client response does not include version")
    case Version:
        // Version match
    default:
        // Version mismatch
        Logger.Warnln(host, "VERSION MISMATCH")
        return srv.handleSessionVersionMismatch(s)
    }

    // Main handling
    switch clRsp.Name() {
    case "hello":

        roleBytes, err := json.Marshal(role)
        Try(err)
        srvRsp := NewResponse("hello")
        srvRsp.Set("role", roleBytes)
        Logger.Infoln(host, "HELLO CLIENT")
        return s.WriteResponse(srvRsp)

    case "monitor-data":

        //
        md, ok := clRsp.Args()["monitorData"].(map[string] interface{})
        Assert(ok, "Malformed monitor data")
        srv.RecordMonitorData(host, md)

        // OK
        srvRsp := NewResponse("ok")
        return s.WriteResponse(srvRsp)

    default:
        panic("Unknown response")
    }

}

func (srv *Server) handleSessionVersionMismatch(s *Session) error {
    srvRsp := NewResponse("version-mismatch")
    srvRsp.Set("executable", srv.cachedExecutable)
    return s.WriteResponse(srvRsp)
}