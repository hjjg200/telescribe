package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "strconv"
    "syscall"
    "time"
    "./log"

    "github.com/hjjg200/go-together"
    . "github.com/hjjg200/go-act"
)

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
    flDebugFilter string

    executablePath string
)

func setFlags() (err error) {

    // Server flags
    flag.BoolVar(
        &flServer, "server", false, 
        "Run as a server",
    )
    flag.StringVar(
        &flServerConfigPath, "server_config_path", "./serverConfig.json",
        "(Server) The path to the server config file. The server configuration must be done in a file rather than in a command.",
    )

    // Client flags
    flag.StringVar(
        &flClientHostname, "host", "",
        "(Client) The hostname of the server for the client to connect to",
    )
    flag.StringVar(
        &flClientAlias, "alias", "default",
        "(Client) The alias of the client",
    )
    flag.IntVar(
        &flClientPort, "port", 1226, 
        "(Client) The port of the server for the client to connect to",
    )
    flag.StringVar(
        &flClientKnownHostsPath, "known_hosts_path", "./clientKnownHosts",
        "(Client) The file that contains all the public key fingerprints of the accepted servers. Crucial for preventing MITM attacks that may exploit the auto update procedure.",
    )
    flag.BoolVar(
        &flClientDaemon, "daemon", false, 
        "(Client) Whether to run the client as daemon.",
    )

    // Debug flags
    flag.BoolVar(
        &flDebug, "debug", false,
        "(Debug) verbose",
    )
    flag.StringVar(
        &flDebugFilter, "debug_filter", "",
        "(Debug) regular expression for debug category filtering; empty string stands for .*",
    )

    // Parse
    flag.Parse()

    // Check wrong flags
    defer CatchFunc(
        &err, func(is ...interface{}) {
            flag.PrintDefaults()
        },
    )

    // Client
    if !flServer {
        Assert(
            flClientHostname != "",
            "Hostname must be given",
        )
        Assert(
            flClientPort >= 1 && flClientPort <= 65535,
            fmt.Sprintf("Bad port: %d", flClientPort),
        )
    }

    // Debug
    log.Debug = flDebug
    log.DebugFilter, err = log.NewFilter(flDebugFilter)
    Try(err)

    return nil

}

const (
    threadMain = iota
)
var railSwitch *together.RailSwitch
func registerSignalHandler() {
    
    // Thread-related
    railSwitch = together.NewRailSwitch()

    // Signal Catcher
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <- sig
        fmt.Println()
        EventLogger.Infoln("Waiting for tasks to finish...")
        railSwitch.Close()
        EventLogger.Infoln("Bye bye")
        EventLogFile.Close()
        AccessLogFile.Close()
        os.Exit(0)
    }()

}

var AccessLogger *log.Logger
var AccessLogFile *log.File
var EventLogger *log.Logger
var EventLogFile *log.File
func registerLoggers() (err error) {

    defer Catch(&err)

    // Access and Event
    AccessLogger = &log.Logger{}
    AccessLogger.AddWriter(os.Stderr, log.ANSIColorer)

    EventLogger = &log.Logger{}
    EventLogger.AddWriter(os.Stderr, log.ANSIColorer)
    EventLogFile, err = log.NewFile("event")
    Try(err)
    EventLogger.AddWriter(EventLogFile, nil)

    return nil

}

func main() {

    var err error
    defer CatchFunc(&err, EventLogger.Fatalln)

    // Prepare
    Try(registerLoggers())
    registerSignalHandler()
    Try(setFlags())

    // Executable path
    executablePath, err = os.Executable()
    Try(err)
    executablePath, err = filepath.EvalSymlinks(executablePath)
    Try(err)

    switch {
    case !flServer: // Client
        if flClientDaemon {
            // Daemon
            EventLogger.Infoln("Running as daemon...")
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
                EventLogger.Infoln("Spawning a child process...")
                    
                // Spawn Child
                child := exec.Command("bash", "-c", clArgs)
                child.Stderr = os.Stderr
                child.Stdout = os.Stdout
                child.Stdin = os.Stdin
                Try(child.Start())

                EventLogger.Warnln("The child process exited:", child.Wait())
                time.Sleep(childSpawnInterval)
            }
        } else {
            // Non-daemon
            addr := fmt.Sprintf("%s:%d", flClientHostname, flClientPort)
            cl := NewClient(addr)
            EventLogger.Infoln("Starting as a client for", addr)
            EventLogger.Panicln(cl.Start())
        }
    case flServer: // Server

        // Access Log
        AccessLogFile, err = log.NewFile("access")
        Try(err)
        AccessLogger.AddWriter(AccessLogFile, nil)

        srv := NewServer()
        srv.Start()

    }

}