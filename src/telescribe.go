package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "./log"
)

/*

Check
Get
Set
Add
Put
Remove
Update
Read
Ensure

*/

// # Handshake
// Client -> Server
// : {
// :   "name": "handshake-initiate",
// :   "version": "telescribe-..."
// : }
// Server -> Client
// : Check the hostname is in the whitelist
// : if not
// : {
// :   "name": "not-whitelisted"
// : }
// : if in whitelist
// : if version matched
// : {
// :   "name": "handshake-server-publickey",
// :   "publicKey": "..."
// : }
// : if not
// : {
// :   "name": "version-mismatch",
// :   "executable": "..."
// : }
// : exits
// Client -> Server
// : {
// :   "name": "handshake-client-publickey",
// :   "publicKey": "..."
// : }
// Server -> Client (It is now encrypted from here)
// : {
// :   "name": "handshake-ping"
// : }
// Client -> Server
// : {
// :   "name": "handshake-pong"
// : }
// Server -> Client
// : {
// :   "name": "handshake-config",
// :   "config": "..."
// : }
// Client -> Server
// : Load the config and send the relevant data periodically
// : {
// :   "name": "monitored-items",
// :   "version": "telescribe-...",
// :   "timestamp": ...,
// :   "monitoredItems": "..."
// : }
// Server -> Client
// : Server does not respond unless...
// : Version is different
// : {
// :   "name": "version-mismatch",
// :   "executable": "..."
// : }
// : Session restarted (public key does not work)
// : {
// :   "name": "session-restarted"
// : }

const Version = "telescribe-alpha-0.5"

var (
    flServer bool
    flServerConfigPath string

    flClient bool
    flClientHostname string
    flClientPort int
    flClientKnownHostsPath string

    flDebug bool

    executablePath string
)

func setFlags() {

    flag.BoolVar(&flServer, "server", false, "Run as a server")
    flag.StringVar(&flServerConfigPath, "server-config-path", "./serverConfig.json", "(Server) The path to the server config file. The server configuration must be done in a file rather than in a command.")

    flag.BoolVar(&flClient, "client", false, "Run as a client")
    flag.StringVar(&flClientHostname, "client-host", "", "(Client) The hostname of the server for the client to connect to")
    flag.IntVar(&flClientPort, "client-port", 1226, "(Client) The port of the server for the client to connect to")
    flag.StringVar(&flClientKnownHostsPath, "client-known-hosts-path", "./clientKnownHosts", "(Client) The file that contains all the public key fingerprints of the accepted servers. Crucial for preventing MITM attacks that may exploit the auto update procedure.")
    
    flag.BoolVar(&flDebug, "debug", false, "(Debug) verbose")
    flag.Parse()

    // Check wrong flags
    
    //// Client or server
    if !flServer && !flClient {
        fmt.Println("-server or -client must be given\n")
        flag.PrintDefaults()
        os.Exit(1)
    }

    //// Mutually exclusive
    if flServer && flClient {
        fmt.Println("-server and -client are mutually exclusive\n")
        flag.PrintDefaults()
        os.Exit(1)
    }

    //// Port range
    if flClientPort < 1 || flClientPort > 65535 {
        fmt.Printf("Bad port: %d\n\n", flClientPort)
        flag.PrintDefaults()
        os.Exit(1)
    }

    //// Client
    if flClient {
        if flClientHostname == "" {
            fmt.Println("No hostname was given\n")
            flag.PrintDefaults()
            os.Exit(1)
        }
    }

}

var Logger *log.Logger

func main() {

    // Loggers
    Logger = &log.Logger{}
    Logger.AddWriter(os.Stderr, true)

    //
    setFlags()
    log.Debug = flDebug

    // Cache
    var err error
    executablePath, err = os.Executable()
    if err != nil {
        Logger.Fatalln(err)
    }
    executablePath, err = filepath.EvalSymlinks(executablePath)
    if err != nil {
        Logger.Fatalln(err)
    }

    switch {
    case flClient:
        addr := fmt.Sprintf("%s:%d", flClientHostname, flClientPort)
        cl := NewClient(addr)
        Logger.Infoln("Starting as a client for", addr)
        Logger.Panicln(cl.Start())
    case flServer:
        srv := NewServer()
        Logger.Panicln(srv.Start())
    }

}