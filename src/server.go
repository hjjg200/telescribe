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
    "os/signal"
    "path/filepath"
    "sort"
    "strings"
    "syscall"
    "time"
    "github.com/hjjg200/go-together"

    . "github.com/hjjg200/go-act"
)

const (
    dataCacheExt              = ".cache"
    clientConfigWatchInterval = time.Second * 5
)

// SERVER CONFIG ---

type ServerConfig struct { // srvCfg
    // General
    AuthPrivateKeyPath  string `json:"authPrivateKeyPath"`
    ClientConfigPath    string `json:"clientConfigPath"`
    // Http
    HttpUsers           []HttpUser `json:"http.users"`
    HttpCertFilePath    string     `json:"http.certFilePath"` // For TLS
    HttpKeyFilePath     string     `json:"http.keyFilePath"` // For TLS
    // Monitor
    DataCacheInterval   int    `json:"monitor.dataCacheInterval"` // (minutes)
    DataCacheDir        string `json:"monitor.dataCacheDir"`
    MaxDataLength       int    `json:"monitor.maxDataLength"`
    GapThresholdTime    int    `json:"monitor.gapThresholdTime"` // (minutes)
    DecimationThreshold int    `json:"monitor.decimationThreshold"`
    DecimationInterval  int    `json:"monitor.decimationInterval"` // (minutes)
    // Web
    Web                 WebConfig `json:"web"`
    // Network
    Bind                string `json:"network.bind"`
    Port                int    `json:"network.port"`
    Tickrate            int    `json:"network.tickrate"` // (hz)
    // Alarm
    WebhookUrl          string `json:"alarm.webhookUrl"`
}

var DefaultServerConfig = ServerConfig{
    // General
    AuthPrivateKeyPath: "./.serverAuth.priv",
    ClientConfigPath:   "./clientConfig.json",
    // Http
    HttpUsers: []HttpUser{
        HttpUser{
            Name:     "user1",
            Password: "",
            Permissions: []string{
                "api/v1.get.*",
            },
        },
    },
    HttpCertFilePath:    "",
    HttpKeyFilePath:     "",
    // Monitor
    DataCacheInterval:   1,
    DataCacheDir:        "./serverCache.d",
    MaxDataLength:       43200, // 30 days for 1-minute interval
    GapThresholdTime:    15,
    DecimationThreshold: 1500,
    DecimationInterval:  10,
    // Web
    Web:                 DefaultWebConfig,
    // Network
    Bind:                "0.0.0.0",
    Port:                1226,
    Tickrate:            60,
    // Alarm
    WebhookUrl:          "",
}


// CLIENT CONFIG ---

type ClientConfig struct { // clCfg
    ClientMap map[string/* clId */] ClientInfo    `json:"clientMap"`
    RoleMap                         ClientRoleMap `json:"roleMap"`
}

var DefaultClientConfig = ClientConfig{
    // ClientMap
    ClientMap: map[string/* clientId */] ClientInfo{
        "example-01": ClientInfo{
            Host:  "127.0.0.1",
            Alias: "example",
            Role:  "foo bar",
        },
    },
    // Roles
    RoleMap: ClientRoleMap{
        "foo": {
            MonitorConfigMap: map[MonitorKey] MonitorConfig{
                "cpu-usage": MonitorConfig{
                    FatalRange:   "80:",
                    WarningRange: "50:",
                    Format:       "{.1f}%",
                },
                "memory-usage": MonitorConfig{},
            },
            MonitorInterval: 60,
        },
        "bar": {
            MonitorConfigMap: map[MonitorKey] MonitorConfig{
                "swap-usage":   MonitorConfig{},
                "memory-usage": MonitorConfig{
                    Format: "Using {.f}%",
                },
            },
            MonitorInterval: 60,
        },
    },
}


// SERVER ---

type Server struct { // srv
    config                    ServerConfig
    cachedExecutable          []byte
    httpListener              net.Listener
    httpRouter                *httpRouter
    authFingerprint           string
    clientConfig              ClientConfig
    clientConfigVersion       map[string/* clientId */] string
    clientMonitorDataTableBox map[string/* clientId */] MonitorDataTableBox
    clientMonitorDataMap      map[string/* clientId */] MonitorDataMap
}

func NewServer() *Server {
    srv := &Server{
        clientMonitorDataMap: make(map[string/* clientId */] MonitorDataMap),
    }
    return srv
}

func (srv *Server) LoadConfig(p string) (err error) {

    // Catch
    defer Catch(&err)

    // Load default first
    srv.config = DefaultServerConfig

    // Open the config file
    f, err := os.OpenFile(p, os.O_RDONLY, 0644)
    if err != nil && !os.IsNotExist(err) {
        // Unexpected error
        panic(err)
    } else if err == nil {
        // File exists
        dec := json.NewDecoder(f)
        Try(dec.Decode(&srv.config))
        Try(f.Close())
    }

    // Save config
    f2, err := os.OpenFile(p, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0644)
    Try(err)
    enc := json.NewEncoder(f2)
    enc.SetIndent("", "  ")
    Try(enc.Encode(srv.config))
    Try(f2.Close())
    return nil

}

func(srv *Server) loadClientConfig() (err error) {

    // Catch
    defer CatchFunc(&err, Logger.Warnln)

    // File path
    fn := srv.config.ClientConfigPath
    
    // Open file
    f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
    switch {
    case err != nil && !os.IsNotExist(err): // Unexpected error
        panic(err)

    case os.IsNotExist(err): // Does not exist
        // Save default config
        srv.clientConfig = DefaultClientConfig
        f2, err := os.OpenFile(fn, os.O_WRONLY | os.O_CREATE, 0600)
        Try(err)
        enc := json.NewEncoder(f2)
        enc.SetIndent("", "  ")
        Try(enc.Encode(srv.clientConfig))
        Try(f2.Close())

    default: // Exists
        dec := json.NewDecoder(f)
        cc  := ClientConfig{}
        Try(dec.Decode(&cc))
        srv.clientConfig = cc
    
    }

    // Version
    clCfg   := srv.clientConfig
    ccv     := make(map[string] string)
    for clId, clInfo := range clCfg.ClientMap {
        clRole    := clCfg.RoleMap.Get(clInfo.Role)
        ccv[clId]  = clRole.Version()
    }
    srv.clientConfigVersion = ccv
    
    return nil

}

func(srv *Server) cacheExecutable() (err error) {
    
    // Catch
    defer Catch(&err)

    // Read the executable file
    f, err := os.OpenFile(executablePath, os.O_RDONLY, 0644)
    Try(err)
    defer Try(f.Close())

    buf := bytes.NewBuffer(nil)
    io.Copy(buf, f)

    srv.cachedExecutable = buf.Bytes()
    return

}

func(srv *Server) checkAuthPrivateKey() error {
    apk := srv.config.AuthPrivateKeyPath
    return LoadAuthPrivateKey(apk)
}

func(srv *Server) Addr() string {
    return fmt.Sprintf("%s:%d", srv.config.Bind, srv.config.Port)
}

func(srv *Server) Start() (err error) {

    defer CatchFunc(&err, Logger.Panicln)

    // Config
    Try(srv.LoadConfig(flServerConfigPath))
    Logger.Infoln("Loaded server config")
    Try(srv.loadClientConfig())
    Logger.Infoln("Loaded client config")

    // Private Key
    Try(srv.checkAuthPrivateKey())
    Logger.Infoln("The fingerprint of the authentication public key is:")
    Logger.Infoln(sessionAuthPriv.PublicKey.Fingerprint())

    // Cache Executable
    Try(srv.cacheExecutable())
    Logger.Infoln("Cached executable for auto-update")

    // Read Monitor Data Cache
    Try(srv.readCachedClientMonitorDataMap())
    Logger.Infoln("Read the cached monitored items")

    // Ensure Directories
    Try(EnsureDirectory(srv.config.DataCacheDir))
    Logger.Infoln("Ensured necessary directories")

    // Network
    addr    := srv.Addr()
    ln, err := net.Listen("tcp", addr)
    Try(err)
    Logger.Infoln("Network is configured to listen at", addr)

    // Thread-related
    hs := together.NewHoldSwitch()
    const (
        threadMain = iota
        threadCleanUp
    )

    // Signal Catcher
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <- sig
        fmt.Println()
        Logger.Infoln("Waiting for tasks to finish...")
        hs.Add(threadCleanUp, 1)
        Logger.Infoln("Bye bye")
        os.Exit(0)
    }()

    // Flush Cache Thread
    go func() {
        for {func() {
            // Function wrapping in order to use defer
            defer CatchFunc(nil, Logger.Warnln)

            // Sleep at beginning
            time.Sleep(time.Minute * time.Duration(srv.config.DataCacheInterval))

            // Add task
            hs.Add(threadMain, 1)
            Try(srv.CacheClientMonitorDataMap())
            Logger.Infoln("Cached client monitor data")

            // Task done
            hs.Done(threadMain)
        }()}
    }()
    Logger.Infoln("Started monitor data caching thread")

    // Client Config Version Update
    go func() {
        ccp     := srv.config.ClientConfigPath
        st, _   := os.Stat(ccp)
        lastMod := st.ModTime()
        for {func() {
            // Catch
            defer CatchFunc(nil, Logger.Warnln)

            // Sleep at beginning
            time.Sleep(clientConfigWatchInterval)

            // Add task
            hs.Add(threadMain, 1)

            // Mod Time Check
            st, err := os.Stat(ccp)
            Try(err)

            // Changed
            if lastMod != st.ModTime() {
                Try(srv.loadClientConfig())
                Logger.Infoln("Reloaded client config")
                lastMod = st.ModTime()
            }
            hs.Done(threadMain)
        }()}
    }()
    Logger.Infoln("Started client config reloading thread")

    // Chart-ready Data Preparing Thread
    go func() {
        for {func() {
            defer CatchFunc(nil, Logger.Warnln)

            // Add task
            hs.Add(threadMain, 1)
            tBoxMap := make(map[string/* clId */] MonitorDataTableBox)
            gthSec  := int64(srv.config.GapThresholdTime * 60) // To seconds
            for clId, mdMap := range srv.clientMonitorDataMap {
                // MonitorData
                tsMap  := make(map[int64/* timestamp */] struct{})
                mdtMap := make(map[MonitorKey] []byte)
                // Table-writing loop
                for mKey, mData := range mdMap {
                    // Decimate monitor data
                    decimated := LttbMonitorData(
                        mData, srv.config.DecimationThreshold,
                    )
                    // Write CSV(table)
                    table  := bytes.NewBuffer(nil)
                    prevTs := decimated[0].Timestamp
                    fmt.Fprint(table, "timestamp,value\n")
                    for _, each := range decimated {
                        ts := each.Timestamp
                        // Check if there is gap
                        if ts - prevTs > gthSec {
                            // Put NaN(Gap)
                            midTs        := (ts + prevTs) / 2
                            tsMap[midTs]  = struct{}{}
                            fmt.Fprintf(table, "%d,NaN\n", midTs)
                        }
                        prevTs    = ts
                        tsMap[ts] = struct{}{}
                        fmt.Fprintf(table, "%d,%f\n", ts, each.Value)
                    }
                    // Assign csv
                    mdtMap[mKey] = table.Bytes()
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
                tBoxMap[clId] = MonitorDataTableBox{
                    Boundaries: bds.Bytes(),
                    DataMap: mdtMap,
                }

            }
            // Assign
            srv.clientMonitorDataTableBox = tBoxMap
            Logger.Infoln("Chart-ready data prepared")

            // Task done
            hs.Done(threadMain)

            // Sleep at end
            time.Sleep(time.Minute * time.Duration(srv.config.DecimationInterval))
        }()}
    }()
    Logger.Infoln("Started data decimation thread")

    // Http
    go srv.startHttpServer()
    Logger.Infoln("Started HTTP server")

    // Main
    Logger.Infoln("Successfully started the server")
    for {

        // Sleep
        time.Sleep(time.Duration(1000.0 / float64(srv.config.Tickrate)) * time.Millisecond)

        // Connection
        conn, err := ln.Accept()
        if err != nil {
            Logger.Warnln(err)
            continue
        }

        go func() {
            var host string
            defer CatchFunc(nil, Logger.Warnln, host)

            host, _  = HostnameOf(conn)
            rd      := bufio.NewReader(conn)
            // Start line
            startLine, err := rd.ReadString('\n')
            if err == io.EOF { return }
            Assert(err == nil, "Unexpected start line:" + startLine)
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
                // Source reader
                src := bufio.NewReader(io.MultiReader(bytes.NewReader(already), conn))
                go connCopy(conn, proxy) // Proxy -> Conn
                for {
                    // Look for requests
                    req, err := http.ReadRequest(src)
                    if err == io.EOF {
                        break
                    } else if err != nil {
                        panic(err)
                    }

                    // New request
                    Logger.Infoln(host, req.Method, req.URL.Path, req.Proto)
                    req.WriteProxy(proxy) // Conn -> Proxy
                }
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

func(srv *Server) readCachedClientMonitorDataMap() (err error) {

    defer Catch(&err)

    // Search for cache files
    matches, err := filepath.Glob(
        srv.config.DataCacheDir + "/*" + dataCacheExt,
    )
    Try(err)

    for _, match := range matches {func() {
        defer CatchFunc(nil, Logger.Warnln, "Failed to read cache:" + match)

        f, err2 := os.OpenFile(match, os.O_RDONLY, 0644)
        Try(err2)

        var (
            clId string
            mKey MonitorKey
            cmp  []byte
        )

        dec := gob.NewDecoder(f)
        Try(dec.Decode(&clId))
        Try(dec.Decode(&mKey))
        Try(dec.Decode(&cmp))
        Try(f.Close())

        // Decompress
        mData, err2 := DecompressMonitorData(cmp)
        Try(err2)

        // Assign
        _, ok := srv.clientMonitorDataMap[clId]
        if !ok {
            srv.clientMonitorDataMap[clId] = make(MonitorDataMap)
        }
        srv.clientMonitorDataMap[clId][mKey] = mData
    }()}

    return

}

func(srv *Server) CacheClientMonitorDataMap() (err error) {

    defer CatchFunc(&err, Logger.Warnln)

    for clId, mDataMap := range srv.clientMonitorDataMap {
        
        for mKey, mData := range mDataMap {func() {
            defer CatchFunc(nil, Logger.Warnln, "Failed to cache:", clId, mKey)

            // Compress
            cmp, err2 := CompressMonitorData(mData)
            Try(err2)

            // Encode
            buf := bytes.NewBuffer(nil)
            enc := gob.NewEncoder(buf)
            Try(enc.Encode(clId))
            Try(enc.Encode(mKey))
            Try(enc.Encode(cmp))

            // Write file
            h  := fmt.Sprintf("%x", Sha256Sum([]byte(
                clId + string(mKey), // TODO: Add specs for cache name in the documentation
            )))
            fn := srv.config.DataCacheDir + "/" + h + dataCacheExt
            Try(rewriteFile(fn, buf))

        }()}

    }

    return

}

func(srv *Server) RecordValueMap(clId string, timestamp int64, valMap map[MonitorKey] interface{}) {

    // Ensure
    _, ok := srv.clientMonitorDataMap[clId]
    if !ok {
        srv.clientMonitorDataMap[clId] = make(MonitorDataMap)
    }

    //
    initialized := make(MonitorData, 0)
    fatalValues := make(map[MonitorKey] float64)
    appendValue := func(mKey MonitorKey, val float64) {

        short, ok := srv.clientMonitorDataMap[clId][mKey]
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
        srv.clientMonitorDataMap[clId][mKey] = append(
            short,
            MonitorDatum{
                Timestamp: timestamp,
                Value:     val,
            },
        )

        // Check Status
        st := srv.getMonitorConfig(clId, mKey).StatusOf(val)
        if st == MonitorStatusFatal {
            fatalValues[mKey] = val
        }

    }

    //
    for mKey, val := range valMap {
        switch cast := val.(type) {
        case float64:
            appendValue(mKey, cast)
        case map[string] float64:
            for idx, subVal := range cast {
                longKey := mKey.WithIndex(idx)
                appendValue(longKey, subVal)
            }
        }
    }

    // Send webhook
    go func() {
        err := srv.sendAlarmWebhook(clId, timestamp, fatalValues)
        if err != nil {
            Logger.Warnln("Failed to send webhook:", err)
        }
    }()

}

// WEBHOOK ---

type alarmWebhook struct {
    ClientId    string `json:"clientId"`
    Timestamp   int64 `json:"timestamp"`
    FatalValues map[MonitorKey] float64 `json:"fatalValues"`
}
func(srv *Server) sendAlarmWebhook(clId string, timestamp int64, fatalValues map[MonitorKey] float64) error {

    // Empty Values
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
        ClientId:    clId,
        Timestamp:   timestamp,
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

func(srv *Server) getMonitorConfig(clId string, mKey MonitorKey) (MonitorConfig) {

    aBase, aParam, aIdx := mKey.Base(), mKey.Parameter(), mKey.Index()
    // Base + parameter match
    var bpMatch MonitorConfig

    clCfg   := srv.clientConfig
    clInfo  := clCfg.ClientMap[clId]
    clRole  := clCfg.RoleMap.Get(clInfo.Role)
    mCfgMap := clRole.MonitorConfigMap
    for b, mCfg := range mCfgMap {
        bBase, bParam, bIdx := b.Base(), b.Parameter(), b.Index()
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

func(srv *Server) HandleSession(s *Session) (err error) {

    defer CatchFunc(&err, Logger.Warnln)

    // Get Address
    host, err := s.RemoteHost()
    Try(err)

    // Check Whitelisted
    clCfg       := srv.clientConfig
    whitelisted := false
    for _, clInfo := range clCfg.ClientMap {
        // Find the first with the address
        if clInfo.Host == host {
            whitelisted = true
            break
        }
    }
    if !whitelisted {
        srvRsp := NewResponse("not-whitelisted")
        s.WriteResponse(srvRsp)
        return fmt.Errorf("%s [non-whitelisted] tried to establish a connection", host)
    }

    Logger.Infoln(host, "connected")
    clRsp, err := s.NextResponse()
    Try(err)

    // Role
    var clId   string
    var clInfo ClientInfo
    alias := clRsp.String("alias")
    for id, info := range clCfg.ClientMap {
        if info.Host == host && info.Alias == alias {
            clId   = id
            clInfo = info
            break
        }
    }
    Assert(clId != "", "Client not found in the config")
    clRole := clCfg.RoleMap.Get(clInfo.Role)

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
        Logger.Warnln(clId, "version mismatch, session terminated")
        return nil
    }

    // Main handling
    switch clRsp.Name() {
    case "hello":

        roleBytes, err := json.Marshal(clRole)
        Try(err)
        srvRsp := NewResponse("hello")
        srvRsp.Set("role", roleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[clId])
        Logger.Infoln(clId, "HELLO CLIENT")
        return s.WriteResponse(srvRsp)

    case "monitor-record":
        //
        valMap, ok := clRsp.Args()["valueMap"].(map[string] interface{})
        Assert(ok, "Malformed value map")
        castMap := make(map[MonitorKey] interface{})
        for key, val := range valMap {
            castMap[MonitorKey(key)] = val
        }
        timestamp := clRsp.Int64("timestamp")
        srv.RecordValueMap(clId, timestamp, castMap)
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
    case srv.clientConfigVersion[clId]:
        // Version match
    default:
        // Version mismatch
        roleBytes, err := json.Marshal(clRole)
        Try(err)
        srvRsp := NewResponse("reconfigure")
        srvRsp.Set("role", roleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[clId])
        return s.WriteResponse(srvRsp)
    }

    // # OK
    srvRsp := NewResponse("ok")
    return s.WriteResponse(srvRsp)

}