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
    dataStoreExt              = ".store"
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
    DataStoreInterval   int    `json:"monitor.dataStoreInterval"` // (minutes)
    DataStoreDir        string `json:"monitor.dataStoreDir"`
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
    DataStoreInterval:   1,
    DataStoreDir:        "./serverStore.d",
    MaxDataLength:       43200, // 30 days long for 1-minute interval
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
    InfoMap ClientInfoMap `json:"infoMap"`
    RuleMap ClientRuleMap `json:"ruleMap"`
}

var DefaultClientConfig = ClientConfig{
    
    InfoMap: ClientInfoMap{

        "example-01": ClientInfo{
            Host:  "127.0.0.1",
            Alias: "Example",
            Tags:  "basic example",
        },

    },
    
    RuleMap: ClientRuleMap{

        "basic": {

            MonitorConfigMap: MonitorConfigMap{
                "cpu-count": MonitorConfig{
                    Format: "{} CPUs",
                    Constant: true,
                },
                "memory-size-gb": MonitorConfig{
                    Format: "{.2f} GB",
                    Constant: true,
                },
                "swap-size-gb": MonitorConfig{
                    Format: "{.2f} GB",
                    Constant: true,
                },
                "disk-size-gb": MonitorConfig{
                    Format: "{.2f} GB",
                    Constant: true,
                },
            },
            MonitorInterval: 60,

        },

        "example": {

            MonitorConfigMap: MonitorConfigMap{
                "cpu-usage": MonitorConfig{
                    FatalRange:   "90:",
                    WarningRange: "82:",
                    Format:       "{.1f}%",
                },
                "memory-usage": MonitorConfig{
                    FatalRange:   "88:",
                    WarningRange: "75:",
                    Format:       "{.1f}%",
                },
                "swap-usage": MonitorConfig{
                    FatalRange:   "88:",
                    WarningRange: "75:",
                    Format:       "{.1f}%",
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
    clientConfigVersion       map[string/* clId */] string
    clientMonitorDataTableBox map[string/* clId */] MonitorDataTableBox
    clientMonitorDataMap      map[string/* clId */] MonitorDataMap
}

func NewServer() *Server {
    srv := &Server{
        clientMonitorDataMap: make(map[string/* clId */] MonitorDataMap),
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
    defer Catch(&err)

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
    clCfg := srv.clientConfig
    ccv   := make(map[string] string)
    for clId, clInfo := range clCfg.InfoMap {
        clRule    := clCfg.RuleMap.Get(clInfo.Tags)
        ccv[clId]  = clRule.Version()
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

    defer Catch(&err)

    // Config
    Try(srv.LoadConfig(flServerConfigPath))
    EventLogger.Infoln("Loaded server config")
    Try(srv.loadClientConfig())
    EventLogger.Infoln("Loaded client config")

    // Private key
    Try(srv.checkAuthPrivateKey())
    EventLogger.Infoln("The fingerprint of the authentication public key is:")
    EventLogger.Infoln(sessionAuthPriv.PublicKey.Fingerprint())

    // Cache executable
    Try(srv.cacheExecutable())
    EventLogger.Infoln("Cached executable for auto-update")

    // Read stored monitor data
    Try(srv.readStoredClientMonitorDataMap())
    EventLogger.Infoln("Read the stored monitored data")

    // Ensure directories
    Try(EnsureDirectory(srv.config.DataStoreDir))
    EventLogger.Infoln("Ensured necessary directories")

    // Network
    addr    := srv.Addr()
    ln, err := net.Listen("tcp", addr)
    Try(err)
    EventLogger.Infoln("Network is configured to listen at", addr)

    // Schedule cleanups
    railSwitch.OnEnd(threadMain, func() {

        defer func() {
            if r := recover(); r != nil {
                EventLogger.Warnln(r)
            }
        }()
        Try(srv.StoreClientMonitorDataMap())
        EventLogger.Infoln("Stored client monitor data")

    })

    // Data storing thread
    go func() {

        itv := time.Minute * time.Duration(srv.config.DataStoreInterval)

        for Sleep(itv) && railSwitch.Queue(threadMain, 1) {

            timer := EventLogger.Timer("time:StoreClientMonitorDataMap")
            err := srv.StoreClientMonitorDataMap()
            timer.Stop()

            // Task done
            railSwitch.Proceed(threadMain)

            if err != nil {
                EventLogger.Warnln(err)
                continue
            }
            EventLogger.Infoln("Stored client monitor data")

        }

    }()
    EventLogger.Infoln("Started monitor data caching thread")

    // Client Config Version Update
    go func() {

        ccp     := srv.config.ClientConfigPath
        st, _   := os.Stat(ccp)
        lastMod := st.ModTime()

        for Sleep(clientConfigWatchInterval) && railSwitch.Queue(threadMain, 1) {

            // Mod Time Check
            st, err := os.Stat(ccp)
            if err != nil {
                EventLogger.Warnln(err)
            } else if lastMod != st.ModTime() {
                // Changed
                err = srv.loadClientConfig()
                if err != nil {
                    EventLogger.Warnln(err)
                } else {
                    EventLogger.Infoln("Reloaded client config")
                    lastMod = st.ModTime()
                }
            }

            railSwitch.Proceed(threadMain)

        }

    }()
    EventLogger.Infoln("Started client config reloading thread")

    // Chart-ready csv preparing thread
    go func() {
        
        for railSwitch.Queue(threadMain, 1) {
            
            func() {

                defer func() {
                    if r := recover(); r != nil {
                        EventLogger.Warnln(r)
                    }
                }()

                //
                timer := EventLogger.Timer("time:DataPreparation")

                clMdtBox := make(map[string/* clId */] MonitorDataTableBox)
                gthSec   := int64(srv.config.GapThresholdTime * 60) // To seconds

                for clId, mdMap := range srv.clientMonitorDataMap {

                    // Maps
                    tsMap  := make(map[int64/* timestamp */] struct{})
                    mdtMap := make(map[string] []byte)

                    // Table-writing loop
                    for mKey, mData := range mdMap {

                        // Decimate monitor data
                        decimated := LttbMonitorData(
                            mData, srv.config.DecimationThreshold,
                        )

                        // Write CSV(table)
                        csv    := bytes.NewBuffer(nil)
                        prevTs := decimated[0].Timestamp
                        fmt.Fprint(csv, "timestamp,value,per\n")

                        // Write rows
                        for _, each := range decimated {

                            ts := each.Timestamp

                            // Check if there is gap
                            if ts - prevTs > gthSec {
                                // Put NaN which indicates a gap
                                midTs        := (ts + prevTs) / 2
                                tsMap[midTs]  = struct{}{}
                                fmt.Fprintf(csv, "%d,NaN,NaN\n", midTs)
                            }

                            prevTs    = ts
                            tsMap[ts] = struct{}{}
                            fmt.Fprintf(csv, "%d,%f,%d\n", ts, each.Value, each.Per)

                        }

                        // Assign csv
                        mdtMap[mKey] = csv.Bytes()

                    }

                    // Timestamps slice
                    i, tss := 0, make([]int64, len(tsMap))
                    for t := range tsMap {
                        tss[i] = t
                        i++
                    }
                    sort.Sort(Int64Slice(tss)) // Sort

                    // TODO make this documented
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
                    clMdtBox[clId] = MonitorDataTableBox{
                        Boundaries: bds.Bytes(),
                        DataMap: mdtMap,
                    }

                }

                // Assign
                srv.clientMonitorDataTableBox = clMdtBox
                EventLogger.Infoln("Chart-ready data prepared")

                timer.Stop()

            }()

            // Task done
            railSwitch.Proceed(threadMain)

            // Sleep at end
            time.Sleep(time.Minute * time.Duration(srv.config.DecimationInterval))
    
        }
    }()
    EventLogger.Infoln("Started data decimation thread")

    // Http
    go srv.startHttpServer()
    EventLogger.Infoln("Started HTTP server")

    // Main
    EventLogger.Infoln("Successfully started the server")
    for {

        // Sleep
        time.Sleep(time.Duration(1000.0 / float64(srv.config.Tickrate)) * time.Millisecond)

        // Connection
        conn, err := ln.Accept()
        if err != nil {
            EventLogger.Warnln(err)
            continue
        }

        go func() {

            host, _ := HostnameOf(conn)
            defer func() {
                if r := recover(); r != nil {
                    EventLogger.Warnln(host, r)
                }
            }()

            rd := bufio.NewReader(conn)

            // Start line
            startLine, err := rd.ReadString('\n')
            if err == io.EOF { return }
            Assert(err == nil, "Unexpected start line: " + startLine)

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

                // Keep connection open and stream requests continuously
                for {

                    // Look for requests
                    req, err := http.ReadRequest(src)
                    if err == io.EOF {
                        return
                    } else if err != nil {
                        panic(err)
                    }

                    // New request
                    AccessLogger.Infoln(host, req.Method, req.URL.Path, req.Proto)
                    req.WriteProxy(proxy) // Conn -> Proxy

                }

            case strings.Contains(startLine, "TELESCRIBE"):

                // TELESCRIBE
                s := NewSession(conn)

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

func(srv *Server) readStoredClientMonitorDataMap() (err error) {

    defer Catch(&err)

    // Search for stored files
    matches, err := filepath.Glob(
        srv.config.DataStoreDir + "/*" + dataStoreExt,
    )
    Try(err)

    // Per file
    for _, match := range matches {func() {

        defer func() {
            if r := recover(); r != nil {
                EventLogger.Warnln("Failed to read stored data:", match, r)
            }
        }()

        f, err2 := os.OpenFile(match, os.O_RDONLY, 0644)
        Try(err2)

        var (
            clId string
            mKey string
            cmp  []byte
        )

        // Decode gob
        dec := gob.NewDecoder(f)
        Try(dec.Decode(&clId))
        Try(dec.Decode(&mKey))
        Try(dec.Decode(&cmp))
        Try(f.Close())

        // Check config
        mCfg, ok := srv.getMonitorConfig(clId, mKey)
        switch {
        case mCfg.Constant, // Ignore constant items
            !ok: // Ignore items with no config
            return
        }

        // Decompress
        mData, err2 := DecompressMonitorData(cmp)
        Try(err2)

        // Assign
        _, ok = srv.clientMonitorDataMap[clId]
        if !ok {
            srv.clientMonitorDataMap[clId] = make(MonitorDataMap)
        }
        srv.clientMonitorDataMap[clId][mKey] = mData

    }()}

    return

}

func(srv *Server) StoreClientMonitorDataMap() (err error) {

    defer Catch(&err)

    for clId, mdMap := range srv.clientMonitorDataMap {
        
        for mKey, mData := range mdMap {func() {

            defer func() {
                if r := recover(); r != nil {
                    EventLogger.Warnln("Failed to store:", clId, mKey, r)
                }
            }()

            // Check config
            mCfg, ok := srv.getMonitorConfig(clId, mKey)
            switch {
            case mCfg.Constant, // Ignore constant items
                !ok: // Ignore items with no config
                return
            }

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
                clId + string(mKey),
            )))
            fn := srv.config.DataStoreDir + "/" + h + dataStoreExt
            Try(rewriteFile(fn, buf))

        }()}

    }

    return

}

func(srv *Server) RecordValueMap(clId string, timestamp int64, valMap map[string] interface{}, per int) {

    // Ensure
    _, ok := srv.clientMonitorDataMap[clId]
    if !ok {
        srv.clientMonitorDataMap[clId] = make(MonitorDataMap)
    }

    //
    fatalValues := make(map[string] float64)
    appendValue := func(mKey string, val float64) {

        short, ok := srv.clientMonitorDataMap[clId][mKey]
        if !ok {
            short = make(MonitorData, 0)
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
                Per:       per,
            },
        )

        // Check Status
        cfg, ok := srv.getMonitorConfig(clId, mKey)
        if !ok {
            EventLogger.Warnln("Monitor config for", mKey, "was not found")
            return
        }

        // Fatal Check
        st := cfg.StatusOf(val)
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
                b, p, _ := ParseMonitorKey(mKey)
                longKey := FormatMonitorKey(b, p, idx)
                appendValue(longKey, subVal)
            }
        }
    }

    // Send webhook
    go func() {
        err := srv.sendAlarmWebhook(clId, timestamp, fatalValues)
        if err != nil {
            EventLogger.Warnln("Failed to send webhook:", err)
        }
    }()

}

// WEBHOOK ---

type alarmWebhook struct {
    ClientId    string `json:"clientId"`
    Timestamp   int64 `json:"timestamp"`
    FatalValues map[string] float64 `json:"fatalValues"`
}
func(srv *Server) sendAlarmWebhook(clId string, timestamp int64, fatalValues map[string] float64) error {

    // Empty values
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

func(srv *Server) getMonitorConfig(clId string, mKey string) (MonitorConfig, bool) {

    aBase, aParam, aIdx := ParseMonitorKey(mKey)
    // Base + parameter match
    var bpMatch MonitorConfig
    ok := false

    // Client-related
    clCfg   := srv.clientConfig
    clInfo  := clCfg.InfoMap[clId]
    clRule  := clCfg.RuleMap.Get(clInfo.Tags)
    mCfgMap := clRule.MonitorConfigMap
    for b, mCfg := range mCfgMap {
        bBase, bParam, bIdx := ParseMonitorKey(b)
        if aBase == bBase && aParam == bParam {
            if aIdx == bIdx {
                // Exact match
                return mCfg, true
            }
            bpMatch = mCfg
            ok = true
        }
    }

    return bpMatch, ok

}

func(srv *Server) HandleSession(s *Session) (err error) {

    logParams := make([]interface{}, 0)
    defer Catch(&err)

    // Get Address
    host, err := s.RemoteHost()
    Try(err)
    logParams = append(logParams, host)

    // Check Whitelisted
    clCfg       := srv.clientConfig
    whitelisted := false
    for _, clInfo := range clCfg.InfoMap {
        // Find the first with the address
        if clInfo.HasAddr(host) {
            whitelisted = true
            break
        }
    }
    if !whitelisted {
        srvRsp := NewResponse("not-whitelisted")
        s.WriteResponse(srvRsp)
        return fmt.Errorf("%s [non-whitelisted] tried to establish a connection", host)
    }

    clRsp, err := s.NextResponse()
    Try(err)

    // Rule
    var clId   string
    var clInfo ClientInfo
    alias := clRsp.String("alias")
    for id, info := range clCfg.InfoMap {
        if info.HasAddr(host) && info.Alias == alias {
            clId   = id
            clInfo = info
            break
        }
    }
    AccessLogger.Infoln(clInfo.Alias, "from", clInfo.Host, "connected")
    Assert(clId != "", "Client must be configured in the config")
    logParams  = append(logParams, clId)
    clRule    := clCfg.RuleMap.Get(clInfo.Tags)

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
        panic("Version mismatch, session terminated")
    }

    // Main handling
    switch clRsp.Name() {
    case "hello":

        ruleBytes, err := json.Marshal(clRule)
        Try(err)

        srvRsp := NewResponse("hello")
        srvRsp.Set("rule", ruleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[clId])
        EventLogger.Infoln(clId, "HELLO CLIENT")
        Try(s.WriteResponse(srvRsp))
        return nil

    case "monitor-record":
        
        timestamp  := clRsp.Int64("timestamp")
        valMap, ok := clRsp.Args()["valueMap"].(map[string] interface{})
        Assert(ok, "Malformed value map")
        per        := clRsp.Int("per")
        srv.RecordValueMap(clId, timestamp, valMap, per)

    default:
        panic("Unknown response")
    }

    // Post Handling

    // Config Version Check
    clCfgVer := clRsp.String("configVersion")
    switch clCfgVer {
    case "":
        // Version empty
        panic("Client response does not include config version")
    case srv.clientConfigVersion[clId]:
        // Version match
    default:
        // Version mismatch
        ruleBytes, err := json.Marshal(clRule)
        Try(err)

        srvRsp := NewResponse("reconfigure")
        srvRsp.Set("rule", ruleBytes)
        srvRsp.Set("configVersion", srv.clientConfigVersion[clId])
        Try(s.WriteResponse(srvRsp))
        return nil
    }

    // OK
    srvRsp := NewResponse("ok")
    Try(s.WriteResponse(srvRsp))
    return nil

}