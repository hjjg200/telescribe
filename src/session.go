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
    "sync"
    "sync/atomic"
    "time"

    "./secret"
    "./secret/p256"
    "./secret/aesgcm"

    . "github.com/hjjg200/go-act"
)

const packetVersionMajor = 0
const packetVersionMinor = 6

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
const clientWaitForInput = time.Second * 30
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
    isServer bool
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
        for {
            buf := make(map[string] *SessionInfo)
            // Concurrent delete
            // Copy
            for k, si := range cachedSessionInfos { buf[k] = si }
            // Delete
            for k, si := range buf {
                if si.IsExpired() {
                    delete(buf, k)
                }
            }
            // Assign
            cachedSessionInfos = buf
            time.Sleep(sessionLifetime)
        }
    }()
}

func NewSession(conn net.Conn) *Session {
    s := &Session{}
    s.SetConn(conn)
    return s
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

func (si *SessionInfo) IsExpired() bool {
    return si.expiry.Sub(time.Now()) < 0
}

var knownHostsPath string
func LoadKnownHosts(kh string) error {

    // KnownHosts file structure
    // # Comment
    // <hostname> <serialized public key>
    // # Server 1
    // localhost 1bh=+vbhBg312...

    knownHostsPath = kh
    sessionKnownHosts = make(map[string] *p256.PublicKey)
    _, err := os.Stat(kh)
    
    switch {
    case err != nil && !os.IsNotExist(err):
        return err
    case os.IsNotExist(err):
        return TouchFile(kh, 0600)
    default:
        content, err := ReadFile(kh, 0600)
        if err != nil {
            return err
        }
        for _, line := range SplitLines(string(content)) {
            cols := SplitWhitespaceN(line, 2)
            if len(cols) < 2 || cols[0][0] == '#' {
                continue
            }
            host := cols[0]
            pub, err := p256.DeserializePublicKey(cols[1])
            if err != nil {
                continue
            }
            EventLogger.Infoln("Loaded the public key of", host)
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
        EventLogger.Infoln("Server authentication private key does not exist.")
        EventLogger.Infoln("Creating a new one at", apk)
        f, err := os.OpenFile(apk, os.O_WRONLY | os.O_CREATE, 0400)
        if err != nil {
            return err
        }
        sessionAuthPriv = p256.GenerateKey()
        f.Write([]byte(p256.SerializePrivateKey(sessionAuthPriv)))
        f.Close()
        EventLogger.Infoln("Issued a new private key for signature authentication.")
        return nil
    default:
        // Exists
        if st.Mode() != 0400 {
            return fmt.Errorf("The server authentication private key is in a wrong permission mode. Please set it to 400.")
        }
        EventLogger.Infoln("Reading the server authentication private key...")
        serialized, err := ReadFile(apk, 0400)
        if err != nil {
            return err
        }
        sessionAuthPriv, err = p256.DeserializePrivateKey(string(serialized))
        if err != nil {
            return err
        }
        EventLogger.Infoln("Successfully loaded the server authentication private key.")
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
        f.Write([]byte(fmt.Sprintf("%s %s", host, p256.SerializePublicKey(pub))))
        if err != nil {
            return err
        }
    }

    f.Close()
    return nil

}

//
// SESSION
//

const (
    packetRecordHeaderStart = "TELESCRIBE"
    packeRecordtHeaderLen = 14 // TELESCRIBE + Major + Minor + Type + \n

    packetTypeHandshakeBegin = byte(0x11)  // 0x10
    packetTypeHandshakeEnd = byte(0x12)
    packetTypeEncrypted = byte(0x21)       // 0x20
    packetTypeSessionNotFound = byte(0x31) // 0x30
)

func (s *Session) Close() error {
    return s.conn.Close()
}

func (s *Session) SetConn(conn net.Conn) {
    s.rawInput = conn
    s.conn = conn
}

func (s *Session) IsExpired() bool {
    if s.info == nil {
        return true
    }
    return s.info.IsExpired()
}

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

    //
    if s.input != nil && s.input.Len() > 0 {
        return s.input.Read(p)
    }

    //
    prh, err := s.readRecordHeader()
    if err != nil {
        return 0, err
    }

    //
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
    case packetTypeEncrypted:
        sessionId, err := readNextPacket(s.rawInput)
        if err != nil {
            return 0, err
        }
        if s.info == nil {
            // No info yet, look for cached session
            si, ok := cachedSessionInfos[string(sessionId)]
            if !ok {
                s.writeRecordHeader(packetTypeSessionNotFound)
                return 0, fmt.Errorf("Session not found")
            }
            s.info = si
            atomic.StoreInt32(&s.handshaken, 1)
        }
        if cmp := bytes.Compare(sessionId, s.info.id); cmp != 0 {
            s.writeRecordHeader(packetTypeSessionNotFound)
            return 0, fmt.Errorf("Session id mismatch")
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
    //             + it doesn't get written until the next read
    // + Option 2: In s.Read close a connection whose session is expired so that Write fails, and Write
    //             makes subsequent attmepts to ensure the data to be written.
    //             + readRecordHeader may not work when the other side is not responding
    //             + conn gets closed so it is not usable again
    // + Option 3: Read must write a header back so that write can read the header
    //             Write -> Header+Data -> Read -> Header -> Write
    //             ? What if read is ongoing while writing?
    //             -> Header can be corrupted
    // + Chosen method for now: Expire sessions in the client side too so that client would know if a session is expired

    switch {
    case !s.Handshaken(),             // Handshake not done
        !s.isServer && s.IsExpired(): // Client and its session is expired

        s.inMu.Lock()
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
    buf.Write([]byte{byte(packetVersionMajor), byte(packetVersionMinor)})
    buf.Write([]byte{typ})
    buf.Write([]byte{'\n'})
    _, err := s.conn.Write(buf.Bytes())
    return err
}

func (s *Session) BeginHandshake() (err error) {

    defer Catch(&err)

    // Only clients begin handshake
    s.isServer = false

    if sessionKnownHosts == nil {
        return fmt.Errorf("Client must have known host list.")
    }

    host, err := HostnameOf(s.conn)
    Try(err)
    authPub, haveAuthPub := sessionKnownHosts[host]

    Try(s.writeRecordHeader(packetTypeHandshakeBegin))

    digest := bytes.NewBuffer(nil)

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
    err = writeByteSeriesPacket(mw, [][]byte{
        clRnd, s.info.ephmPub.Bytes(), challenge,
    })
    Try(err)
    clHsMsg, err := readNextPacket(bytes.NewReader(buf.Bytes()))
    Try(err)
    digest.Write(clHsMsg)
    
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
        var ok bool
        authPub, ok = new(p256.PublicKey).SetBytes(srvAuthPubBytes)
        Assert(ok, "Bad public key bytes")
        fp := authPub.Fingerprint()
        EventLogger.Warnln(
            "The server you are trying to connect has an unknown public key fingerprint:\n" +
            fp +
            "\n\n" +
            "Accept the server's authentication public key? (y/N): ",
        )

        done := false
        go func() {
            time.Sleep(clientWaitForInput)
            Assert(done, "No response from user")
        }()
        stdRd := bufio.NewReader(os.Stdin)
        y, err := stdRd.ReadString('\n')
        Try(err)
        done = true
        switch y[:1] {
        case "y", "Y":
            sessionKnownHosts[host] = authPub
            Try(FlushKnownHosts())
        default:
            Try(fmt.Errorf("Did not accept the server request."))
        }

    } else {
        // Compare Public Key
        cmp := bytes.Compare(authPub.Bytes(), srvAuthPubBytes)
        givenAuthPub, ok := new(p256.PublicKey).SetBytes(srvAuthPubBytes)
        Assert(ok, "Bad public key bytes")
        if cmp != 0 {
            EventLogger.Warnln(
                "The host's public key fingerprint does not match!\n" +
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

    digest.Write(bx)
    
    // Calc master secret
    // SHA256(PreMasterSecret || SHA256(digest) || clRnd || srvRnd)

    srvPub, ok := new(p256.PublicKey).SetBytes(srvPubBytes)
    Assert(ok, "Bad public key bytes")
    preMaster := s.calculatePreMaster(srvPub)
    digestHash := Sha256Sum(digest.Bytes())
    master := aesgcm.NewKey(Sha256Sum(
        preMaster, digestHash, clRnd, srvRnd,
    ))
    s.info.ephmMaster = master
    s.info.thirdPub = srvPub
    atomic.StoreInt32(&s.handshaken, 1)

    return nil

}

func (s *Session) EndHandshake() (err error) {

    defer Catch(&err)
    
    // Only servers end handshake
    s.isServer = true

    if sessionAuthPriv == nil {
        return fmt.Errorf("Server must have an auth private key.")
    }

    //
    digest := bytes.NewBuffer(nil)

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

    digest.Write(bx)

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
    digest.Write(bx)

    // Digest
    // Entire client handshake message || entire server handshake message
    // Calc master secret
    // SHA256(PreMasterSecret || SHA256(digest) || clRnd || srvRnd)

    preMaster := s.calculatePreMaster(clPub)
    digestHash := Sha256Sum(digest.Bytes())
    master := aesgcm.NewKey(Sha256Sum(
        preMaster, digestHash, clRnd, srvRnd,
    ))
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
        return nil, fmt.Errorf(fmt.Sprint("Bad varint: ", err))
    }

    buf         := bytes.NewBuffer(nil)
    copied, err := io.CopyN(buf, r, n)

    if err != nil || n != copied {
        return nil, fmt.Errorf(fmt.Sprint("Bad packet: ", err))
    }
    return buf.Bytes(), nil
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
    n, err := s.rawInput.Read(p)
    if err != nil {
        return PacketRecordHeader{}, err
    }
    if n != prhl {
        return PacketRecordHeader{}, fmt.Errorf("Record header too short")
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

func (rp *Response) Int(key string) int {
    v, _ := rp.args[key].(float64)
    return int(v)
}

func (rp *Response) Int64(key string) int64 {
    // Json numbers are all float64
    v, _ := rp.args[key].(float64)
    return int64(v)
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