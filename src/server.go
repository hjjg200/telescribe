package main

import (
    "bufio"
    "bytes"
    "encoding/gob"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
    . "github.com/hjjg200/go-act"
)

const (
    dataCacheExt = ".cache"
    clientConfigWatchInterval = time.Second * 5
)

type Server struct { // srv
    config ServerConfig
    cachedExecutable []byte
    httpListener net.Listener
    authFingerprint string
    clientConfig ClientConfig
    clientConfigVersion map[string/* fullName */] string
    clientMonitorDataTableBox map[string/* fullName */] MonitorDataTableBox
    clientMonitorDataMap map[string/* fullName */] MonitorDataMap
    httpRouter *httpRouter
}

type ServerConfig struct { // srvCfg
    // General
    AuthPrivateKeyPath string `json:"authPrivateKeyPath"`
    ClientConfigPath string `json:"clientConfigPath"`
    // Http
    HttpUsername string `json:"http.username"`
    HttpPassword string `json:"http.password"` // lowercase sha256
    HttpCertFilePath string `json:"http.certFilePath"` // For TLS
    HttpKeyFilePath string `json:"http.keyFilePath"` // For TLS
    // Monitor
    DataCacheInterval int `json:"monitor.dataCacheInterval"` // (minutes)
    DataCacheDir string `json:"monitor.dataCacheDir"`
    MaxDataLength int `json:"monitor.maxDataLength"`
    GapThresholdTime int `json:"monitor.gapThresholdTime"` // (minutes)
    DecimationThreshold int `json:"monitor.decimationThreshold"`
    DecimationInterval int `json:"monitor.decimationInterval"` // (minutes)
    // Web
    Web WebOptions `json:"web"`
    // Network
    Bind string `json:"network.bind"`
    Port int `json:"network.port"`
    Tickrate int `json:"network.tickrate"` // (hz) How many times will the server process connections in a second (Hz)
    // Alarm
    WebhookUrl string `json:"alarm.webhookUrl"`
}

type ClientConfig struct { // clCfg
    Hosts map[string/* host */] map[string/* alias */] string `json:"hosts"`
    Roles map[string/* alias */] ClientRole `json:"roles"`
}

type ClientRole struct { // role
    MonitorConfigMap map[string/* raw key */] MonitorConfig `json:"monitorConfigMap"`
    MonitorInterval int `json:"monitorInterval"`
}

func (role ClientRole) Version() string {
    j, _ := json.Marshal(role)
    return fmt.Sprintf("%x", Sha256Sum(j))[:6]
}

var DefaultServerConfig = ServerConfig{
    // General
    AuthPrivateKeyPath: "./.serverAuth.priv",
    ClientConfigPath: "./clientConfig.json",
    // Http
    HttpUsername: "user",
    HttpPassword: "",
    HttpCertFilePath: "",
    HttpKeyFilePath: "",
    // Monitor
    DataCacheInterval: 1,
    DataCacheDir: "./serverCache.d",
    MaxDataLength: 43200, // 30 days for 1-minute interval
    GapThresholdTime: 15,
    DecimationThreshold: 1500,
    DecimationInterval: 10,
    // Web
    Web: DefaultWebOptions,
    // Network
    Bind: "127.0.0.1",
    Port: 1226,
    Tickrate: 60,
    // Alarm
    WebhookUrl: "",
}

var DefaultWebOptions = WebOptions{
    Durations: []int{3*3600, 12*3600, 3*86400, 7*86400, 30*86400},
    FormatNumber: "{e+0.2f}",
    FormatDateLong: "DD HH:mm",
    FormatDateShort: "MM/DD",
}

var DefaultClientConfig = ClientConfig{
    // Client Config
    Hosts: map[string/* host */] map[string/* alias */] string {
        "127.0.0.1": {
            "default": "example",
            "foo": "bar",
        },
    },
    Roles: map[string/* alias */] ClientRole{
        "example": {
            MonitorConfigMap: map[string/* raw key */] MonitorConfig{
                "cpu-usage": MonitorConfig{
                    FatalRange: "80:",
                    WarningRange: "50:",
                    Format: "{.1f}%",
                },
            },
            MonitorInterval: 60,
        },
        "bar": {
            MonitorConfigMap: map[string/* raw key */] MonitorConfig{
                "swap-usage": MonitorConfig{},
                "memory-usage": MonitorConfig{
                    Format: "Using {.0f}%",
                },
            },
            MonitorInterval: 60,
        },
    },
}

func NewServer() *Server {
    srv := &Server{}
    srv.clientMonitorDataMap = make(map[string/* fullName */] MonitorDataMap)
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

func (srv *Server) loadClientConfig() error {

    fn := srv.config.ClientConfigPath
    
    // Open file
    f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
    switch {
    case err != nil && !os.IsNotExist(err):
        // Unexpected error
        return err
    case os.IsNotExist(err):
        // Does not exist
        // + Save default config
        srv.clientConfig = DefaultClientConfig
        f2, err := os.OpenFile(fn, os.O_WRONLY | os.O_CREATE, 0600)
        if err != nil {
            return err
        }
        enc := json.NewEncoder(f2)
        enc.SetIndent("", "  ")
        err = enc.Encode(srv.clientConfig)
        if err != nil {
            return err
        }
        f.Close()
    default:
        // Exists
        dec := json.NewDecoder(f)
        cc  := ClientConfig{}
        err = dec.Decode(&cc)
        if err != nil {
            return err
        }
        srv.clientConfig = cc
    }

    // Version
    ccv := make(map[string] string)
    roleVersion := make(map[string] string)
    for host, roleMap := range srv.clientConfig.Hosts {
        for alias, role := range roleMap {
            fullName := formatFullName(host, alias)
            if _, ok := roleVersion[role]; !ok {
                // Assign version
                roleVersion[role] = srv.clientConfig.Roles[role].Version()
            }
            ccv[fullName] = roleVersion[role]
        }
    }
    srv.clientConfigVersion = ccv
    
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

func (srv *Server) Addr() string {
    return fmt.Sprintf("%s:%d", srv.config.Bind, srv.config.Port)
}

func (srv *Server) Start() (err error) {

    defer Catch(&err)

    // Config
    Try(srv.LoadConfig(flServerConfigPath))
    Logger.Infoln("Loaded Config")
    Try(srv.loadClientConfig())
    Logger.Infoln("Loaded Client Config")

    // Private Key
    Try(srv.checkAuthPrivateKey())
    Logger.Infoln("The fingerprint of the authentication public key is:")
    Logger.Infoln(sessionAuthPriv.PublicKey.Fingerprint())

    // Cache Executable
    Try(srv.cacheExecutable())
    Logger.Infoln("Cached Executable for Auto-Update")

    // Read Monitor Data Cache
    Try(srv.readCachedClientMonitorDataMap())
    Logger.Infoln("Read the Cached Monitored Items")

    // Ensure Directories
    Try(EnsureDirectory(srv.config.DataCacheDir))
    Logger.Infoln("Ensured Necessary Directories")

    // Network
    addr    := srv.Addr()
    ln, err := net.Listen("tcp", addr)
    Try(err)
    Logger.Infoln("Network Configured to Listen at", addr)

    // Flush cache thread
    go func() {
        for {
            time.Sleep(time.Minute * time.Duration(srv.config.DataCacheInterval))
            err := srv.CacheClientMonitorDataMap()
            if err != nil {
                Logger.Warnln(err)
                continue
            }
            Logger.Infoln("Cached Client Monitor Data")
        }
    }()
    Logger.Infoln("Started Monitor Data Caching Thread")

    // Client Config Version Update
    go func() {
        ccvp    := srv.config.ClientConfigPath
        st, _   := os.Stat(ccvp)
        lastMod := st.ModTime()
        for {func() {
            // Catch
            defer CatchFunc(Logger.Warnln)
            time.Sleep(clientConfigWatchInterval)
            // Mod Time Check
            st, err := os.Stat(ccvp)
            Try(err)
            // Changed
            if lastMod != st.ModTime() {
                err := srv.loadClientConfig()
                Try(err)
                Logger.Infoln("Reloaded Client Config")
                lastMod = st.ModTime()
            }
        }()}
    }()
    Logger.Infoln("Started Client Config Reloading Thread")

    // Chart-ready Data Preparing Thread
    go func() {
        for {func() {
            defer CatchFunc(Logger.Warnln)
            tBoxMap := make(map[string/* fullName */] MonitorDataTableBox)
            gthSec  := int64(srv.config.GapThresholdTime * 60) // To seconds
            for fullName, mdMap := range srv.clientMonitorDataMap {
                // MonitorData
                tsMap  := make(map[int64/* timestamp */] struct{})
                mdtMap := make(map[string/* key */] []byte)
                // Table-writing loop
                for key, md := range mdMap {
                    // Decimate monitor data
                    decimated := LttbMonitorData(
                        md, srv.config.DecimationThreshold,
                    )
                    // Write CSV(table)
                    table  := bytes.NewBuffer(nil)
                    prevTs := decimated[0].Timestamp
                    fmt.Fprint(table, "timestamp,value\n")
                    for _, each := range decimated {
                        ts := each.Timestamp
                        if ts - prevTs > gthSec {
                            // Put NaN (Gap)
                            midTs        := (ts + prevTs) / 2
                            tsMap[midTs]  = struct{}{}
                            fmt.Fprintf(table, "%d,NaN\n", midTs)
                        }
                        prevTs    = ts
                        tsMap[ts] = struct{}{}
                        fmt.Fprintf(table, "%d,%f\n", ts, each.Value)
                    }
                    // Assign Csv
                    mdtMap[key] = table.Bytes()
                }
                // Timestamps Slice
                i, tss := 0, make([]int64, len(tsMap))
                for t := range tsMap {
                    tss[i] = t
                    i++
                }
                sort.Sort(Int64Slice(tss))
                // Write boundaries table
                bds := bytes.NewBuffer(nil)
                fmt.Fprint(bds, "timestamp\n")
                fmt.Fprintf(bds, "%d\n", tss[0])
                for i := 1; i < len(tss); i++ {
                    prev := tss[i-1]
                    curr := tss[i]
                    if curr - prev > gthSec {
                        fmt.Fprintf(bds, "%d\n", prev)
                        fmt.Fprintf(bds, "%d\n", curr)
                    }
                }
                fmt.Fprintf(bds, "%d\n", tss[len(tss)-1])
                // Assign
                tBoxMap[fullName] = MonitorDataTableBox{
                    Boundaries: bds.Bytes(),
                    DataMap: mdtMap,
                }
            }
            // Assign
            srv.clientMonitorDataTableBox = tBoxMap
            Logger.Debugln("Chart data prepared")
            time.Sleep(time.Minute * time.Duration(srv.config.DecimationInterval))
        }()}
    }()
    Logger.Infoln("Started Data Decimation Thread")

    // Http
    go srv.startHttpServer()
    Logger.Infoln("Started HTTP Server")

    // Main
    Logger.Infoln("Successfully Started the Server")
    for {

        time.Sleep(time.Duration(1000.0 / float64(srv.config.Tickrate)) * time.Millisecond)

        // Connection
        conn, err := ln.Accept()
        if err != nil {
            Logger.Warnln(err)
            continue
        }

        go func() {
            defer CatchFunc(Logger.Warnln)

            host, _ := HostnameOf(conn)
            rd      := bufio.NewReader(conn)
            // Start line
            startLine, err := rd.ReadString('\n')
            Try(err)
            // Read rest bytes without advancing the reader
            rest, err := rd.Peek(rd.Buffered()) 
            Try(err)
            // Bytes that are already read
            already := append([]byte(startLine), rest...)

            switch {
            case strings.Contains(startLine, "HTTP"):
                // HTTP
                proxy, err := net.Dial("tcp", srv.HttpAddr())
                Try(err)
                startLine = strings.Trim(startLine, "\r\n")
                Logger.Infoln(host, startLine)
                proxy.Write(already)
                go connCopy(proxy, conn)
                go connCopy(conn, proxy)
            case strings.Contains(startLine, "TELESCRIBE"):
                // TELESCRIBE
                s := NewSession(conn)
                defer s.Close()
                // Prepend raw input
                s.PrependRawInput(bytes.NewReader(already))
                Try(srv.HandleSession(s))
            default:
            }
        }()
    }

    err = fmt.Errorf("Server terminated")
    return

}

func (srv *Server) readCachedClientMonitorDataMap() (err error) {

    defer Catch(&err)

    matches, err := filepath.Glob(
        srv.config.DataCacheDir + "/*" + dataCacheExt,
    )
    Try(err)

    for _, match := range matches {

        f, err2 := os.OpenFile(match, os.O_RDONLY, 0644)
        if err2 != nil {
            Logger.Warnln(err2)
            continue
        }

        var (
            fullName, key string
            cmp []byte
        )

        dec := gob.NewDecoder(f)
        dec.Decode(&fullName)
        dec.Decode(&key)
        dec.Decode(&cmp)
        f.Close()

        //
        md, err2 := DecompressMonitorData(cmp)
        if err2 != nil {
            Logger.Warnln(err2)
            continue
        }

        //
        _, ok := srv.clientMonitorDataMap[fullName]
        if !ok {
            srv.clientMonitorDataMap[fullName] = make(MonitorDataMap)
        }
        srv.clientMonitorDataMap[fullName][key] = md

    }

    return

}

func (srv *Server) CacheClientMonitorDataMap() (err error) {

    defer Catch(&err)

    for fullName, mdMap := range srv.clientMonitorDataMap {
        
        for key, md := range mdMap {
            
            // Buffer
            cmp, err2 := CompressMonitorData(md)
            if err2 != nil {
                Logger.Warnln(err2)
                continue
            }

            buf := bytes.NewBuffer(nil)
            enc := gob.NewEncoder(buf)
            enc.Encode(fullName)
            enc.Encode(key)
            enc.Encode(cmp)

            // Write file
            h  := fmt.Sprintf("%x", Sha256Sum([]byte(fullName + key)))
            fn := srv.config.DataCacheDir + "/" + h + dataCacheExt
            err2 = rewriteFile(fn, buf)
            if err2 != nil {
                Logger.Warnln(err2)
                continue
            }

        }

    }

    return

}

func (srv *Server) RecordValueMap(fullName string, timestamp int64, valMap map[string] interface{}) {

    //
    _, ok := srv.clientMonitorDataMap[fullName]
    if !ok {
        srv.clientMonitorDataMap[fullName] = make(MonitorDataMap)
    }

    //
    initialized := make(MonitorData, 0)
    fatalValues := make(map[string] float64)
    appendValue := func(key string, val float64) {

        short, ok := srv.clientMonitorDataMap[fullName][key]
        if !ok {
            short = initialized
        }

        // Trim Data
        if len(short) > srv.config.MaxDataLength {
            // Get MaxLength - 1 items
            start := len(short) - srv.config.MaxDataLength + 1
            short  = short[start:]
        }

        // Append
        srv.clientMonitorDataMap[fullName][key] = append(
            short,
            MonitorDatum{
                Timestamp: timestamp,
                Value: val,
            },
        )

        // Check Status
        st := srv.getMonitorConfig(fullName, key).StatusOf(val)
        if st == MonitorStatusFatal {
            fatalValues[key] = val
        }

    }

    //
    for key, val := range valMap {
        switch cast := val.(type) {
        case float64:
            appendValue(key, cast)
        case map[string] float64:
            for idx, subVal := range cast {
                longKey := FormatMonitorrKey(key, "", idx)
                appendValue(longKey, subVal)
            }
        }
    }

    //
    go func() {
        err := srv.sendAlarmWebhook(fullName, timestamp, fatalValues)
        if err != nil {
            Logger.Warnln("Failed to send webhook:", err)
        }
    }()

}

type alarmWebhook struct {
    FullName string `json:"fullName"`
    Timestamp int64 `json:"timestamp"`
    FatalValues map[string/* key */] float64 `json:"fatalValues"`
}
func (srv *Server) sendAlarmWebhook(fullName string, timestamp int64, fatalValues map[string] float64) error {

    //
    if len(fatalValues) == 0 {
        return nil
    }

    // Empty URL
    url := srv.config.WebhookUrl
    if url == "" {
        return nil
    }

    // Body
    aw := alarmWebhook{
        FullName: fullName,
        Timestamp: timestamp,
        FatalValues: fatalValues,
    }
    body, _ := json.Marshal(aw)

    // Request
    rsp, err := http.Post(
        url, "application/json", bytes.NewReader(body),
    )
    if err != nil {
        return err
    }

    // Check Response
    if rsp.StatusCode != 200 {
        // Webhook Receiver must reply back with 200
        return fmt.Errorf("status code %d", rsp.StatusCode)
    }

    return nil

}

func (srv *Server) getMonitorConfig(fullName, key string) (MonitorConfig) {

    aBase, aParam, aIdx := ParseMonitorrKey(key)
    // Base + param match
    var bpMatch MonitorConfig

    host, alias := parseFullName(fullName)
    role       := srv.clientConfig.Hosts[host][alias]
    mCfgMap    := srv.clientConfig.Roles[role].MonitorConfigMap
    for b, mCfg := range mCfgMap {
        bBase, bParam, bIdx := ParseMonitorrKey(b)
        if aBase == bBase && aParam == bParam {
            if aIdx == bIdx {
                // Exact match
                return mCfg
            }
            bpMatch = mCfg
        }
    }

    return bpMatch

}

func (srv *Server) HandleSession(s *Session) (err error) {

    defer Catch(&err)

    //
    host, err := s.RemoteHost()
    Try(err)
    roleMap, whitelisted := srv.clientConfig.Hosts[host]
    if !whitelisted {
        srvRsp := NewResponse("not-whitelisted")
        s.WriteResponse(srvRsp)
        return fmt.Errorf("%s [non-whitelisted] tried to establish a connection", host)
    }

    Logger.Infoln(host, "connected")
    clRsp, err := s.NextResponse()
    Try(err)

    // Role
    alias    := clRsp.String("alias")
    fullName := formatFullName(host, alias)
    role, ok := srv.clientConfig.Roles[roleMap[alias]]
    Assert(ok, "Client must have its role")

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
        srvRsp := NewResponse("version-mismatch")
        srvRsp.Set("executable", srv.cachedExecutable)
        s.WriteResponse(srvRsp)
        Logger.Warnln(host, "version mismatch, session terminated")
        return nil
    }

    // Main handling
    switch clRsp.Name() {
    case "hello":

        roleBytes, err := json.Marshal(role)
        Try(err)
        srvRsp := NewResponse("hello")
        srvRsp.Set("role", roleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[fullName])
        Logger.Infoln(host, "HELLO CLIENT")
        return s.WriteResponse(srvRsp)

    case "monitor-record":
        //
        valMap, ok := clRsp.Args()["valueMap"].(map[string] interface{})
        Assert(ok, "Malformed value map")
        timestamp := clRsp.Int64("timestamp")
        srv.RecordValueMap(fullName, timestamp, valMap)
    default:
        panic("Unknown response")
    }

    // Post Handling
    // # Config Version Check
    configVersion := clRsp.String("configVersion")
    switch configVersion {
    case "":
        // Version empty
        panic("Client response does not include config version")
    case srv.clientConfigVersion[fullName]:
        // Version match
    default:
        // Version mismatch
        roleBytes, err := json.Marshal(role)
        Try(err)
        srvRsp := NewResponse("reconfigure")
        srvRsp.Set("role", roleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[fullName])
        return s.WriteResponse(srvRsp)
    }

    // # OK
    srvRsp := NewResponse("ok")
    return s.WriteResponse(srvRsp)

}