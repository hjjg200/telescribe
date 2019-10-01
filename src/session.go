package main

import (
    "bufio"
    "bytes"
    "crypto/elliptic"
    "encoding/base64"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    "net"
    "os"
    "strings"
    "regexp"
    "sync"
    "sync/atomic"
    "time"
    . "./tc"
    "./secret"
    "./secret/p256"
    "./secret/aesgcm"
)

const packetVersionMajor = byte(0x00)
const packetVersionMinor = byte(0x05)

/*

        Client -> Server
        + Begin Handshake
        
        Server -> Client
        + Store session
        + End Handshake

        Client -> Server
        + Encrypted Response

        Server -> Client
        + Encrypted Response

*/

const sessionLifetime = time.Minute * 30
var sessionAutoIncrement int64 = 0

type SessionInfo struct {
    id []byte
    ephmPriv *p256.PrivateKey
    ephmPub *p256.PublicKey
    ephmMaster *aesgcm.Key
    thirdPub *p256.PublicKey // third party's public
    expiry time.Time
}

type Session struct {
    handshaken int32
    info *SessionInfo
    rawInput io.Reader
    input *bytes.Reader
    conn net.Conn
    inMu sync.Mutex
    outMu sync.Mutex
}

var cachedSessionInfos map[string] *SessionInfo
var sessionKnownHosts map[string] *p256.PublicKey // P256 public key
var sessionAuthPriv *p256.PrivateKey // P256 private key

func init() {
    cachedSessionInfos = make(map[string] *SessionInfo)
    go func() {
        for k, si := range cachedSessionInfos {
            if si.expiry.Sub(time.Now()) < 0 {
                delete(cachedSessionInfos, k)
            }
        }
        time.Sleep(sessionLifetime)
    }()
}

func NewSession(conn net.Conn) *Session {
    s := &Session{}
    s.SetConn(conn)
    return s
}

func (s *Session) SetConn(conn net.Conn) {
    s.rawInput = conn
    s.conn = conn
}

func NewSessionInfo() (*SessionInfo) {

    sessionAutoIncrement++
    id := new(big.Int).SetInt64(sessionAutoIncrement).Bytes()
    priv := p256.GenerateKey()

    si := &SessionInfo{
        id: id,
        ephmPriv: priv,
        ephmPub: &priv.PublicKey,
        expiry: time.Now().Add(sessionLifetime),
    }
    cachedSessionInfos[string(id)] = si

    return si

}

var knownHostsPath string
func LoadKnownHosts(fn string) error {

    // KnownHosts file structure
    // # Comment
    // <hostname> <serialized public key>
    // # Server 1
    // localhost 1bh=+vbhBg312...

    knownHostsPath = fn
    sessionKnownHosts = make(map[string] *p256.PublicKey)
    kh := fn
    st, err := os.Stat(kh)
    
    switch {
    case err != nil && !os.IsNotExist(err):
        return err
    case os.IsNotExist(err):
        f, err := os.OpenFile(kh, os.O_WRONLY | os.O_CREATE, 0600)
        if err != nil {
            return err
        }
        f.Close()
        return nil
    default:
        f, err := os.OpenFile(kh, os.O_RDONLY, 0600)
        if err != nil {
            return err
        }
        content := make([]byte, st.Size())
        _, err = f.Read(content)
        if err != nil {
            return err
        }
        f.Close()
        wsRgx := regexp.MustCompile("\\s+")
        for _, line := range strings.Split(string(content), "\n") {
            cols := wsRgx.Split(line, 2)
            if len(cols) < 2 || cols[0][0] == '#' {
                continue
            }
            host := cols[0]
            pub, err := p256.DeserializePublicKey(cols[1])
            if err != nil {
                Logger.Warnln(err)
                continue
            }
            sessionKnownHosts[host] = pub
        }
        return nil
    }

}

func LoadAuthPrivateKey(apk string) error {

    st, err := os.Stat(apk)

    switch {
    case err != nil && !os.IsNotExist(err):
        // Panic
        return err
    case os.IsNotExist(err):
        // Not exists
        Logger.Infoln("Server authentication private key does not exist.")
        Logger.Infoln("Creating a new one at", apk)
        f, err := os.OpenFile(apk, os.O_WRONLY | os.O_CREATE, 0400)
        if err != nil {
            return err
        }
        sessionAuthPriv = p256.GenerateKey()
        f.Write([]byte(p256.SerializePrivateKey(sessionAuthPriv)))
        f.Close()
        Logger.Infoln("Issued a new private key for signature authentication.")
        return nil
    default:
        // Exists
        if st.Mode() != 0400 {
            return fmt.Errorf("The server authentication private key is in a wrong permission mode. Please set it to 400.")
        }
        Logger.Infoln("Reading the server authentication private key...")
        f, err := os.OpenFile(apk, os.O_RDONLY, 0400)
        if err != nil {
            return err
        }
        buf := make([]byte, st.Size())
        _, err = f.Read(buf)
        if err != nil {
            return err
        }
        sessionAuthPriv, err = p256.DeserializePrivateKey(string(buf))
        if err != nil {
            return err
        }
        Logger.Infoln("Successfully loaded the server authentication private key.")
        return nil
    }

}

func FlushKnownHosts() error {

    kh := knownHostsPath
    f, err := os.OpenFile(kh, os.O_WRONLY | os.O_TRUNC, 0600)
    if err != nil {
        return err
    }

    for host, pub := range sessionKnownHosts {
        _, err = f.Write([]byte(
            fmt.Sprintf("%s %s", host, p256.SerializePublicKey(pub)),
       ))
        if err != nil {
            return err
        }
    }

    f.Close()
    return nil

}

const (
    packetRecordHeaderStart = "TELESCRIBE"
    packeRecordtHeaderLen = 14 // TELESCRIBE + Major + Minor + Type + \n
    packetTypeHandshakeBegin = byte(0x01)
    packetTypeHandshakeEnd = byte(0x02)
    packetTypeEncrypted = byte(0x11)
    packetTypeSessionNotFound = byte(0x21)
)

// Response/Request
//
//
//
//
//
//

func (s *Session) RemoteHost() (string, error) {
    return HostnameOf(s.conn)
}

func (s *Session) PrependRawInput(r io.Reader) {
    s.rawInput = io.MultiReader(r, s.rawInput)
}

func (s *Session) ReadEncrypted(p []byte) (i int, err error) {
    defer Catch(&err)
    encrypted, err := readNextPacket(s.rawInput)
    Try(err)
    decrypted, err := aesgcm.Decrypt(s.info.ephmMaster, encrypted)
    Try(err)
    s.input = bytes.NewReader(decrypted)
    return s.input.Read(p)
}

func (s *Session) WriteEncrypted(p []byte) (i int, err error) {
    defer Catch(&err)
    s.writeRecordHeader(packetTypeEncrypted)
    encrypted := aesgcm.Encrypt(s.info.ephmMaster, p)
    Try(writeByteSlicePacket(s.conn, s.info.id))
    Try(writeByteSlicePacket(s.conn, encrypted))
    return s.conn.Write(encrypted)
}

func (s *Session) Read(p []byte) (int, error) {
    return s.read(p, false)
}

func (s *Session) read(p []byte, nested bool) (int, error) {

    // Lock mutex unless it is nested read call
    if !nested {
        s.inMu.Lock()
        defer s.inMu.Unlock()
    }

    if s.input != nil && s.input.Len() > 0 {
        return s.input.Read(p)
    }

    //
    prh, err := s.readRecordHeader()
    if err != nil {
        return 0, err
    }

    switch prh.typ {
    case packetTypeSessionNotFound:
        // Not found
        atomic.StoreInt32(&s.handshaken, 0)
        return 0, fmt.Errorf("Need to perform another handshake")
    case packetTypeHandshakeBegin:
        s.outMu.Lock()
        err := s.EndHandshake()
        if err != nil {
            return 0, err
        }
        s.outMu.Unlock()
        // Do a nested read call
        return s.read(p, true)
    // case packetTypeHandshakeEnd:
    case packetTypeEncrypted:
        sessionId, err := readNextPacket(s.rawInput)
        if err != nil {
            return 0, err
        }
        if s.info == nil {
            // No info yet
            si, ok := cachedSessionInfos[string(sessionId)]
            if !ok {
                s.writeRecordHeader(packetTypeSessionNotFound)
                return 0, fmt.Errorf("Session not found")
            }
            s.info = si
        } else {
            if cmp := bytes.Compare(sessionId, s.info.id); cmp != 0 {
                s.writeRecordHeader(packetTypeSessionNotFound)
                return 0, fmt.Errorf("Session id mismatch")
            }
        }
        return s.ReadEncrypted(p)
    }

    return 0, fmt.Errorf("Invalid")
}

func (s *Session) Write(p []byte) (int, error) {
    s.outMu.Lock()
    defer s.outMu.Unlock()

    // TODO ensure that failed writes due to expired sessions to be written again after a handshake
    // + Option 1: Cache recent write and write it again in Session.Read
    // + Option 2:

    if !s.Handshaken() {
        s.inMu.Lock()
        Logger.Debugln("Handshake attempt")
        err := s.BeginHandshake()
        if err != nil {
            return 0, err
        }
        s.inMu.Unlock()
    }

    return s.WriteEncrypted(p)
}

func HostnameOf(conn net.Conn) (string, error) {
    host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
    if err != nil {
        return "", err
    }
    return host, err
}

func (s *Session) calculatePreMaster(pub *p256.PublicKey) ([]byte) {
    x, y := pub.X, pub.Y
    curve := elliptic.P256()
    pmX, pmY := curve.ScalarMult(x, y, s.info.ephmPriv.D.Bytes())
    pm := elliptic.Marshal(curve, pmX, pmY)
    return pm
}

func (s *Session) writeRecordHeader(typ byte) error {

    buf := bytes.NewBuffer(nil)
    buf.Write([]byte(packetRecordHeaderStart))
    buf.Write([]byte{packetVersionMajor, packetVersionMinor})
    buf.Write([]byte{typ})
    buf.Write([]byte{'\n'})
    _, err := s.conn.Write(buf.Bytes())
    return err

}

func (s *Session) BeginHandshake() (err error) {

    defer Catch(&err)

    if sessionKnownHosts == nil {
        return fmt.Errorf("Client must have known host list.")
    }

    host, err := HostnameOf(s.conn)
    Try(err)
    authPub, haveAuthPub := sessionKnownHosts[host]

    Try(s.writeRecordHeader(packetTypeHandshakeBegin))

    forDigest := bytes.NewBuffer(nil)

    // Block from client
    // | length(varint) | client random length | client random |
    // | public key length | public key |
    // | challenge length | challenge |
    si := NewSessionInfo()
    si.id = nil
    s.info = si
    clRnd := secret.RandomBytes(32)
    challenge := secret.RandomBytes(32)
    buf := bytes.NewBuffer(nil)
    mw := io.MultiWriter(buf, s.conn)
    Logger.Debugln("Attempting to write to server")
    err = writeByteSeriesPacket(mw, [][]byte{
        clRnd, s.info.ephmPub.Bytes(), challenge,
    })
    Try(err)
    clHsMsg, err := readNextPacket(bytes.NewReader(buf.Bytes()))
    Try(err)
    forDigest.Write(clHsMsg)
    
    Logger.Debugln("Sent all handshake packet")
    
    // Block from server
    // | length(varint) | server random length | server random |
    // | public key length | public key |
    // | signature length | public key signature signed with auth priv |
    // | challenge signature length | challenge signature |
    // | server auth pub length | server auth pub |
    // | session id length | session id |

    //
    prh, err := s.readRecordHeader()
    Try(err)
    Logger.Debugln(prh)

    //
    if prh.typ != packetTypeHandshakeEnd {
        Try(fmt.Errorf("Bad handshake record header"))
    }

    bx, err := readNextPacket(s.rawInput)
    Try(err)
    srvHs := bytes.NewReader(bx)
    srvRnd, err := readNextPacket(srvHs)
    Try(err)
    srvPubBytes, err := readNextPacket(srvHs)
    Try(err)
    srvPubSig, err := readNextPacket(srvHs)
    Try(err)
    srvChallenge, err := readNextPacket(srvHs)
    Try(err)
    srvAuthPubBytes, err := readNextPacket(srvHs)
    Try(err)

    if !haveAuthPub {
        // TODO scan for y

        authPub, ok := new(p256.PublicKey).SetBytes(srvAuthPubBytes)
        Assert(ok, "Bad public key bytes")
        fp := authPub.Fingerprint()
        fmt.Println("The server you are trying to connect has an unknown public key fingerprint:")
        fmt.Println(fp, "\n")
        fmt.Println("Accept the server's authentication public key? (y/N): ")

        stdRd := bufio.NewReader(os.Stdin)
        y, err := stdRd.ReadString('\n')
        Try(err)
        y = y[:1]
        if y == "y" || y == "Y" {
            sessionKnownHosts[host] = authPub
            Try(FlushKnownHosts())
        } else {
            Try(fmt.Errorf("Did not accept the server request."))
        }
    } else {
        // Compare Public Key
        cmp := bytes.Compare(authPub.Bytes(), srvAuthPubBytes)
        givenAuthPub, ok := new(p256.PublicKey).SetBytes(srvAuthPubBytes)
        Assert(ok, "Bad public key bytes")
        if cmp != 0 {
            Logger.Warnln("The host's public key fingerprint does not match!\n" +
                "Terminating the connection!\n\n" +
                "Have:", authPub.Fingerprint() +
                "Given:", givenAuthPub.Fingerprint(),
            )
            Try(fmt.Errorf("Auth public key does not match"))
        }
    }

    sessionId, err := readNextPacket(srvHs)
    Try(err)
    s.info.id = sessionId

    // Verify server pub
    verified := p256.Verify(authPub, srvPubBytes, srvPubSig)
    challengeResult := p256.Verify(authPub, challenge, srvChallenge)

    if !(verified && challengeResult) {
        return fmt.Errorf("Invalid signature")
    }

    forDigest.Write(bx)
    
    // Calc master secret
    // SHA256(PreMasterSecret || SHA256(digest) || clRnd || srvRnd)

    buf = bytes.NewBuffer(nil)
    srvPub, ok := new(p256.PublicKey).SetBytes(srvPubBytes)
    Assert(ok, "Bad public key bytes")
    preMaster := s.calculatePreMaster(srvPub)
    digest := Sha256Sum(forDigest.Bytes())
    buf.Write(preMaster)
    buf.Write(digest)
    buf.Write(clRnd)
    buf.Write(srvRnd)

    master := aesgcm.NewKey(Sha256Sum(buf.Bytes()))
    s.info.ephmMaster = master
    s.info.thirdPub = srvPub
    atomic.StoreInt32(&s.handshaken, 1)

    return nil

}

func (s *Session) EndHandshake() (err error) {

    defer Catch(&err)
    //
    if sessionAuthPriv == nil {
        return fmt.Errorf("Server must have an auth private key.")
    }

    Logger.Debugln("Accepted handshake begin")

    //
    forDigest := bytes.NewBuffer(nil)

    bx, err := readNextPacket(s.rawInput)
    bxRd := bytes.NewReader(bx)
    Try(err)
    clRnd, err := readNextPacket(bxRd)
    Try(err)
    clPubBytes, err := readNextPacket(bxRd)
    Try(err)
    clPub, ok := new(p256.PublicKey).SetBytes(clPubBytes)
    Assert(ok, "Bad public key bytes")
    clChallenge, err := readNextPacket(bxRd)
    Try(err)

    forDigest.Write(bx)

    // Response
    Try(s.writeRecordHeader(packetTypeHandshakeEnd))

    si := NewSessionInfo()
    s.info = si
    srvRnd := secret.RandomBytes(32)
    srvPubSig := p256.Sign(sessionAuthPriv, si.ephmPub.Bytes())
    challengeSig := p256.Sign(sessionAuthPriv, clChallenge)

    buf := bytes.NewBuffer(nil)
    mw := io.MultiWriter(buf, s.conn)
    err = writeByteSeriesPacket(mw, [][]byte{
        srvRnd, si.ephmPub.Bytes(), srvPubSig,
        challengeSig, sessionAuthPriv.PublicKey.Bytes(), si.id,
    })
    Try(err)

    bx, err = readNextPacket(bytes.NewReader(buf.Bytes()))
    Try(err)
    forDigest.Write(bx)

    // Digest
    // Entire client handshake message || entire server handshake message
    // Calc master secret
    // SHA256(PreMasterSecret || SHA256(digest) || clRnd || srvRnd)

    buf = bytes.NewBuffer(nil)
    preMaster := s.calculatePreMaster(clPub)
    digest := Sha256Sum(forDigest.Bytes())
    buf.Write(preMaster)
    buf.Write(digest)
    buf.Write(clRnd)
    buf.Write(srvRnd)

    master := aesgcm.NewKey(Sha256Sum(buf.Bytes()))
    si.ephmMaster = master
    si.thirdPub = clPub
    atomic.StoreInt32(&s.handshaken, 1)

    return nil

}

type byteReader struct {
    r io.Reader
}

func (br byteReader) ReadByte() (byte, error) {
    p := make([]byte, 1)
    _, err := br.r.Read(p)
    if err != nil {
        return byte(0), err
    }
    return p[0], nil
}

func writeVarint(w io.Writer, i int64) error {
    p := make([]byte, binary.MaxVarintLen64)
    n := binary.PutVarint(p, i)
    _, err := w.Write(p[:n])
    return err
}

func readVarint(r io.Reader) (int64, error) {
    return binary.ReadVarint(byteReader{ r })
}

func readNextPacket(r io.Reader) ([]byte, error) {
    n, err := readVarint(r)
    if err != nil {
        return nil, err
    }
    p := make([]byte, n)
    n2, err := r.Read(p)
    if err != nil || n != int64(n2) {
        return nil, fmt.Errorf("Bad packet")
    }
    return p, nil
}

func writeByteSlicePacket(w io.Writer, p []byte) (err error) {
    defer Catch(&err)
    Try(writeVarint(w, int64(len(p))))
    _, err = w.Write(p)
    return
}

func writeByteSeriesPacket(w io.Writer, bx [][]byte) (err error) {
    defer Catch(&err)

    buf := bytes.NewBuffer(nil)
    for _, p := range bx {
        Try(writeByteSlicePacket(buf, p))
    }
    all := buf.Bytes()
    buf = bytes.NewBuffer(nil)
    writeVarint(buf, int64(len(all)))
    buf.Write(all)

    _, err = w.Write(buf.Bytes())
    return err
}

type PacketRecordHeader struct {
    vMinor, vMajor, typ byte
}

func (s *Session) readRecordHeader() (PacketRecordHeader, error) {
    prhl := packeRecordtHeaderLen
    p := make([]byte, prhl)
    Logger.Debugln("Attenpting to read header")
    n, err := s.rawInput.Read(p)
    if err != nil || n != prhl {
        return PacketRecordHeader{}, fmt.Errorf("Bad record header")
    }

    // Check
    if string(p[:10]) != packetRecordHeaderStart ||
        p[10] != packetVersionMajor ||
        p[11] != packetVersionMinor {
        return PacketRecordHeader{}, fmt.Errorf("Bad record header")
    }

    return PacketRecordHeader{
        vMajor: p[10], vMinor: p[11],  typ: p[12],
    }, nil
}

func (s *Session) WriteResponse(rp Response) error {
    return WriteResponse(s, rp)
}

func (s *Session) NextResponse() (Response, error) {
    return ReadResponse(s)
}

func (s *Session) Handshaken() bool {
    i := atomic.LoadInt32(&s.handshaken)
    return i == 1
}

//
// SESSION ERROR HANDLING
//

var (
    SessionErrExpired = fmt.Errorf("Session is expired.")
)

func IsSessionExpired(err error) bool {
    return err == SessionErrExpired
}

//
// RESPONSE
//

type Response struct {
    name string
    args map[string] interface{}
}

func WriteResponse(w io.Writer, rp Response) error {

    j, err := json.Marshal(rp.Args())
    if err != nil {
        return err
    }
    return writeByteSeriesPacket(w, [][]byte{
        []byte(rp.Name()), j,
    })
    
}

func ReadResponse(r io.Reader) (Response, error) {

    bx, err := readNextPacket(r)
    Logger.Debugln(string(bx))
    if err != nil {
        return Response{}, err
    }
    bxRd := bytes.NewReader(bx)
    nameBytes, err := readNextPacket(bxRd)
    if err != nil {
        return Response{}, err
    }
    name := string(nameBytes)
    body, err := readNextPacket(bxRd)
    if err != nil {
        return Response{}, err
    }
    args := make(map[string] interface{})
    err = json.Unmarshal(body, &args)
    if err != nil {
        return Response{}, err
    }

    return Response{
        name: name,
        args: args,
    }, nil

}

func NewResponse(name string) Response {
    return Response{
        name: name,
        args: make(map[string] interface{}),
    }
}

func (rp *Response) Name() string {
    return rp.name
}

func (rp *Response) Get(key string) interface{} {
    v, _ := rp.args[key]
    return v
}

func (rp *Response) Args() map[string] interface{} {
    return rp.args
}

func (rp *Response) Set(key string, val interface{}) {
    rp.args[key] = val
}

func (rp *Response) SetArgs(args map[string] interface{}) {
    rp.args = args
}

func (rp *Response) String(key string) (str string) {
    str, _ = rp.args[key].(string)
    return
}

func (rp *Response) Int(key string) (i int) {
    i, _ = rp.args[key].(int)
    return
}

func (rp *Response) Float64(key string) (f float64) {
    f, _ = rp.args[key].(float64)
    return
}

func (rp *Response) Bytes(key string) []byte {
    // Json uses base64 to encode []byte
    b, err := base64.StdEncoding.DecodeString(rp.String(key))
    if err != nil {
        return nil
    }
    return b
}