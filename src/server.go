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
    monitorDataCacheExt = ".cache"
    clientConfigWatchInterval = time.Second * 5
)

type Server struct {
    config ServerConfig
    clientConfigCluster ClientConfigCluster
    clientConfigVersion map[string] string // [fullName] => sha256[:6]
    httpListener net.Listener
    authFingerprint string

    //
    monitorDataTables MonitorDataTables
    clientMonitorData map[string] map[string] MonitorDataSlice
    cachedExecutable []byte
}

type ServerConfig struct {
    // General
    AuthPrivateKeyPath string `json:"authPrivateKeyPath"`
    ClientConfigPath string `json:"clientConfigPath"`
        // TODO Log latest.log, 20190102.log ...
    // Http
    HttpUsername string `json:"http.username"`
    HttpPassword string `json:"http.password(sha256)"`
    HttpCertFilePath string `json:"http.certFilePath"` // For TLS
    HttpKeyFilePath string `json:"http.keyFilePath"` // For TLS
    // Monitor
    MonitorDataCacheInterval int `json:"monitor.dataCacheInterval(minutes)"`
    MonitorDataCacheDir string `json:"monitor.dataCacheDir"`
    MaxDataLength int `json:"monitor.maxDataLength"`
    GraphPointThreshold int `json:"monitor.graphThreshold"`
    GraphDecimationInterval int `json:"monitor.graphDecimationInterval(minutes)"`
    // Graph (HTML)
    Graph GraphOptions `json:"graph"`
    // Network
    Bind string `json:"network.bind"`
    Port int `json:"network.port"`
    Tickrate int `json:"network.tickrate(hz)"` // How many times will the server process connections in a second (Hz)
    // Alarm
    WebhookUrl string `json:"alarm.webhookUrl"`
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
    MonitorDataCacheInterval: 1,
    MonitorDataCacheDir: "./serverCache.d",
    MaxDataLength: 43200, // 30 days for 1-minute interval
    GraphPointThreshold: 1500,
    GraphDecimationInterval: 10,
    // Graph (HTML)
    Graph: DefaultGraphOptions,
    // Network
    Bind: "127.0.0.1",
    Port: 1226,
    Tickrate: 60,
    // Alarm
    WebhookUrl: "",
}

var DefaultGraphOptions = GraphOptions{
    GapThresholdTime: 30,
    Durations: []int{3*3600, 12*3600, 3*86400, 7*86400, 30*86400},
    FormatNumber: "{e+0.2f}",
    FormatDateLong: "DD HH:mm",
    FormatDateShort: "MM/DD",
}

var DefaultClientConfigCluster = ClientConfigCluster{
    // Client Config
    ClientAliases: map[string] ClientAliasConfig{
        "127.0.0.1": {
            "default": "example",
            "foo": "bar",
        },
    },
    ClientRoles: map[string] ClientRoleConfig{
        "example": {
            MonitorInfos: map[string] MonitorInfo{
                "cpu-usage": MonitorInfo{
                    FatalRange: "80:",
                    WarningRange: "50:",
                },
            },
            MonitorInterval: 60,
        },
        "bar": {
            MonitorInfos: map[string] MonitorInfo{
                "swap-usage": MonitorInfo{},
                "memory-usage": MonitorInfo{},
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

func (srv *Server) loadClientConfigCluster() error {

    fn := srv.config.ClientConfigPath
    
    //
    f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
    switch {
    case err != nil && !os.IsNotExist(err):
        return err
    case os.IsNotExist(err):
        // Save default config
        srv.clientConfigCluster = DefaultClientConfigCluster
        f2, err := os.OpenFile(fn, os.O_WRONLY | os.O_CREATE, 0600)
        if err != nil {
            return err
        }
        enc := json.NewEncoder(f2)
        enc.SetIndent("", "  ")
        err = enc.Encode(DefaultClientConfigCluster)
        if err != nil {
            return err
        }
        f.Close()
    default:
        //
        dec := json.NewDecoder(f)
        ccc := ClientConfigCluster{}
        err = dec.Decode(&ccc)
        if err != nil {
            return err
        }
        srv.clientConfigCluster = ccc
    }

    // Version
    ccv := make(map[string] string)
    roleVersion := make(map[string] string)
    for host, aliases := range srv.clientConfigCluster.ClientAliases {
        for alias, role := range aliases {
            fullName := formatFullName(host, alias)
            if _, ok := roleVersion[role]; !ok {
                j, _ := json.Marshal(srv.clientConfigCluster.ClientRoles[role])
                roleVersion[role] = fmt.Sprintf("%x", Sha256Sum(j))[:6]
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

func (srv *Server) Start() (err error) {

    defer Catch(&err)

    // Config
    Try(srv.LoadConfig(flServerConfigPath))
    Logger.Infoln("Loaded Config")
    Try(srv.loadClientConfigCluster())
    Logger.Infoln("Loaded Client Config Cluster")

    //
    Try(srv.checkAuthPrivateKey())
    Logger.Infoln("The fingerprint of the authentication public key is:")
    Logger.Infoln(sessionAuthPriv.PublicKey.Fingerprint())

    // Cache Executable
    Try(srv.cacheExecutable())
    Logger.Infoln("Cached Executable for Auto-Update")

    // Read Monitor Data Cache
    Try(srv.readCachedMonitoredItems())
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

    // Client Config Version Update
    go func() {
        ccvp := srv.config.ClientConfigPath
        st, _ := os.Stat(ccvp)
        lastMod := st.ModTime()
        for {
            time.Sleep(clientConfigWatchInterval)

            // Mod Time Check
            st, err := os.Stat(ccvp)
            if err != nil {
                Logger.Warnln(err)
                continue
            }
            // Changed
            if lastMod != st.ModTime() {
                goErr := srv.loadClientConfigCluster()
                if goErr != nil {
                    Logger.Warnln(goErr)
                    continue
                }
                Logger.Infoln("Reloaded Client Config Cluster")
                lastMod = st.ModTime()
            }

        }
    }()
    Logger.Infoln("Started Client Config Cluster Reloading Thread")

    // Graph-ready Data Preparing Thread
    go func() {
        // monitorDataTables
        for {

            clients := make(map[string] MDTClient)
            for fullName, mdsMap := range srv.clientMonitorData {
                // MonitorData
                tsMap := make(map[int64] struct{})
                mdss  := make(map[string] []byte)
                for key, mds := range mdsMap {

                    // Decimate monitor data
                    decimated := LTTBMonitorDataSlice(
                        mds, srv.config.GraphPointThreshold,
                    )
                    // Put Gaps
                    table := bytes.NewBuffer(nil)
                    gthSec := int64(srv.config.Graph.GapThresholdTime * 60) // To seconds
                    prevTs := decimated[0].Timestamp
                    fmt.Fprint(table, "timestamp,value\n")
                    for _, each := range decimated {
                        ts := each.Timestamp
                        if ts - prevTs > gthSec {
                            avgTs := (ts + prevTs) / 2
                            tsMap[avgTs] =  struct{}{}
                            fmt.Fprintf(table, "%d,NaN\n", avgTs)
                        }
                        tsMap[ts] =  struct{}{}
                        fmt.Fprintf(table, "%d,%f\n", ts, each.Value)
                        prevTs = ts
                    }
                    // Assign Csv
                    mdss[key] = table.Bytes()

                }

                // Timestamps
                i, ts := 0, make([]int64, len(tsMap))
                for t := range tsMap {
                    ts[i] = t
                    i++
                }
                sort.Sort(Int64Slice(ts))
                tt := bytes.NewBuffer(nil)
                fmt.Fprint(tt, "timestamp\n")
                for _, t := range ts {
                    fmt.Fprintf(tt, "%d\n", t)
                }

                clients[fullName] = MDTClient{
                    Timestamps: tt.Bytes(),
                    MonitorDataSlices: mdss,
                }
            }

            srv.monitorDataTables = clients
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

        go func () {
            host, _ := HostnameOf(conn)
            rd := bufio.NewReader(conn)
            startLine, err := rd.ReadString('\n') // Start line
            if err != nil {
                Logger.Warnln(err)
                return
            }

            rest, err := rd.Peek(rd.Buffered()) // Read rest bytes without advancing the reader
            if err != nil {
                Logger.Warnln(err)
                return
            }

            already := append([]byte(startLine), rest...) // Bytes that are already read

            switch {
            case strings.Contains(startLine, "HTTP"):
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
            case strings.Contains(startLine, "TELESCRIBE"):
                s := NewSession(conn)
                defer s.Close()
                s.PrependRawInput(bytes.NewReader(already))
                err := srv.HandleSession(s)
                if err != nil {
                    Logger.Warnln(err)
                }
            default:
            }
        }()
    }

    err = fmt.Errorf("Server terminated")
    return

}

func (srv *Server) readCachedMonitoredItems() (err error) {

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
        mds, forErr := DecompressMonitorDataSlice(cmp)
        if forErr != nil {
            Logger.Warnln(forErr)
            continue
        }

        //
        _, ok := srv.clientMonitorData[fullName]
        if !ok {
            srv.clientMonitorData[fullName] = make(map[string] MonitorDataSlice)
        }
        srv.clientMonitorData[fullName][key] = mds

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

    for fullName, mdsMap := range srv.clientMonitorData {
        
        for key, mds := range mdsMap {
            
            h := Sha256Sum([]byte(fullName + key))
            fn := srv.config.MonitorDataCacheDir + "/" + fmt.Sprintf("%x", h) + monitorDataCacheExt

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

            enc := gob.NewEncoder(f)
            enc.Encode(fullName)
            enc.Encode(key)
            enc.Encode(cmp)
            f.Close()

        }

    }

    return

}

func (srv *Server) RecordMonitorData(fullName string, timestamp int64, md map[string] interface{}) {

    //
    _, ok := srv.clientMonitorData[fullName]
    if !ok {
        srv.clientMonitorData[fullName] = make(map[string] MonitorDataSlice)
    }

    //
    initialized := make([]MonitorDataSliceElem, 0)
    fatalValues := make(map[string] float64)
    appendValue := func(key string, val float64) {

        short, ok := srv.clientMonitorData[fullName][key]
        if !ok {
            short = initialized
        }

        // Trim Data
        if len(short) > srv.config.MaxDataLength {
            // Get MaxLength - 1 items
            start := len(short) - srv.config.MaxDataLength + 1
            short = short[start:]
        }

        // Append
        srv.clientMonitorData[fullName][key] = append(
            short,
            MonitorDataSliceElem{
                Value: val,
                Timestamp: timestamp,
            },
        )

        // Check Status
        st := srv.getMonitorInfo(fullName, key).StatusOf(val)
        if st == MonitorStatusFatal {
            fatalValues[key] = val
        }

    }

    //
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
    FatalValues map[string] float64 `json:"fatalValues"`
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

func (srv *Server) getMonitorInfos(fullName string) map[string] MonitorInfo {
    host, alias := parseFullName(fullName)
    role := srv.clientConfigCluster.ClientAliases[host][alias]
    roleCfg := srv.clientConfigCluster.ClientRoles[role]
    return roleCfg.MonitorInfos
}

func (srv *Server) getMonitorInfo(fullName, key string) (MonitorInfo) {

    aBase, aParam, aIdx := ParseMonitorrKey(key)
    var bpMatch MonitorInfo

    mis := srv.getMonitorInfos(fullName)
    for b, mi := range mis {
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
    roles, whitelisted := srv.clientConfigCluster.ClientAliases[host]
    if !whitelisted {
        srvRsp := NewResponse("not-whitelisted")
        s.WriteResponse(srvRsp)
        return fmt.Errorf("%s [non-whitelisted] tried to establish a connection", host)
    }

    Logger.Infoln(host, "connected")
    clRsp, err := s.NextResponse()
    Try(err)

    // Role
    alias := clRsp.String("alias")
    fullName := formatFullName(host, alias)
    role, ok := srv.clientConfigCluster.ClientRoles[roles[alias]]
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

    case "monitor-data":

        //
        md, ok := clRsp.Args()["monitorData"].(map[string] interface{})
        Assert(ok, "Malformed monitor data")
        timestamp := clRsp.Int64("timestamp")
        srv.RecordMonitorData(fullName, timestamp, md)

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