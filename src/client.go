package main

import (
    "bufio"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "net"
    "os"
    "strings"
    "time"
    "regexp"
    "./monitor"
    "./secret"
)

const defaultHandshakeRetryInterval = time.Minute * 1

type Client struct {
    serverAddr string
    config ClientConfig
}

type ClientConfig struct {
    Alias string `json:"alias"`
    Comment string `json:"comment"`
    MonitorInfos map[string] MonitorInfo `json:"monitorInfos"`
    MonitorInterval int `json:"monitorInterval"`
}

func NewClient(serverAddr string) *Client {
    return &Client{
        serverAddr: serverAddr,
    }
}

func (cl *Client) handshake() error {

    // Initiate Handshake
    
    // Connection
    conn, err := net.Dial("tcp", cl.serverAddr)
    if err != nil {
        return err
    }

    Logger.Infoln("HANDSHAKE INITIATE")

    s := NewSession(conn)
    s.WriteResponse({

    })



    // Config
    rsp, err = ReadNextJsonResponse(cl, conn)
    if err != nil {
        return err
    }
    if rsp.Name != "handshake-client-config" {
        return fmt.Errorf("Bad config response")
    }
    config := rsp.Bytes("config")
    signature := rsp.Bytes("sha256Signature")

    h := sha256.New()
    h.Write(config)
    verified = secret.Verify(cl.serverAuthPublicKey, h.Sum(nil)[:], signature)
    if !verified {
        return fmt.Errorf("Config signature is invalid!")
    }

    err = json.Unmarshal(config, &cl.config)
    if err != nil {
        return err
    }

    return nil

}

func (cl *Client) checkKnownHosts() error {
    return LoadKnownHosts(flClientKnownHostsPath)
}

func (cl *Client) autoUpdate(executable []byte, signature []byte) error {

    // Verification
    h := sha256.New()
    h.Write(executable)
    execHash := h.Sum(nil)

    // Verify
    verified := secret.Verify(cl.serverAuthPublicKey, execHash, signature)
    if !verified {
        return fmt.Errorf("Authentication Failed!")
    }

    Logger.Infoln("Hash Authentication Successful!")

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
    handshakeRetryInterval := defaultHandshakeRetryInterval

    for {

        err = cl.handshake()
        if err != nil {
            Logger.Warnln("Handshake Failed:", err)
            time.Sleep(handshakeRetryInterval)
            continue
        }

        Logger.Infoln("SUCCESSFUL HANDSHAKE")
        // Config
        monitorInterval := time.Second * time.Duration(cl.config.MonitorInterval)
        handshakeRetryInterval = monitorInterval
        pass := make(chan struct{})
        go func() {
            pass <- struct{}{}
        }()

        // Loop
        MonitorLoop:
        for {

            <- pass

            // Sleep
            go func() {
                time.Sleep(monitorInterval)
                pass <- struct{}{}
            }()

            // Connection
            conn, err := net.Dial("tcp", cl.serverAddr)
            if err != nil {
                Logger.Warnln("Server Not Responding")
                continue
            }

            md := make(map[string] interface{})
            for k := range cl.config.MonitorInfos {
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
            err = WriteEncryptedJsonResponse(
                cl, conn,
                Response{
                    Name: "monitor-data",
                    Args: map[string] interface{} {
                        "version": Version,
                        "timestamp": time.Now().Unix(),
                        "monitorData": md,
                    },
                },
            )
            if err != nil {
                Logger.Warnln(err)
            }
            Logger.Infoln("Sent Data")

            //
            rsp, err := ReadNextJsonResponse(cl, conn)
            if err != nil {
                Logger.Warnln(err)
            }

            switch rsp.Name {
            case "ok":
            case "version-mismatch":
                Logger.Warnln("Version mismatch! Attempting to auto-update...")
                executable := rsp.Bytes("executable")
                signature := rsp.Bytes("sha256Signature")
        
                err = cl.autoUpdate(executable, signature)
                if err != nil {
                    Logger.Warnln(err)
                }
                break MonitorLoop
            case "session-expired":
                break MonitorLoop
            }
            
            conn.Close()

        }

    }

    return nil

}