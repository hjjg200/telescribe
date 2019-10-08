package main

import (
    "encoding/json"
    "fmt"
    "net"
    "os"
    "time"
    "./monitor"

    . "github.com/hjjg200/go-act"
)

const defaultHelloRetryInterval = time.Minute * 1

type Client struct {
    serverAddr string
    s *Session
    role ClientRoleConfig
}

type ClientConfigCluster struct {
    ClientAliases map[string] ClientAliasConfig `json:"aliases"`
    ClientRoles map[string] ClientRoleConfig `json:"roles"`
}
type ClientAliasConfig map[string] string // [alias] = role
type ClientRoleConfig struct {
    MonitorInfos map[string] MonitorInfo `json:"monitorInfos"`
    MonitorInterval int `json:"monitorInterval"`
}

func NewClient(serverAddr string) *Client {
    return &Client{
        serverAddr: serverAddr,
    }
}

func (cl *Client) hello() (err error) {

    defer Catch(&err)
    
    // Connection
    conn, err := net.Dial("tcp", cl.serverAddr)
    Try(err)
    Logger.Infoln("HELLO SERVER")

    defer conn.Close()
    s := NewSession(conn)
    cl.s = s
    clRsp := NewResponse("hello")
    clRsp.Set("version", Version)
    clRsp.Set("alias", flClientAlias)
    Try(s.WriteResponse(clRsp))

    // Config
    srvRsp, err := s.NextResponse()
    Try(err)

    switch srvRsp.Name() {
    case "hello":
        role := srvRsp.Bytes("role")
        return json.Unmarshal(role, &cl.role)
    case "version-mismatch":
        executable := srvRsp.Bytes("executable")
        return cl.autoUpdate(executable)
    default:
        return fmt.Errorf("Bad config response")
    }

}

func (cl *Client) checkKnownHosts() error {
    return LoadKnownHosts(flClientKnownHostsPath)
}

func (cl *Client) autoUpdate(executable []byte) error {

    Logger.Infoln("Started Auto Update Procedure.")
    Logger.Infoln("The service must be set to automatically restart.")

    // Remove
    err := os.Remove(executablePath)
    if err != nil {
        return err
    }

    // Write New Executable
    f, err := os.OpenFile(executablePath, os.O_CREATE | os.O_WRONLY, 0755)
    if err != nil {
        return err
    }
    _, err = f.Write(executable)
    if err != nil {
        return err
    }

    f.Close()

    Logger.Infoln("Successfully updated the executable.")
    Logger.Infoln("Exiting the application...")
    os.Exit(1)
    return nil

}

func (cl *Client) Start() error {

    err := cl.checkKnownHosts()
    if err != nil {
        return err
    }

    //
    hri := defaultHelloRetryInterval

    for {

        err = cl.hello()
        if err != nil {
            Logger.Warnln("Hello Failed:", err)
            time.Sleep(hri)
            continue
        }

        Logger.Infoln("SUCCESSFUL HELLO")
        // Config
        monitorInterval := time.Second * time.Duration(cl.role.MonitorInterval)
        hri = monitorInterval
        pass := make(chan struct{})
        go func() {
            pass <- struct{}{}
        }()

        // Loop
        MonitorLoop:
        for {

            var conn net.Conn
            <- pass // Wait

            // Sleep
            go func() {
                time.Sleep(monitorInterval)
                pass <- struct{}{}
            }()

            // Connection
            if conn != nil {
                conn.Close()
                conn = nil
            }
            conn, err := net.Dial("tcp", cl.serverAddr)
            if err != nil {
                Logger.Warnln("Server Not Responding")
                continue
            }
            cl.s.SetConn(conn)

            md := make(map[string] interface{})
            for k := range cl.role.MonitorInfos {
                getter, ok := monitor.Getter(k)
                if !ok {
                    md[k] = nil
                    continue
                }
                got := getter()
                for i, v := range got {
                    md[i] = v
                }
            }

            // Send to Server
            clRsp := NewResponse("monitor-data")
            clRsp.Set("version", Version)
            clRsp.Set("alias", flClientAlias)
            clRsp.Set("timestamp", time.Now().Unix())
            clRsp.Set("monitorData", md)
            err = cl.s.WriteResponse(clRsp)
            if err != nil {
                Logger.Warnln(err)
                continue
            }
            Logger.Infoln("Sent Data")

            //
            srvRsp, err := cl.s.NextResponse()
            if err != nil {
                Logger.Warnln(err)
                continue
            }

            switch srvRsp.Name() {
            case "ok":
            // case "reconfigure":
            case "version-mismatch":
                Logger.Warnln("Version mismatch! Attempting to auto-update...")
                executable := srvRsp.Bytes("executable")

                Logger.Fatalln(cl.autoUpdate(executable))
            case "session-expired":
                break MonitorLoop
            }

        }

    }

    return nil

}