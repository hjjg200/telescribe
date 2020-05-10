package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "time"
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

//const Version = "telescribe-alpha-0.12"
var Version string
const (
    childSpawnInterval = time.Second * 10
)

var (
    flServer bool
    flServerConfigPath string

    flClientHostname string
    flClientAlias string
    flClientPort int
    flClientKnownHostsPath string
    flClientDaemon bool

    flDebug bool

    executablePath string
)

func setFlags() {

    flag.BoolVar(&flServer, "server", false, "Run as a server")
    flag.StringVar(&flServerConfigPath, "server-config-path", "./serverConfig.json", "(Server) The path to the server config file. The server configuration must be done in a file rather than in a command.")

    flag.StringVar(&flClientHostname, "host", "", "(Client) The hostname of the server for the client to connect to")
    flag.StringVar(&flClientAlias, "alias", "default", "(Client) The alias of the client")
    flag.IntVar(&flClientPort, "port", 1226, "(Client) The port of the server for the client to connect to")
    flag.StringVar(&flClientKnownHostsPath, "known-hosts-path", "./clientKnownHosts", "(Client) The file that contains all the public key fingerprints of the accepted servers. Crucial for preventing MITM attacks that may exploit the auto update procedure.")
    flag.BoolVar(&flClientDaemon, "daemon", false, "(Client) Whether to run the client as daemon.")

    flag.BoolVar(&flDebug, "debug", false, "(Debug) verbose")
    flag.Parse()

    // Check wrong flags

    //// Port range
    if flClientPort < 1 || flClientPort > 65535 {
        fmt.Printf("Bad port: %d\n\n", flClientPort)
        flag.PrintDefaults()
        os.Exit(1)
    }

    //// Client
    if !flServer {
        if flClientHostname == "" {
            fmt.Println("No hostname was given\n")
            flag.PrintDefaults()
            os.Exit(1)
        }
    }

}

var AccessLogger *log.Logger
var AccessLogFile *log.File
var EventLogger *log.Logger
var EventLogFile *log.File

func main() {

    // Loggers
    AccessLogger = &log.Logger{}
    AccessLogger.AddWriter(os.Stderr, true)

    EventLogger = &log.Logger{}
    EventLogger.AddWriter(os.Stderr, true)
    EventLogFile, err := log.NewFile("event")
    if err != nil {
        EventLogger.Fatalln(err)
    }
    EventLogger.AddWriter(EventLogFile, false)

    //
    setFlags()
    log.Debug = flDebug

    // Executable path
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
    case !flServer: // Client
        if flClientDaemon {
            // Daemon
            Logger.Infoln("Running as daemon...")
            clArgs := strconv.Quote(executablePath)
            flag.Visit(func (f *flag.Flag) {
                if f.Name == "daemon" && f.Value.String() == "true" {
                    // flag.boolValue.String() returns strconv.FormatBool()
                    // Exclude daemon true option
                    return
                }
                clArgs += " "
                clArgs += "-" + f.Name + "="
                clArgs += strconv.Quote(f.Value.String())
                // As golang/flag package allows only the non-boolean flags to be in the form below
                // -flag value
                // clArgs is formatted in the way in which boolean flags also work
                // i.e., -flag=value
            })

            for {
                Logger.Infoln("Spawning a child process...")
                    
                // Spawn Child
                child := exec.Command("bash", "-c", clArgs)
                child.Stderr = os.Stderr
                child.Stdout = os.Stdout
                child.Stdin = os.Stdin
                err := child.Start()
                if err != nil {
                    Logger.Fatalln(err)
                }
                err = child.Wait()
                Logger.Warnln("The child process exited:", err)
                time.Sleep(childSpawnInterval)
            }
        } else {
            // Non-daemon
            addr := fmt.Sprintf("%s:%d", flClientHostname, flClientPort)
            cl := NewClient(addr)
            Logger.Infoln("Starting as a client for", addr)
            Logger.Panicln(cl.Start())
        }
    case flServer: // Server
        // Access Log
        AccessLogFile, err = log.NewFile("access")
        if err != nil {
            EventLogger.Fatalln(err)
        }
        AccessLogger.AddWriter(AccessLogFile, false)

        srv := NewServer()
        srv.Start()
    }

}