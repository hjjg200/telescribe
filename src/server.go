package main

import (
    "bufio"
    "bytes"
    "crypto/rsa"
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
    "./secret"
)

const (
    monitorDataCacheExt = ".cache"
)

type Server struct {
    config ServerConfig
    httpListener net.Listener
    telescribeListener net.Listener
    serverPrivateKey *rsa.PrivateKey // Randomly issued private key for the current session
    clientPublicKey map[string] *rsa.PublicKey // 
    authPrivateKey *rsa.PrivateKey
    authFingerprint string
    cachedExecutable []byte
    cachedExecutableSha256Signature []byte
    clientMonitorData map[string] map[string] MonitorDataSlice
    graphDataComposite GrpahDataComposite
    graphDataCompositeJson []byte
}

type ServerConfig struct {
    // General
    AuthPrivateKeyPath string `json:"authPrivateKeyPath"`
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
    ClientConfigs map[string] ClientConfig `json:"clientConfigs"` // Whitelisted clients
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
    ClientConfigs: map[string] ClientConfig{
        "localhost": {
            Alias: "example",
            Comment: "This is an example.",
            MonitorInfos: map[string] MonitorInfo{
                "general-cpuUsage": MonitorInfo{
                    FatalRange: "80.8:100.0,0:5.0",
                    WarningRange: "22.5:55",
                },
            },
            MonitorInterval: 60,
        },
    },
}

type ServerInstance struct {
    parent *Server
    host string
}

func NewServer( prt int ) *Server {
    srv := &Server{}
    srv.clientPublicKey = make( map[string] *rsa.PublicKey )
    srv.clientMonitorData = make( map[string] map[string] MonitorDataSlice )
    return srv
}

func ( srv *Server ) NewInstance( host string ) *ServerInstance {
    return &ServerInstance{
        parent: srv,
        host: host,
    }
}

func ( si *ServerInstance ) MyPrivateKey() *rsa.PrivateKey {
    return si.parent.serverPrivateKey
}

func ( si *ServerInstance ) MyPublicKey() *rsa.PublicKey {
    return &si.parent.serverPrivateKey.PublicKey
}

func ( si *ServerInstance ) TheirPublicKey() *rsa.PublicKey {
    return si.parent.clientPublicKey[si.host]
}

func ( si *ServerInstance ) SetTheirPublicKey( rsaPub *rsa.PublicKey ) {
    si.parent.clientPublicKey[si.host] = rsaPub
}

func ( si *ServerInstance ) Parent() *Server {
    return si.parent
}

func ( si *ServerInstance ) Host() string {
    return si.host
}

func ( si *ServerInstance ) ClientConfig() ClientConfig {
    clCfg, ok := si.Parent().config.ClientConfigs[si.Host()]
    if !ok {
        panic( "No client config" )
    }
    return clCfg
}

func ( srv *Server ) LoadConfig( p string ) error {

    // Load Default First
    srv.config = DefaultServerConfig

    f, err := os.OpenFile( p, os.O_RDONLY, 0644 )
    if err != nil && !os.IsNotExist( err ) {
        return err
    } else if err == nil {
        dec := json.NewDecoder( f )
        err = dec.Decode( &srv.config )
        if err != nil {
            return err
        }
        f.Close()
    }

    // Save config
    f2, err := os.OpenFile( p, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0644 )
    if err != nil {
        return err
    }
    enc := json.NewEncoder( f2 )
    enc.SetIndent( "", "  " )
    err = enc.Encode( srv.config )
    if err != nil {
        return err
    }
    f2.Close()
    return nil

}

func ( srv *Server ) cacheExecutable() error {
    
    f, err := os.OpenFile( executablePath, os.O_RDONLY, 0644 )
    if err != nil {
        return err
    }
    defer f.Close()

    buf := bytes.NewBuffer( nil )
    io.Copy( buf, f )

    srv.cachedExecutable = buf.Bytes()
    srv.cachedExecutableSha256Signature = secret.Sign(
        srv.authPrivateKey, Sha256Sum( srv.cachedExecutable )[:],
    )

    return nil

}

func ( srv *Server ) checkAuthPrivateKey() error {

    apk := srv.config.AuthPrivateKeyPath
    st, err := os.Stat( apk )

    switch {
    case err != nil && !os.IsNotExist( err ):
        // Panic
        return err
    case os.IsNotExist( err ):
        // Not exists
        Logger.Infoln( "Server authentication private key does not exist." )
        Logger.Infoln( "Creating a new one at", apk )
        f, err := os.OpenFile( apk, os.O_WRONLY | os.O_CREATE, 0400 )
        if err != nil {
            return err
        }
        srv.authPrivateKey = secret.RandomPrivateKey()
        f.Write( []byte( secret.SerializePrivateKey( srv.authPrivateKey ) ) )
        f.Close()
        Logger.Infoln( "Issued a new private key for signature authentication." )
        return nil
    default:
        // Exists
        if st.Mode() != 0400 {
            return fmt.Errorf( "The server authentication private key is in a wrong permission mode. Please set it to 400." )
        }
        Logger.Infoln( "Reading the server authentication private key..." )
        f, err := os.OpenFile( apk, os.O_RDONLY, 0400 )
        if err != nil {
            return err
        }
        buf := make( []byte, st.Size() )
        _, err = f.Read( buf )
        if err != nil {
            return err
        }
        srv.authPrivateKey, err = secret.DeserializePrivateKey( string( buf ) )
        if err != nil {
            return err
        }
        Logger.Infoln( "Successfully loaded the server authentication private key." )
        return nil
    }

}

func ( srv *Server ) Start() error {

    // Config
    err := srv.LoadConfig( flServerConfigPath )
    if err != nil {
        return err
    }
    Logger.Infoln( "Loaded Config" )

    // Session Key
    srv.serverPrivateKey = secret.RandomPrivateKey()
    Logger.Infoln( "Issued a Random Key for the Current Session" )

    // Authentication
    err = srv.checkAuthPrivateKey()
    if err != nil {
        return err
    }
    srv.authFingerprint = secret.FingerprintPublicKey( &srv.authPrivateKey.PublicKey )
    Logger.Infoln( "The fingerprint of the authentication public key is:" )
    Logger.Infoln( srv.authFingerprint )

    // Cache Executable
    err = srv.cacheExecutable()
    if err != nil {
        return err
    }
    Logger.Infoln( "Cached Executable for Auto-Update" )

    // Read Monitor Data Cache
    err = srv.readCachedMonitordItems()
    if err != nil {
        return err
    }
    Logger.Infoln( "Read the Cached Monitored Items" )

    // Ensure Directories
    err = EnsureDirectory( srv.config.MonitorDataCacheDir )
    if err != nil {
        return err
    }
    Logger.Infoln( "Ensured Necessary Directories" )

    // Network
    addr := fmt.Sprintf( "%s:%d", srv.config.Bind, srv.config.Port )
    ln, err := net.Listen( "tcp", addr )
    if err != nil {
        return err
    }
    Logger.Infoln( "Network Configured to Listen at", addr )

    // Sub Listener
    srv.telescribeListener, err = net.Listen( "tcp", "127.0.0.1:0" )
    if err != nil {
        return err
    }
    Logger.Infoln( "Sub Listener Started" )

    // Flush cache
    go func() {
        for {
            time.Sleep( time.Minute * time.Duration( srv.config.MonitorDataCacheInterval ) )
            goErr := srv.FlushCachedMonitoredItems()
            if goErr != nil {
                Logger.Warnln( goErr )
            }
            Logger.Infoln( "Flushed Client MonitorData Cache" )
        }
    }()
    Logger.Infoln( "Started Monitor Data Caching Thread" )

    // Graph-ready Data Preparing Thread
    go func() {
        //graphClientMonitorData
        for {

            ca := make( map[string] string )
            cmd := make( map[string] map[string] MonitorDataSlice )
            for host, mdsMap := range srv.clientMonitorData {
                ca[host] = srv.config.ClientConfigs[host].Alias
                cmd[host] = make( map[string] MonitorDataSlice )
                for key, mds := range mdsMap {
                    cmd[host][key] = LTTBMonitorDataSlice(
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
            var err error
            srv.graphDataCompositeJson, err = json.Marshal( srv.graphDataComposite )
            if err != nil {
                Logger.Warnln( err )
            } else {
                Logger.Infoln( "Cached Graph-ready Data" )
            }
            time.Sleep( time.Minute * time.Duration( srv.config.GraphDecimationInterval ) )
        }
    }()
    Logger.Infoln( "Started Data Decimation Thread" )

    // Http
    go srv.startHttpServer()
    Logger.Infoln( "Started HTTP Server" )

    // Telescribe
    go func() {
        for {
            conn, err := srv.telescribeListener.Accept()
            if err != nil {
                Logger.Warnln( err )
                continue
            }

            host, err := readStringPacket( conn )
            if err != nil {
                Logger.Warnln( err )
                continue
            }

            go srv.NewInstance( host ).HandleClientConnection( conn )

            time.Sleep( time.Duration( 1000.0 / float64( srv.config.Tickrate ) ) * time.Millisecond )
        }
    }()
    Logger.Infoln( "Started Telescribe Server" )

    Logger.Infoln( "Successfully Started the Server" )

    copyIO := func( dest, src net.Conn ) {
        defer src.Close()
        defer dest.Close()
        io.Copy( dest, src )
    }
    forwardConnection := func( pre []byte, src net.Conn, addr string ) {
        proxy, err := net.Dial( "tcp", addr )
        if err != nil {
            return
        }
        startLine := strings.Split( string( pre ), "\n" )[0]
        switch {
        case strings.Contains( startLine, "HTTP" ):
        case strings.Contains( startLine, "TELESCRIBE" ):
            // Write Remote Host
            host, err := HostnameOf( src )
            if err != nil {
                return
            }
            proxy.Write( packStringPacket( host ) )
        }
        proxy.Write( pre ) // Write the bytes that are already read
        go copyIO( src, proxy )
        go copyIO( proxy, src )
    }
    for {

        time.Sleep( time.Duration( 1000.0 / float64( srv.config.Tickrate ) ) * time.Millisecond )

        conn, err := ln.Accept()
        if err != nil {
            Logger.Warnln( err )
            continue
        }

        rd := bufio.NewReader( conn )
        startLine, err := rd.ReadString( '\n' ) // Start line
        if err != nil {
            Logger.Warnln( err )
            continue
        }

        rest, err := rd.Peek( rd.Buffered() ) // Read rest bytes without advancing the reader
        if err != nil {
            Logger.Warnln( err )
            continue
        }

        var destAddr string
        switch {
        case strings.Contains( startLine, "HTTP" ):
            destAddr = srv.httpListener.Addr().String()
        case strings.Contains( startLine, "TELESCRIBE" ):
            destAddr = srv.telescribeListener.Addr().String()
        default:
            continue
        }

        go forwardConnection( append( []byte( startLine ), rest... ), conn, destAddr )

    }

}

func ( srv *Server ) readCachedMonitordItems() ( err error ) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf( "%v", r )
        }
    }()

    matches, err := filepath.Glob( srv.config.MonitorDataCacheDir + "/*" + monitorDataCacheExt )
    if err != nil {
        return
    }

    for _, match := range matches {

        f, forErr := os.OpenFile( match, os.O_RDONLY, 0644 )
        if forErr != nil {
            Logger.Warnln( forErr )
            continue
        }
        buf := bytes.NewBuffer( nil )
        io.Copy( buf, f )
        f.Close()

        //
        base := filepath.Base( match )
        ext := filepath.Ext( base )
        skeyString := base[:len( base ) - len( ext )]
        skey, forErr := hex.DecodeString( skeyString )
        if forErr != nil {
            Logger.Warnln( forErr )
            continue
        }

        //
        var (
            host, key string
        )
        rd := bytes.NewReader( skey )
        dec := gob.NewDecoder( rd )
        dec.Decode( &host )
        dec.Decode( &key )

        //
        mds, forErr := DecompressMonitorDataSlice( buf.Bytes() )
        if forErr != nil {
            Logger.Warnln( forErr )
            continue
        }

        //
        _, ok := srv.clientMonitorData[host]
        if !ok {
            srv.clientMonitorData[host] = make( map[string] MonitorDataSlice )
        }
        srv.clientMonitorData[host][key] = mds

    }

    return

}

func ( srv *Server ) FlushCachedMonitoredItems() ( err error ) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf( "%v", r )
        }
    }()

    for host, mdsMap := range srv.clientMonitorData {
        
        for key, mds := range mdsMap {
            
            skey := bytes.NewBuffer( nil )
            enc := gob.NewEncoder( skey )
            enc.Encode( host )
            enc.Encode( key )
            fn := srv.config.MonitorDataCacheDir + "/" + fmt.Sprintf( "%x", skey.Bytes() ) + monitorDataCacheExt

            f, forErr := os.OpenFile( fn, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0644 )
            if forErr != nil {
                Logger.Warnln( forErr )
                continue
            }

            cmp, forErr := CompressMonitorDataSlice( mds )
            if forErr != nil {
                Logger.Warnln( forErr )
                continue
            }

            f.Write( cmp )
            f.Close()

        }

    }

    return

}

func ( si *ServerInstance ) RecordMonitorData( md map[string] interface{} ) {

    //
    srv := si.Parent()
    _, ok := srv.clientMonitorData[si.Host()]
    if !ok {
        srv.clientMonitorData[si.Host()] = make( map[string] MonitorDataSlice )
    }

    //
    initialized := make( []MonitorDataSliceElem, 0 )
    timestamp := time.Now().Unix()

    appendValue := func( key string, val float64 ) {
        short, ok := srv.clientMonitorData[si.Host()][key]
        if !ok {
            short = initialized
        }
        if len( short ) > srv.config.MaxDataLength {
            // Get MaxLength - 1 items
            start := len( short ) - srv.config.MaxDataLength + 1
            short = short[start:]
        }

        srv.clientMonitorData[si.Host()][key] = append(
            short,
            MonitorDataSliceElem{
                Value: val,
                Timestamp: timestamp,
            },
        )
    }

    for key, val := range md {
        switch cast := val.( type ) {
        case float64:
            appendValue( key, cast )
        case map[string] float64:
            for subKey, subVal := range cast {
                longKey := fmt.Sprintf( "%s[%s]", key, subKey )
                appendValue( longKey, subVal )
            }
        }
    }

}

func ( srv *Server ) getMonitorInfo( host, key string ) ( MonitorInfo ) {

    aBase, aParam, aIdx := ParseMonitorrKey( key )
    var bpMatch MonitorInfo

    for b, mi := range srv.config.ClientConfigs[host].MonitorInfos {
        bBase, bParam, bIdx := ParseMonitorrKey( b )
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

func ( si *ServerInstance ) HandleClientConnection( conn net.Conn ) {

    //
    whitelisted := false
    for k := range si.Parent().config.ClientConfigs {
        if k == si.Host() {
            whitelisted = true
            break
        }
    }
    if !whitelisted {
        WriteJsonResponse(
            si, conn,
            Response{
                Name: "not-whitelisted",
            },
        )
        Logger.Infoln( si.Host(), "[non-whitelisted] tried to establish a connection" )
        return
    }

    Logger.Infoln( si.Host(), "connected" )
    rsp, err := ReadNextJsonResponse( si, conn )
    if err != nil {
        WriteJsonResponse(
            si, conn,
            Response{
                Name: "session-expired",
            },
        )
        Logger.Infoln( si.Host(), "session expiry" )
        return
    }

    // TODO expired-session

    switch rsp.Name {
    case "monitor-data":

        // TODO version check
        if rsp.String( "version" ) != Version {
            // ...
            Logger.Warnln( si.Host(), "VERSION MISMATCH" )
            err = WriteJsonResponse(
                si, conn,
                Response{
                    Name: "version-mismatch",
                    Args: map[string] interface{} {
                        "sha256Signature": si.Parent().cachedExecutableSha256Signature,
                        "executable": si.Parent().cachedExecutable,
                    },
                },
            )
            if err != nil {
                return
            }
            return
        }

        //miCfg := si.ClientConfig().MonitorInfos
        md, ok := rsp.Args["monitorData"].( map[string] interface{} )
        if !ok {
            return
        }
        si.RecordMonitorData( md )

        // OK
        err = WriteJsonResponse(
            si, conn,
            Response{
                Name: "ok",
            },
        )
        if err != nil {
            return
        }

        return
    case "handshake-initiate":
        Logger.Infoln( si.Host(), "HANDSHAKE INITIATE" )

        // Server PublicKey
        err = WriteJsonResponse(
            si, conn,
            Response{
                Name: "handshake-server-publickey",
                Args: map[string] interface{} {
                    "publicKey": secret.SerializePublicKey( si.MyPublicKey() ),
                    "authPublicKey": secret.SerializePublicKey( &si.Parent().authPrivateKey.PublicKey ),
                },
            },
        )
        if err != nil {
            return
        }

         // Version Challenge
        Logger.Infoln( si.Host(), "VERSION CHALLENGE" )
        rsp, err := ReadNextJsonResponse( si, conn )
        if err != nil {
            return
        }
        if rsp.Name != "handshake-version-challenge" {
            return
        }
        ver := rsp.String( "version" )
        if ver != Version {
            Logger.Warnln( si.Host(), "VERSION MISMATCH" )
            err = WriteJsonResponse(
                si, conn,
                Response{
                    Name: "handshake-version-challenge-response",
                    Args: map[string] interface{} {
                        "result": "mismatch",
                        "sha256Signature": si.Parent().cachedExecutableSha256Signature,
                        "executable": si.Parent().cachedExecutable,
                    },
                },
            )
            Logger.Infoln( si.Host(), "TERMINATING CONNECTION" )
            return
        } else {
            err = WriteJsonResponse(
                si, conn,
                Response{
                    Name: "handshake-version-challenge-response",
                    Args: map[string] interface{} {
                        "result": "match",
                    },
                },
            )
            if err != nil {
                return
            }
        }

        // Authentication Challenge
        Logger.Infoln( si.Host(), "AUTHENTICATION CHALLENGE" )
        rsp, err = ReadNextJsonResponse( si, conn )
        if err != nil {
            return
        }
        if rsp.Name != "handshake-authentication-challenge" {
            return
        }
        msg := rsp.Bytes( "message" )
        signature := secret.Sign( si.Parent().authPrivateKey, msg )
        
        err = WriteJsonResponse(
            si, conn,
            Response{
                Name: "handshake-challenge-response",
                Args: map[string] interface{} {
                    "signature": signature,
                },
            },
        )
        if err != nil {
            return
        }

        // Client public key
        rsp, err = ReadNextJsonResponse( si, conn )
        if err != nil {
            return
        }
        if rsp.Name != "handshake-client-publickey" {
            return
        }
        serializedPublicKey := rsp.String( "publicKey" )
        rsaPub, err := secret.DeserializePublicKey( serializedPublicKey )
        if err != nil {
            return
        }
        si.SetTheirPublicKey( rsaPub )

        // Ping
        err = WriteEncryptedJsonResponse(
            si, conn,
            Response{
                Name: "handshake-ping",
            },
        )
        if err != nil {
            return
        }

        // Pong
        rsp, err = ReadNextJsonResponse( si, conn )
        if err != nil {
            return
        }
        if rsp.Name != "handshake-pong" {
            return
        }

        // Client Config
        ccfg := si.Parent().config.ClientConfigs[si.Host()]
        j, err := json.Marshal( ccfg )
        if err != nil {
            return
        }
        sha256Signature := secret.Sign( si.Parent().authPrivateKey, Sha256Sum( j )[:] )
        err = WriteEncryptedJsonResponse(
            si, conn,
            Response{
                Name: "handshake-client-config",
                Args: map[string] interface{} {
                    "config": j,
                    "sha256Signature": sha256Signature,
                },
            },
        )

        Logger.Infoln( si.Host(), "SUCCESSFUL HANDSHAKE" )

    }

}
