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

    executablePath string
)

func setFlags() (err error) {

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
    defer CatchFunc(
        &err, func(is ...interface{}) {
            flag.PrintDefaults()
        },
    )

    //// Port range
    Assert(
        flClientPort >= 1 && flClientPort <= 65535,
        fmt.Sprintf("Bad port: %d", flClientPort),
    )

    //// Client
    if !flServer {
        Assert(
            flClientHostname == "",
            "Hostname must be given",
        )
    }

    return nil

}

const (
    ThreadMain = iota
    ThreadCleanUp
)
var HoldSwitch *together.HoldSwitch
func registerSignalHandler() {
    
    // Thread-related
    HoldSwitch = together.NewHoldSwitch()

    // Signal Catcher
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <- sig
        fmt.Println()
        EventLogger.Infoln("Waiting for tasks to finish...")
        HoldSwitch.Add(ThreadCleanUp, 1)
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

    AccessLogger = &log.Logger{}
    AccessLogger.AddWriter(os.Stderr, true)

    EventLogger = &log.Logger{}
    EventLogger.AddWriter(os.Stderr, true)
    EventLogFile, err = log.NewFile("event")
    Try(err)
    EventLogger.AddWriter(EventLogFile, false)

    return nil

}

func main() {

    var err error
    defer CatchFunc(&err, EventLogger.Fatalln)

    // Prepare
    Try(registerLoggers())
    registerSignalHandler()
    setFlags()
    log.Debug = flDebug

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
        AccessLogger.AddWriter(AccessLogFile, false)

        srv := NewServer()
        srv.Start()

    }

}