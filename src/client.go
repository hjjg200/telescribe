package main

import (
    "encoding/json"
    "fmt"
    "net"
    "os"
    "time"
    "./monitor"

    . "github.com/hjjg200/go-act"
    "github.com/hjjg200/go-together"
)

const defaultHelloRetryInterval = time.Minute * 1

type Client struct {
    serverAddr    string
    s             *Session
    role          ClientRole
    configVersion string
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
    defer conn.Close()
    EventLogger.Infoln("HELLO SERVER")

    // Session
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
        return cl.configureRole(
            srvRsp.String("configVersion"),
            srvRsp.Bytes("role"), 
        )
    case "version-mismatch":
        return cl.autoUpdate(srvRsp.Bytes("executable"))
    default:
        return fmt.Errorf("Bad config response")
    }

}

func (cl *Client) configureRole(cv string, role []byte) error {
    cl.configVersion = cv
    cl.role = ClientRole{}
    return json.Unmarshal(role, &cl.role)
}

func (cl *Client) checkKnownHosts() error {
    return LoadKnownHosts(flClientKnownHostsPath)
}

func (cl *Client) autoUpdate(executable []byte) error {

    EventLogger.Infoln("Started Auto Update Procedure.")
    EventLogger.Infoln("The service must be set to automatically restart.")

    // Remove the current executable
    // -> You need to unlink(rm) the runinng executable file first in order to replace it with a new one
    err := os.Remove(executablePath)
    if err != nil {
        return err
    }

    // Write New Executable
    f, err := os.OpenFile(executablePath, os.O_CREATE | os.O_WRONLY, 0755)
    if err != nil {
        return err
    }
    n, err := f.Write(executable)
    if err != nil {
        return err
    }

    EventLogger.Infoln("Written bytes:", n)
    f.Close()

    EventLogger.Infoln("Successfully updated the executable.")
    EventLogger.Infoln("Exiting the application...")
    os.Exit(1)
    // APP EXITED
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
            EventLogger.Warnln("Hello Failed:", err)
            time.Sleep(hri)
            continue
        }

        EventLogger.Infoln("SUCCESSFUL HELLO")
        // Config
        // + Monitor Interval Shorthand Func
        mrif   := func() time.Duration { return time.Second * time.Duration(cl.role.MonitorInterval) }
        hri     = mrif()
        passer := together.NewPasser(mrif())

        // Loop
        MonitorLoop:
        for {

            var conn net.Conn
            passer.Pass()

            // Connection
            if conn != nil {
                conn.Close()
                conn = nil
            }
            conn, err := net.Dial("tcp", cl.serverAddr)
            if err != nil {
                EventLogger.Warnln("Server Not Responding")
                continue
            }
            cl.s.SetConn(conn)

            // Monitored values
            valMap := make(map[string] interface{})
            mapLen := 0
            for rawKey := range cl.role.MonitorConfigMap {
                getter, ok := monitor.Getter(string(rawKey))
                if !ok {
                    valMap[rawKey] = nil
                    continue
                }
                got := getter()
                for key, val := range got {
                    valMap[key] = val
                    mapLen++
                }
            }

            // Send to Server
            clRsp := NewResponse("monitor-record")
            clRsp.Set("version", Version)
            clRsp.Set("configVersion", cl.configVersion)
            clRsp.Set("alias", flClientAlias)
            clRsp.Set("timestamp", time.Now().Unix())
            clRsp.Set("valueMap", valMap)
            err = cl.s.WriteResponse(clRsp)
            if err != nil {
                EventLogger.Warnln(err)
                continue
            }
            AccessLogger.Infoln("Sent Data:", mapLen, "items")

            // Get response
            srvRsp, err := cl.s.NextResponse()
            if err != nil {
                EventLogger.Warnln(err)
                continue
            }

            switch srvRsp.Name() {
            case "ok":
            case "reconfigure":
                err = cl.configureRole(
                    srvRsp.String("configVersion"),
                    srvRsp.Bytes("role"), 
                )
                if err != nil {
                    EventLogger.Warnln(err)
                    break MonitorLoop
                }
                // Passer Interval
                passer.SetInterval(mrif())
                EventLogger.Infoln("Reconfigured!")
            case "version-mismatch":
                EventLogger.Warnln("Version mismatch! Attempting to auto-update...")
                cl.autoUpdate(srvRsp.Bytes("executable"))
            case "session-expired":
                break MonitorLoop
            }

        }

    }

    return nil

}


// INFO ---

type ClientInfo struct {
    Host  string `json:"host"`
    Alias string `json:"alias"`
    Role  string `json:"role"`
    ips   []net.IP
}

type ClientInfoMap map[string/* clientId */] ClientInfo

// Returns whether the client's host resolves to the same ip addresses
// as the ip addresses to which the given address resolves
func(clInfo *ClientInfo) HasAddr(addr string) bool {

    if clInfo.ips == nil {
        var err error
        clInfo.ips, err = net.LookupIP(clInfo.Host)
        if err != nil {
            EventLogger.Warnln("Failed to lookup ip for the client host:", clInfo.Host)
            return false
        }
    }

    addrIps, err := net.LookupIP(addr)
    if err != nil {
        EventLogger.Warnln("Failed to lookup ip for the address:", addr)
        return false
    }

    for _, lhs := range clInfo.ips {
        for _, rhs := range addrIps {
            if lhs.Equal(rhs) {
                return true
            }
        }
    }
    
    return false

}

// ROLE ---

type ClientRole struct { // clRole
    MonitorConfigMap map[string/* mKey */] MonitorConfig `json:"monitorConfigMap"`
    MonitorInterval  int                       `json:"monitorInterval"`
}

func(clRole ClientRole) Version() string {
    j, _ := json.Marshal(clRole)
    return fmt.Sprintf("%x", Sha256Sum(j))[:6]
}
func(clRole ClientRole) Merge(rhs ClientRole) ClientRole {
    lhs := clRole
    // MonitorConfigMap
    if lhs.MonitorConfigMap == nil {
        lhs.MonitorConfigMap = make(map[string] MonitorConfig)
    }
    for mKey, mCfg := range rhs.MonitorConfigMap {
        lhs.MonitorConfigMap[mKey] = mCfg
    }
    // MonitorInterval
    lhs.MonitorInterval = rhs.MonitorInterval

    return lhs
}

type ClientRoleMap map[string/* roleName */] ClientRole

func(roleMap ClientRoleMap) Get(r string) ClientRole {
    split := SplitWhitespace(r)
    ret   := ClientRole{}
    for _, n := range split {
        if clRole, ok := roleMap[n]; ok {
            ret = ret.Merge(clRole)
        }
    }
    return ret
}

// STATUS ---

type ClientStatus struct { // clStat
    Timestamp int64   `json:"timestamp"`
    Value     float64 `json:"value"`
    Status    int     `json:"status"`
}