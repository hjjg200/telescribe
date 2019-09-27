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
    //config clientConfig
    serverAddr string
    serverPulicKey *rsa.PublicKey
    serverAuthPublicKey *rsa.PublicKey
    clientPrivateKey *rsa.PrivateKey
    knownHosts map[string] string
    config ClientConfig
}

type ClientConfig struct {
    Alias string `json:"alias"`
    Comment string `json:"comment"`
    MonitorInfos map[string] MonitorInfo `json:"monitorInfos"`
    MonitorInterval int `json:"monitorInterval"`
}

func NewClient( serverAddr string ) *Client {
    return &Client{
        serverAddr: serverAddr,
    }
}

func ( cl *Client ) MyPrivateKey() *rsa.PrivateKey {
    return cl.clientPrivateKey
}

func ( cl *Client ) TheirPublicKey() *rsa.PublicKey {
    return cl.serverPulicKey
}

func ( cl *Client ) SetTheirPublicKey( rsaPub *rsa.PublicKey ) {
    cl.serverPulicKey = rsaPub
}

func ( cl *Client ) handshake() error {

    // Initiate Handshake
    
    // Connection
    conn, err := net.Dial( "tcp", cl.serverAddr )
    if err != nil {
        return err
    }

    Logger.Infoln( "HANDSHAKE INITIATE" )

    err = WriteJsonResponse(
        cl, conn,
        Response{
            Name: "handshake-initiate",
        },
    )
    if err != nil {
        return err
    }

    // Server Response 1
    rsp, err := ReadNextJsonResponse( cl, conn )
    if err != nil {
        return err
    }
    Logger.Infoln( "SERVER RESPONSE 1:", rsp.Name )
    switch rsp.Name {
    case "not-whitelisted":
        return fmt.Errorf( "Client is not whitelisted" )
    case "handshake-server-publickey":

        // Public Keys
        serializedPublicKey := rsp.String( "publicKey" )
        serializedAuthPublicKey := rsp.String( "authPublicKey" )

        rsaPub, err := secret.DeserializePublicKey( serializedPublicKey )
        if err != nil {
            return fmt.Errorf( "Bad public key from the server" )
        }
        rsaAuthPub, err := secret.DeserializePublicKey( serializedAuthPublicKey )
        if err != nil {
            return fmt.Errorf( "Bad auth public key from the server" )
        }

        cl.serverPulicKey = rsaPub
        cl.serverAuthPublicKey = rsaAuthPub

        // Check Known Hosts
        fp := secret.FingerprintPublicKey( rsaAuthPub )
        host := flClientHostname
        haveFp, ok := cl.knownHosts[host]

        if ok && haveFp != fp {
            return fmt.Errorf( "The fingerprint for %s does not match!\nHave: %s\nGiven: %s", host, haveFp, fp )
        } else if !ok {
            fmt.Println( "The server you are trying to connect has an unknown public key fingerprint:" )
            fmt.Println( fp, "\n" )
            fmt.Print( "Accept the server's authentication public key? (y/N): " )
            rd := bufio.NewReader( os.Stdin )
            y, err := rd.ReadString( '\n' )
            if err != nil {
                return err
            }
            y = y[:1]
            if y == "y" || y == "Y" {
                cl.knownHosts[host] = fp
                err = cl.updateKnownHosts()
                if err != nil {
                    return err
                }
            } else {
                return fmt.Errorf( "Did not accept the server request." )
            }
        }

    }

    // Version Challenge
    Logger.Infoln( "VERSION CHALLENGE" )
    err = WriteJsonResponse(
        cl, conn,
        Response{
            Name: "handshake-version-challenge",
            Args: map[string] interface{} {
                "version": Version,
            },
        },
    )
    if err != nil {
        return err
    }

    // Version Challenge Response
    Logger.Infoln( "VERSION CHALLENGE RESPONSE" )
    rsp, err = ReadNextJsonResponse( cl, conn )
    if err != nil {
        return err
    }
    if rsp.Name != "handshake-version-challenge-response" {
        return fmt.Errorf( "Wrong version challenge reponse" )
    }
    msg := rsp.String( "result" )
    if msg == "mismatch" {
        // Auto update
        Logger.Warnln( "Version mismatch! Attempting to auto-update..." )
        executable := rsp.Bytes( "executable" )
        // Signature
        signature := rsp.Bytes( "sha256Signature" )

        return cl.autoUpdate( executable, signature )
    }

    // Authentication Challenge
    chLen := 32
    challengeMsg := make( []byte, chLen )
    rand.Read( challengeMsg )
    Logger.Infoln( "AUTHENTICATION CHALLENGE" )
    Logger.Infoln( "Message:", challengeMsg )
    err = WriteJsonResponse(
        cl, conn,
        Response{
            Name: "handshake-authentication-challenge",
            Args: map[string] interface{} {
                "message": challengeMsg,
            },
        },
    )

    // Challenge Response
    Logger.Infoln( "CHALLENGE RESPONSE" )
    rsp, err = ReadNextJsonResponse( cl, conn )
    if err != nil {
        return err
    }
    if rsp.Name != "handshake-challenge-response" {
        return fmt.Errorf( "Challenge failed" )
    }
    singature := rsp.Bytes( "signature" )
    verified := secret.Verify( cl.serverAuthPublicKey, challengeMsg, singature )
    if !verified {
        return fmt.Errorf( "Challenge failed: wrong signature" )
    }
    Logger.Infoln( "CHALLENGE AUTHENTICATION SUCCESSFUL" )

    // Handshake Client Public Key
    Logger.Infoln( "CLIENT PUBLIC KEY" )
    cl.clientPrivateKey = secret.RandomPrivateKey()
    err = WriteJsonResponse(
        cl, conn,
        Response{
            Name: "handshake-client-publickey",
            Args: map[string] interface{} {
                "publicKey": secret.SerializePublicKey( &cl.clientPrivateKey.PublicKey ),
            },
        },
    )
    if err != nil {
        return err
    }

    // Server Response 3
    Logger.Infoln( "SERVER RESPONSE 3" )
    rsp, err = ReadNextJsonResponse( cl, conn )
    if err != nil {
        return err
    }
    if rsp.Name != "handshake-ping" {
        return fmt.Errorf( "Wrong ping from the server" )
    }

    // Handshake Pong
    Logger.Infoln( "HANDSHAKE PONG" )
    err = WriteEncryptedJsonResponse(
        cl, conn,
        Response{
            Name: "handshake-pong",
        },
    )
    if err != nil {
        return err
    }

    // Config
    rsp, err = ReadNextJsonResponse( cl, conn )
    if err != nil {
        return err
    }
    if rsp.Name != "handshake-client-config" {
        return fmt.Errorf( "Bad config response" )
    }
    config := rsp.Bytes( "config" )
    signature := rsp.Bytes( "sha256Signature" )

    h := sha256.New()
    h.Write( config )
    verified = secret.Verify( cl.serverAuthPublicKey, h.Sum( nil )[:], signature )
    if !verified {
        return fmt.Errorf( "Config signature is invalid!" )
    }

    err = json.Unmarshal( config, &cl.config )
    if err != nil {
        return err
    }

    return nil

}

func ( cl *Client ) checkKnownHosts() error {

    // KnownHosts file structure
    // # Comment
    // <hostname> <public key fingerprint>
    // # Server 1
    // localhost 2048 SHA256:19......

    cl.knownHosts = make( map[string] string )
    kh := flClientKnownHostsPath
    st, err := os.Stat( kh )
    
    switch {
    case err != nil && !os.IsNotExist( err ):
        return err
    case os.IsNotExist( err ):
        f, err := os.OpenFile( kh, os.O_WRONLY | os.O_CREATE, 0600 )
        if err != nil {
            return err
        }
        f.Close()
        return nil
    default:
        f, err := os.OpenFile( kh, os.O_RDONLY, 0600 )
        if err != nil {
            return err
        }
        content := make( []byte, st.Size() )
        _, err = f.Read( content )
        if err != nil {
            return err
        }
        f.Close()
        wsRgx := regexp.MustCompile( "\\s+" )
        for _, line := range strings.Split( string( content ), "\n" ) {
            cols := wsRgx.Split( line, 2 )
            if len( cols ) < 2 || cols[0][0] == '#' {
                continue
            }
            host, fp := cols[0], cols[1]
            cl.knownHosts[host] = fp
        }
        return nil
    }

}

func ( cl *Client ) updateKnownHosts() error {

    kh := flClientKnownHostsPath
    f, err := os.OpenFile( kh, os.O_WRONLY | os.O_TRUNC, 0600 )
    if err != nil {
        return err
    }

    for host, fp := range cl.knownHosts {
        _, err = f.Write( []byte(
            fmt.Sprintf( "%s %s", host, fp ),
        ) )
        if err != nil {
            return err
        }
    }

    f.Close()
    return nil

}

func ( cl *Client ) autoUpdate( executable []byte, signature []byte ) error {

    // Verification
    h := sha256.New()
    h.Write( executable )
    execHash := h.Sum( nil )

    // Verify
    verified := secret.Verify( cl.serverAuthPublicKey, execHash, signature )
    if !verified {
        return fmt.Errorf( "Authentication Failed!" )
    }

    Logger.Infoln( "Hash Authentication Successful!" )

    Logger.Infoln( "Started Auto Update Procedure." )
    Logger.Infoln( "The service must be set to automatically restart." )

    // Remove
    err := os.Remove( executablePath )
    if err != nil {
        return err
    }

    // Write New Executable
    f, err := os.OpenFile( executablePath, os.O_CREATE | os.O_WRONLY, 0755 )
    if err != nil {
        return err
    }
    _, err = f.Write( executable )
    if err != nil {
        return err
    }

    f.Close()

    Logger.Infoln( "Successfully updated the executable." )
    Logger.Infoln( "Exiting the application..." )
    os.Exit( 1 )
    return nil

}

func ( cl *Client ) Start() error {

    err := cl.checkKnownHosts()
    if err != nil {
        return err
    }

    //
    handshakeRetryInterval := defaultHandshakeRetryInterval

    for {

        err = cl.handshake()
        if err != nil {
            Logger.Warnln( "Handshake Failed:", err )
            time.Sleep( handshakeRetryInterval )
            continue
        }

        Logger.Infoln( "SUCCESSFUL HANDSHAKE" )
        // Config
        monitorInterval := time.Second * time.Duration( cl.config.MonitorInterval )
        handshakeRetryInterval = monitorInterval
        pass := make( chan struct{} )
        go func() {
            pass <- struct{}{}
        }()

        // Loop
        MonitorLoop:
        for {

            <- pass

            // Sleep
            go func() {
                time.Sleep( monitorInterval )
                pass <- struct{}{}
            }()

            // Connection
            conn, err := net.Dial( "tcp", cl.serverAddr )
            if err != nil {
                Logger.Warnln( "Server Not Responding" )
                continue
            }

            md := make( map[string] interface{} )
            for k := range cl.config.MonitorInfos {
                getter, ok := monitor.Getter( k )
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
                Logger.Warnln( err )
            }
            Logger.Infoln( "Sent Data" )

            //
            rsp, err := ReadNextJsonResponse( cl, conn )
            if err != nil {
                Logger.Warnln( err )
            }

            switch rsp.Name {
            case "ok":
            case "version-mismatch":
                Logger.Warnln( "Version mismatch! Attempting to auto-update..." )
                executable := rsp.Bytes( "executable" )
                signature := rsp.Bytes( "sha256Signature" )
        
                err = cl.autoUpdate( executable, signature )
                if err != nil {
                    Logger.Warnln( err )
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