package main

import (
    "bufio"
    "bytes"
    "crypto/sha256"
    "encoding/binary"
    "encoding/gob"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"

    . "github.com/hjjg200/go-act"
    "./config"
)

const (
    dataStoreExt              = ".store"
    clientMetaExt             = ".meta"
    clientConfigWatchInterval = time.Second * 5
)

// SERVER CONFIG ---

type ServerConfig struct { // srvCfg
    // General
    AuthPrivateKeyPath  string `json:"authPrivateKeyPath"`
    ClientConfigPath    string `json:"clientConfigPath"`
    ClientMetaDir       string `json:"clientMetaDir"`
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
    DataIndexesFile     string `json:"monitor.dataIndexesFile"`
    DataChunkLength     int    `json:"monitor.dataChunkLength"`
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
    ClientMetaDir:      "./clientMeta.d",
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
    DataIndexesFile:     "./dataIndexes.json",
    DataChunkLength:     1000, // 20 kB per chunk
    // Web
    Web:                 DefaultWebConfig,
    // Network
    Bind:                "0.0.0.0",
    Port:                1226,
    Tickrate:            60,
    // Alarm
    WebhookUrl:          "",
}


// CLIENT META ---

const (
    clientMetaKeyLastHello = "lastHello"
    clientMetaKeyLastConnection = "lastConnection"
    clientMetaKeyGaps = "gaps"
)

func(srv *Server) openClientMetaFile(clId, key string) (*os.File, error) {

    hash := sha256.New()
    hash.Write([]byte(clId))
    hashId := fmt.Sprintf("%x", hash.Sum(nil))

    // Ensure
    dir := srv.config.ClientMetaDir + "/" + hashId
    err := EnsureDirectory(dir)
    if err != nil {
        return nil, err
    }

    //
    fn := dir + "/" + key + clientMetaExt
    return os.OpenFile(fn, os.O_RDWR | os.O_CREATE, 0600)

}
func(srv *Server) readClientMetaFile(clId, key string) (p []byte, err error) {

    defer Catch(&err)
    f, err := srv.openClientMetaFile(clId, key)
    Try(err)

    return ioutil.ReadAll(f)

}
func(srv *Server) rewriteClientMetaFile(clId, key string, p []byte) (err error) {

    defer Catch(&err)
    f, err := srv.openClientMetaFile(clId, key)
    Try(err)
    Try(f.Truncate(0))
    _, err = f.Seek(0, 0)
    Try(err)
    _, err = f.Write(p)
    return err

}
func(srv *Server) appendClientMetaFile(clId, key string, p []byte) (err error) {

    defer Catch(&err)
    f, err := srv.openClientMetaFile(clId, key)
    Try(err)
    _, err = f.Write(p)
    return err

}
func(srv *Server) getClientMetaInt64(clId, key string) (int64, bool) {
    le := binary.LittleEndian
    p, err := srv.readClientMetaFile(clId, key)
    switch {
    case err != nil,
        len(p) < 8:
        return 0, false
    }
    return int64(le.Uint64(p)), true
}
func(srv *Server) getClientMetaInt64Slice(clId, key string) []int64 {
    le := binary.LittleEndian
    p, err := srv.readClientMetaFile(clId, key)
    arr := []int64{}
    switch {
    case err != nil,
        len(p) % 8 != 0:
        return arr
    }
    for i := 0; i < len(p); i += 8 {
        arr = append(arr, int64(le.Uint64(p[i:i+8])))
    }
    return arr
}
func(srv *Server) updateClientMetaInt64(clId, key string, v int64) error {
    le := binary.LittleEndian
    p := make([]byte, 8)
    le.PutUint64(p, uint64(v))
    return srv.rewriteClientMetaFile(clId, key, p)
}
func(srv *Server) appendClientMetaInt64Slice(clId, key string, arr []int64) error {
    le := binary.LittleEndian
    p := make([]byte, 8 * len(arr))
    for i, v := range arr {
        le.PutUint64(p[i*8:i*8 + 8], uint64(v))
    }
    return srv.appendClientMetaFile(clId, key, p)
}

func(srv *Server) GetClientMetaLastHello(clId string) (int64, bool) {
    return srv.getClientMetaInt64(clId, clientMetaKeyLastHello)
}
func(srv *Server) GetClientMetaLastConnection(clId string) (int64, bool) {
    return srv.getClientMetaInt64(clId, clientMetaKeyLastConnection)
}
func(srv *Server) GetClientMetaGaps(clId string) []int64 {
    return srv.getClientMetaInt64Slice(clId, clientMetaKeyGaps)
}
func(srv *Server) UpdateClientMetaLastHello(clId string, ts int64) error {
    return srv.updateClientMetaInt64(clId, clientMetaKeyLastHello, ts)
}
func(srv *Server) UpdateClientMetaLastConnection(clId string, ts int64) error {
    return srv.updateClientMetaInt64(clId, clientMetaKeyLastConnection, ts)
}
func(srv *Server) AppendClientMetaGaps(clId string, from, to int64) error {
    return srv.appendClientMetaInt64Slice(clId, clientMetaKeyGaps, []int64{from, to})
}


// CLIENT CONFIG ---

type ClientConfig struct { // clCfg
    InfoMap ClientInfoMap `json:"infoMap"`
    RuleMap ClientRuleMap `json:"ruleMap"`
}

var DefaultClientInfo = ClientInfo{
    Host: "127.0.0.1",
    Alias: "Undefined",
    Tags: "basic",
}

var DefaultClientRule = ClientRule{
    MonitorConfigMap: MonitorConfigMap{},
    MonitorInterval: 60,
}

var DefaultMonitorConfig = MonitorConfig{
    Absolute: false,
    Alias: "",
    Constant: false,
    Format: "",
    FatalRange: "",
    WarningRange: "",
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

        "basic": ClientRule{

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

        "example": ClientRule{

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
    config                      ServerConfig
    cachedExecutable            []byte
    httpListener                net.Listener
    httpRouter                  *httpRouter
    authFingerprint             string
    clientConfig                ClientConfig
    clientConfigVersion         map[string/* clId */] string
    clientMonitorDataTableBox   map[string/* clId */] MonitorDataTableBox
    clientMonitorDataMap        map[string/* clId */] MonitorDataMap
    clientMonitorDataIndexesMap map[string/* clId */] MonitorDataIndexesMap
    configParser                *config.Parser
    clientConfigParser          *config.Parser
}

func NewServer() *Server {
    srv := &Server{
        clientMonitorDataMap: make(map[string/* clId */] MonitorDataMap),
        clientMonitorDataIndexesMap: make(map[string/* clId */] MonitorDataIndexesMap),
    }
    return srv
}

func (srv *Server) setConfigValidators() (err error) {

    defer Catch(&err)

    cp := srv.configParser

    vAboveZero := func(v int) bool {return v > 0}

    Try(cp.Validator(&DefaultServerConfig.DataStoreInterval, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.MaxDataLength, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.GapThresholdTime, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.DecimationThreshold, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.DecimationInterval, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.Port, func(v int) bool {
        return v >= 0 && v <= 65535
    }))
    Try(cp.Validator(&DefaultServerConfig.Tickrate, vAboveZero))
    Try(cp.Validator(&DefaultServerConfig.Web.Durations, func(v []int) bool {
        for _, d := range v {
            if d <= 0 {return false}
        }
        return true
    }))

    return nil

}

func (srv *Server) LoadConfig(p string) (err error) {

    // Catch
    defer Catch(&err)

    // Open the config file
    f, err := os.OpenFile(p, os.O_RDONLY, 0644)
    switch {
    case os.IsNotExist(err):
        // Not exists
        Try(srv.configParser.Parse([]byte("{}"), &srv.config))
    case err != nil:
        // Unexpected error
        panic(err)
    default:
        // File exists
        p, err := ioutil.ReadAll(f)
        Try(err)
        Try(srv.configParser.Parse(p, &srv.config))
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

func(srv *Server) setClientConfigValidators() (err error) {

    defer Catch(&err)

    cp := srv.clientConfigParser

    Try(cp.Validator(&DefaultClientConfig.InfoMap, func(m ClientInfoMap) bool {
        for k := range m {
            if k == "" {return false}
        }
        return true
    }))
    Try(cp.Validator(&DefaultClientRule.MonitorInterval, func(i int) bool {
        return i > 0
    }))

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
    case os.IsNotExist(err): // Does not exist

        // Save default config
        srv.clientConfig = DefaultClientConfig
        f2, err2 := os.OpenFile(fn, os.O_WRONLY | os.O_CREATE, 0600)
        Try(err2)
        enc := json.NewEncoder(f2)
        enc.SetIndent("", "  ")
        Try(enc.Encode(srv.clientConfig))
        Try(f2.Close())

    case err != nil: // Unexpected error
        panic(err)

    default: // Exists

        buf := bytes.NewBuffer(nil)
        io.Copy(buf, f)
        Try(f.Close())
        Try(srv.clientConfigParser.Parse(buf.Bytes(), &srv.clientConfig))
        tmp, _ := json.MarshalIndent(srv.clientConfig, "", "  ")
        EventLogger.Debugln("config", string(tmp))
        
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
    EventLogger.Debugln("may27:executablePath", executablePath)
    Try(err)

    buf    := bytes.NewBuffer(nil)
    _, err  = io.Copy(buf, f)
    Try(err)
    Try(f.Close())

    srv.cachedExecutable = buf.Bytes()
    EventLogger.Debugln("may27:executableSize", len(srv.cachedExecutable))
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

    // Server config
    configParser, err := config.NewParser(&DefaultServerConfig)
    Try(err)
    srv.configParser = configParser
    Try(srv.setConfigValidators())
    Try(srv.LoadConfig(flServerConfigPath))
    EventLogger.Infoln("Loaded server config")

    // Client config
    clientConfigParser, err := config.NewParser(&DefaultClientConfig)
    Try(err)
    Try(clientConfigParser.ChildDefaults(
        &DefaultClientInfo, &DefaultClientRule, &DefaultMonitorConfig,
    ))
    srv.clientConfigParser = clientConfigParser
    Try(srv.setClientConfigValidators())
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
    Try(srv.readClientMonitorIndexesMap())
    EventLogger.Infoln("Read the monitor data indexes")

    // Ensure directories
    Try(EnsureDirectory(srv.config.DataStoreDir))
    Try(EnsureDirectory(srv.config.ClientMetaDir))
    EventLogger.Infoln("Ensured necessary directories")

    // Network
    addr    := srv.Addr()
    ln, err := net.Listen("tcp", addr)
    Try(err)
    EventLogger.Infoln("Network is configured to listen at", ln.Addr())

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
    go func() {for railSwitch.Queue(threadMain, 1) {func() {

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

        // Task done
        railSwitch.Proceed(threadMain)

        // Sleep at end
        time.Sleep(time.Minute * time.Duration(srv.config.DecimationInterval))

    }()}}()
    EventLogger.Infoln("Started data decimation thread")

    // Debug
    if flDebug {
        go func() {
            for {
                time.Sleep(30 * time.Second)
                var clId, mKey string
                for i, mdMap := range srv.clientMonitorDataMap { // indexes dont count with this for
                    clId = i
                    for j := range mdMap {
                        mKey = j
                        break
                    } 
                    break
                }
                if clId == "" || mKey == "" {
                    continue
                }
                buf := bytes.NewBuffer(nil)
                srv.FprintMonitorDataCsvFilter(
                    buf, clId, mKey, FprintCsvFilter{
                        From: 0,
                        To: 1700785212,
                        Per: 1,
                    },
                )

                EventLogger.Debugln("fprintCsv", buf.String())
                length := srv.GetMonitorDataLength(clId, mKey)
                EventLogger.Debugln("fprintCsv", "\nLength:", length)
                EventLogger.Debugln("fprintCsv", "\n[len - 3:len]:", srv.GetMonitorDataSlice(clId, mKey, length - 3, length))
                EventLogger.Debugln("fprintCsv", "\nLast:", srv.GetMonitorDataSlice(clId, mKey, length - 1, length))
            }
        }()
    }

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

func(srv *Server) readClientMonitorIndexesMap() (err error) {

    defer Catch(&err)

    fn := srv.config.DataIndexesFile
    f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
    
    // Check existence
    if os.IsNotExist(err) {
        return TouchFile(fn, 0600)
    }
    Try(err)

    // JSON
    dec := json.NewDecoder(f)
    Try(dec.Decode(&srv.clientMonitorDataIndexesMap))

    return f.Close()

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
        _, ok := srv.getMonitorConfig(clId, mKey)
        switch {
        case !ok: // Ignore items with no config
            //mCfg.Constant, // Ignore constant items
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

func(srv *Server) GetClientIds() []string {
    infoMap := srv.clientConfig.InfoMap
    ret := []string{}
    for clId := range infoMap {
        ret = append(ret, clId)
    }
    return ret
}

func(srv *Server) GetClientMonitorKeys(clId string) ([]string, bool) {
    m := make(map[string] struct{})
    // Index
    indexes, ok1 := srv.clientMonitorDataIndexesMap[clId]
    for mKey := range indexes {m[mKey] = struct{}{}}
    // In-memory
    inMem, ok2 := srv.clientMonitorDataMap[clId]
    for mKey := range inMem   {m[mKey] = struct{}{}}

    if !(ok1 || ok2) {return nil, false}
    //
    ret := []string{}
    for mKey := range m {ret = append(ret, mKey)}
    return ret, true
}

func(srv *Server) GetMonitorDataLength(clId, mKey string) int {

    sum := 0
    
    // Indexes
    indexes := srv.clientMonitorDataIndexesMap[clId][mKey]
    for _, index := range indexes {
        sum += index.Length
    }

    // In-memory
    inMem := srv.clientMonitorDataMap[clId][mKey]
    sum += len(inMem)

    return sum

}

func(srv *Server) GetMonitorDataSlice(clId, mKey string, from, to int) MonitorData {

    base  := 0
    slice := MonitorData{}

    checker := func(length int) (int, int) {

        var p1, p2 int
        if from >= base && from < base + length { // within
            p1 = from - base
        } else if from < base { // past
            p1 = 0
        } else if from >= base + length { // to come
            p1 = length
        }

        if to > base + length { // to come
            p2 = length
        } else if to > base && to <= base + length { // within
            p2 = to - base
        } else if to < base + 1 {
            p2 = 0
        }

        base += length
        return p1, p2
    }
    
    // Indexes
    indexes := srv.clientMonitorDataIndexesMap[clId][mKey]
    for _, index := range indexes {
        p1, p2 := checker(index.Length)
        if p2 - p1 != 0 {
            part, err := srv.GetMonitorDataFromIndex(index.Uuid)
            if err != nil {
                EventLogger.Warnln("Failed to read index:", index.Uuid)
                continue
            }
            slice = append(slice, part[p1:p2]...)
        }
    }

    // In-memory
    inMem := srv.clientMonitorDataMap[clId][mKey]
    if len(inMem) > 0 {
        p1, p2 := checker(len(inMem))
        slice   = append(slice, inMem[p1:p2]...)
    }

    return slice

}

func(srv *Server) GetMonitorDataFromIndex(uuid string) (MonitorData, error) {

    fn := srv.config.DataStoreDir + "/" + uuid + dataStoreExt
    f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
    if err != nil {
        return nil, err
    }

    p, err := ioutil.ReadAll(f)
    if err != nil {
        return nil, err
    }

    part, err := DeserializeMonitorData(p)
    if err != nil {
        return nil, err
    }

    return part, nil

}

type FprintCsvFilter struct {
    From int64 `json:"from"`
    To   int64 `json:"to"`
    Per  int32 `json:"to"`
}
func(srv *Server) FprintMonitorDataBoundariesFilter(w io.Writer, clId string, filter FprintCsvFilter) {

    // CSV header
    w.Write([]byte("timestamp\n"))

    // Meta
    gaps := srv.GetClientMetaGaps(clId)

    // Find the start and end
    mdMap, ok := srv.clientMonitorDataMap[clId]
    if !ok {
        return
    }

    start, end := int64(1<<63 - 1), int64(0) // max int64, 0
    for _, md := range mdMap {
        if len(md) == 0 {
            continue
        }
        ts0 := md[0].Timestamp
        tsl := md[len(md) - 1].Timestamp
        if ts0 < start {
            start = ts0
        }
        if tsl > end {
            end = tsl
        }
    }

    // Unchanged
    if end == 0 {
        return
    }

    fmt.Fprintf(w, "%d\n", start)
    for i := 0; i < len(gaps); i += 2 {
        g1, g2 := gaps[i], gaps[i+1]
        fmt.Fprintf(w, "%d\n%d\n", g1, g2)
    }
    fmt.Fprintf(w, "%d\n", end)

}
func(srv *Server) FprintMonitorDataCsvFilter(w io.Writer, clId, mKey string, filter FprintCsvFilter) {

    // CSV header
    w.Write([]byte("timestamp,value,per\n"))

    // Get gaps from meta
    gaps := srv.GetClientMetaGaps(clId)

    // Funcs
    avgFunc := func(md MonitorData) float64 {
        acc := 0.0
        for _, datum := range md {
            acc += datum.Value
        }
        return acc / float64(len(md))
    }

    // Cache and flush
    cache := MonitorData{}
    flush := func() {
        if len(cache) == 0 {
            return
        }
        // NaN for mid points of gaps
        if len(gaps) > 0 {
            gMid := 0.0
            for ; len(gaps) > 0; {
                g0, g1 := gaps[0], gaps[1]
                if g1 < cache.From() {
                    gMid = float64(g0 + g1) / 2.0
                    gaps = gaps[2:]
                } else {
                    break
                }
            }

            if gMid != 0.0 {
                fmt.Fprintf(w, "%.1f,NaN,NaN\n", gMid)
            }
        }
        // Data
        fmt.Fprintf(
            w, "%.1f,%f,%d\n",
            cache.MidTime(), avgFunc(cache), cache.Duration(),
        )
        cache = MonitorData{}
    }
    put := func(datum MonitorDatum) {
        if filter.From > datum.Timestamp || filter.To < datum.Timestamp {
            return
        }
        cache = append(cache, datum)
        // When the put data's duration exceed or match the per 
        if cache.Duration() >= int64(filter.Per) {
            flush()
        }
    }

    // Indexes
    indexes := srv.clientMonitorDataIndexesMap[clId][mKey]
    for _, index := range indexes {func() {

        if index.To < filter.From || index.From > filter.To {
            return
        }

        // Debug
        if flDebug {
            fmt.Fprintf(w, "INDEX %s\n", index.Uuid)
        }

        warner := func(args ...interface{}) {
            args = append([]interface{}{"FprintMonitorDataCsvFilter:"}, args...)
            EventLogger.Warnln(args...)
        }

        part, err := srv.GetMonitorDataFromIndex(index.Uuid)
        if err != nil {
            warner("failed index", index.Uuid, err)
            return
        }

        // Write
        for _, datum := range part {
            put(datum)
        }

    }()}

    // In-memory data
    inMem := srv.clientMonitorDataMap[clId][mKey]
    if len(inMem) > 0 && inMem.To() > filter.From {
        if flDebug {
            fmt.Fprintf(w, "IN-MEMORY\n")
        }
        for _, datum := range inMem {
            put(datum)
        }
    }

    // Flush remaining
    flush()

}

func(srv *Server) StoreClientMonitorDataMap() (err error) {

    defer Catch(&err)

    // Current indexes
    clMdIdxMap := srv.clientMonitorDataIndexesMap

    for clId, mdMap := range srv.clientMonitorDataMap {

        // Check indexes
        if _, ok := clMdIdxMap[clId]; !ok {
            clMdIdxMap[clId] = make(MonitorDataIndexesMap)
        }
        
        for mKey, mData := range mdMap {func() {

            defer func() {
                if r := recover(); r != nil {
                    EventLogger.Warnln("Failed to store:", clId, mKey, r)
                }
            }()

            // Check indexes
            indexes := MonitorDataIndexes{}
            if _, ok := clMdIdxMap[clId][mKey]; ok {
                indexes = clMdIdxMap[clId][mKey]
            }

            // Check config
            _, ok := srv.getMonitorConfig(clId, mKey)
            switch {
            case !ok: // Ignore items with no config
                //mCfg.Constant: // Ignore constant items
                return
            }

            // Vars
            stored    := MonitorData{}
            lenChunk  := srv.config.DataChunkLength
            quotient  := len(mData) / lenChunk
            lenStored := quotient * lenChunk
            stored, srv.clientMonitorDataMap[clId][mKey] = mData[:lenStored], mData[lenStored:] // separate

            // Index each
            for i := 0; i < quotient; i++ {
                start   := i * lenChunk
                part    := stored[start:start + lenChunk]
                index   := CreateIndexForMonitorData(part)
                indexes  = indexes.Append(index)

                // Store
                fn := srv.config.DataStoreDir + "/" + index.Uuid + dataStoreExt
                f, err := os.OpenFile(fn, os.O_CREATE | os.O_WRONLY, 0600)
                Try(err)
                f.Write(SerializeMonitorData(part))
                f.Close()
            }

            // Assign
            clMdIdxMap[clId][mKey] = indexes

        }()}

    }

    // Store new indexes
    buf := bytes.NewBuffer(nil)
    enc := json.NewEncoder(buf)
    enc.Encode(clMdIdxMap)
    Try(rewriteFile(srv.config.DataIndexesFile, buf))

    // Replace
    //srv.clientMonitorDataIndexesMap = clMdIdxMap

    return

}

func(srv *Server) RecordValueMap(clId string, timestamp int64, valMap map[string] interface{}, per int32) {

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

        // Meta
        timestamp := time.Now().Unix()
        lastConnection, ok := srv.GetClientMetaLastConnection(clId)
        if ok {
            Try(srv.AppendClientMetaGaps(clId, lastConnection, timestamp))
        }
        Try(srv.UpdateClientMetaLastHello(clId, timestamp))

        return nil

    case "monitor-record":
        
        timestamp  := clRsp.Int64("timestamp")
        valMap, ok := clRsp.Args()["valueMap"].(map[string] interface{})
        Assert(ok, "Malformed value map")
        per        := clRsp.Int32("per")
        srv.RecordValueMap(clId, timestamp, valMap, per)

        // Meta
        srv.UpdateClientMetaLastConnection(clId, timestamp)

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