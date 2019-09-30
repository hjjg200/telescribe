package main

import (
    "bufio"
    "bytes"
    "crypto/rsa"
    "encoding/base64"
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "strconv"
    "strings"
    "regexp"
    . "./tc"
)

const packetVersionMajor = byte( 0x00 )
const packetVersionMinor = byte( 0x05 )

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
    ephmPriv []byte // elliptic P256, ephemeral
    ephmPub []byte // elliptic P256, ephemeral
    ephmMaster []byte
    thirdPub []byte // third party's public
    expiry time.Time
}

type Session struct {
    info *SessionInfo
    rawInput io.Reader
    input *bufio.Reader
    conn net.Conn
    inMu sync.Mutex
    outMu sync.Mutex
}

var cachedSessionInfos map[string] *SessionInfo
var sessionKnownHosts map[string] []byte // marshaled P256 public key
var sessionAuthPriv []byte // P256 private key

func init() {
    cachedSessionInfos = make( map[string] *SessionInfo )
    go func() {
        for k, si := range cachedSessionInfos {
            if si.expiry.Sub( time.Now() ) < 0 {
                delete( cachedSessionInfos, k )
            }
        }
        time.Sleep( sessionLifetime )
    }()
}

func NewSession( conn net.Conn ) *Session {

    s := &Session{
        id: nil,
        rawInput: conn,
        conn: conn,
    }

    return s

}

func NewSessionInfo() ( *SessionInfo ) {

    sessionAutoIncrement++
    id := new( big.Int ).SetInt64( sessionAutoIncrement ).Bytes()
    p256 := elliptic.P256()
    priv, pubx, puby, _ := elliptic.GenerateKey( p256, rand.Reader )
    pub := elliptic.Marshal( p256, pubx, puby )

    si := &SessionInfo{
        id, id,
        ephmPriv: priv,
        ephmPub: pub,
        expiry: time.Now().Add( sessionLifetime ),
    }
    cachedSessionInfos[string( id )] = si

    return si

}

func LoadKnownHosts( fn string ) error {

    f, err := os.OpenFile( fn, os.O_RDONLY, 0400 )
    if err != nil {
        return err
    }
    buf := bytes.NewBuffer( nil )
    io.Copy( buf, f )
    f.Close()

    //
    lines := strings.Split( buf.String(), "\n" )
    for i := range lines {
        line := lines[i]
        if line[0] == '#' {
            // Comments
            continue
        }
        cols := strings.SplitN( line, 2 )
        if len( cols ) == 1 {
            // Malformed
            return fmt.Errorf( "Known hosts file error at line %d", i + 1 )
        }

        // Deserialize public keys
        sessionKnownHosts[cols[0]] = secret.DeserializeP256PublicKey( cols[1] )
    }
    return nil

}

func LoadAuthPrivateKey( apk string ) error {

    st, err := os.Stat( apk )

    switch {
    case err != nil && !os.IsNotExist( err ):
        // Panic
        return err
    case os.IsNotExist( err ):
        // Not exists
        Logger.Infoln( "Server authentication private key does not exist." )
        Logger.Infoln( "Creating a new one at", apk )
        f, err := os.OpenFile( apk, os.O_WRONLY | os.O_CREATE, 0400 )
        if err != nil {
            return err
        }
        sessionAuthPriv = secret.RandomPrivateKey()
        f.Write( []byte( secret.SerializePrivateKey( sessionAuthPriv ) ) )
        f.Close()
        Logger.Infoln( "Issued a new private key for signature authentication." )
        return nil
    default:
        // Exists
        if st.Mode() != 0400 {
            return fmt.Errorf( "The server authentication private key is in a wrong permission mode. Please set it to 400." )
        }
        Logger.Infoln( "Reading the server authentication private key..." )
        f, err := os.OpenFile( apk, os.O_RDONLY, 0400 )
        if err != nil {
            return err
        }
        buf := make( []byte, st.Size() )
        _, err = f.Read( buf )
        if err != nil {
            return err
        }
        sessionAuthPriv, err = secret.DeserializePrivateKey( string( buf ) )
        if err != nil {
            return err
        }
        Logger.Infoln( "Successfully loaded the server authentication private key." )
        return nil
    }

}

const (
    packetRecordHeaderStart = "TELESCRIBE"
    packeRecordtHeaderLen = 14 // TELESCRIBE + Major + Minor + Type + \n
    packetTypeHandshakeBegin = byte( 0x01 )
    packetTypeHandshakeEnd = byte( 0x02 )
    packetTypeEncrypted = byte( 0x11 )
    packetTypeSessionNotFound = byte( 0x21 )
)

// Response/Request
//
//
//
//
//
//

func ( s *Session ) RemoteHost() ( string, error ) {
    return HostnameOf( s.conn.RemoteAddr() )
}

func ( s *Session ) PrependRawInput( r io.Reader ) {
    s.rawInput = io.MultiReader( r, s.rawInput )
}

func ( s *Session ) ReadEncrypted( p []byte ) ( int, error ) {
    buf := make( []byte, len( p ) )
    n, err := s.rawInput.Read( buf )
    if err != nil {
        return 0, err
    }
    decrypted := secret.DecryptAesGcm( s.info.ephmMaster, buf )
    s.input = bufio.NewReader( bytes.NewReader( decrypted ) )
    return s.input.Read( p )
}

func ( s *Session ) WriteEncrypted( p []byte ) ( int, error ) {
    s.writeRecordHeader( packetTypeEncrypted )
    writeVarint( s.conn, len( s.info.id ) )
    s.conn.Write( s.info.id )
    encrypted := secret.EncryptAesGcm( s.info.ephmMaster, p )
    return s.conn.Write( encrypted )
}

func ( s *Session ) Read( p []byte ) ( int, error ) {
    s.inMu.Lock()
    defer s.inMu.Unlock()

    if s.input != nil && s.input.Len() > 0 {
        return s.input.Read( p )
    }

    //
    prh, err := s.readRecordHeader()
    if err != nil {
        return err
    }

    switch prh.typ {
    case packetTypeSessionNotFound:
        // Not found
        atomic.StoreInt32( &s.handshaken, 0 )
        return 0, fmt.Errorf( "Need to perform another handshake" )
    case packetTypeHandshakeBegin:
        s.outMu.Lock()
        err := s.EndHandshake()
        if err != nil {
            return 0, err
        }
        s.outMu.Unlock()
        return s.Read( p )
    // case packetTypeHandshakeEnd:
    case packetTypeEncrypted:
        sessionId, err := readNextPacket( srvHs )
        if err != nil {
            return 0, err
        }
        if s.info == nil {
            // No info yet
            si, ok := cachedSessionInfos[string( sessionId )]
            if !ok {
                s.writeRecordHeader( packetTypeSessionNotFound )
                return 0, fmt.Errorf( "Session not found" )
            }
            s.info = si
        } else {
            if cmp := bytes.Compare( sessionId, s.info.id ); cmp != 0 {
                s.writeRecordHeader( packetTypeSessionNotFound )
                return 0, fmt.Errorf( "Session id mismatch" )
            }
        }
        return s.ReadEncrypted( p )
    }

    return 0, fmt.Errorf( "Invalid" )
}

func ( s *Session ) Write( p []byte ) ( int, error ) {
    s.outMu.Lock()
    defer s.outMu.Unlock()

    if !s.Handshaken() {
        s.inMu.Lock()
        err := s.BeginHandshake()
        if err != nil {
            return 0, err
        }
        s.inMu.Unlock()
    }

    return s.WriteEncrypted( p )
}

func HostnameOf( conn net.Conn ) ( string, error ) {
    host, _, err := net.SplitHostPort( conn.RemoteAddr().String() )
    if err != nil {
        return "", err
    }
    return host, err
}

func ( s *Session ) calculatePreMaster( pub []byte ) ( []byte, error ) {
    p256 := elliptic.P256()
    x, y, err := elliptic.Unmarshal( p256, pub )
    if err != nil {
        return nil, err
    }
    pmX, pmY := p256.ScalarMult( x, y, s.info.ephmPriv )
    pm, _ := elliptic.Marshal( p256, pmX, pmY )
    return pm, nil
}

func ( s *Session ) writeRecordHeader( typ byte ) error {

    buf := bytes.NewBuffer( nil )
    buf.Write( []byte( packetRecordHeaderStart ) )
    buf.Write( []byte{ packetVersionMajor, packetVersionMinor } )
    buf.Write( []byte{ typ } )
    buf.Write( []byte{ '\n' } )
    _, err := s.conn.Write( buf.Bytes() )
    return err

}

func ( s *Session ) BeginHandshake() ( err error ) {

    defer Catch( &err )

    if sessionKnownHosts == nil {
        return 0, fmt.Errorf( "Client must have known host list." )
    }

    err := s.writeRecordHeader( packetTypeHandshakeBegin )
    Try( err )

    host, err := HostnameOf( s.conn )
    Try( err )
    forDigest := bytes.NewBuffer( nil )

    // Block from client
    // | length(varint) | client random length | client random |
    // | public key length | public key |
    // | challenge length | challenge |
    si := NewSessionInfo()
    si.id = nil
    s.info = si
    clRnd := secret.Rand32Bytes()
    challenge := secret.Rand32Bytes()
    buf := bytes.NewBuffer( nil )
    mw := io.MultiWriter( buf, s.conn )
    err = writeByteSeriesPacket( mw, [][]byte{
        clRnd, s.info.ephmPub, challenge,
    } )
    Try( err )
    clHsMsg, err := readNextPacket( bytes.NewReader( buf.Bytes() ) )
    Try( err )
    forDigest.Write( clHsMsg )
    
    // Block from server
    // | length(varint) | server random length | server random |
    // | public key length | public key |
    // | signature length | public key signature signed with auth priv |
    // | challenge signature length | challenge signature |
    // | session id length | session id |

    //
    prh, err := s.readRecordHeader()
    Try( err )

    //
    if prh.typ != packetTypeHandshakeEnd {
        return fmt.Errorf( "Bad handshake record header" )
    }

    bx, err := readNextPacket( s.conn )
    Try( err )
    srvHs := bytes.NewReader( bx )
    srvRnd, err := readNextPacket( srvHs )
    Try( err )
    srvPub, err := readNextPacket( srvHs )
    Try( err )
    srvPubSig, err := readNextPacket( srvHs )
    Try( err )
    srvChallenge, err := readNextPacket( srvHs )
    Try( err )
    sessionId, err := readNextPacket( srvHs )
    Try( err )

    // Verify server pub
    verified := secret.VerifyP256( sessionKnownHosts[host], srvPub, srvPubSig )
    challengeResult := secret.VerifyP256( sessionKnownHosts[host], challenge, srvChallenge )

    if !( verified && challengeResult ) {
        return fmt.Errorf( "Invalid signature" )
    }

    forDigest.Write( bx )
    
    // Calc master secret
    // SHA256( PreMasterSecret || SHA256(digest) || clRnd || srvRnd )

    buf = bytes.NewBuffer( nil )
    preMaster := s.calculatePreMaster( srvPub )
    digest := sha256Sum( forDigest.Bytes() )
    buf.Write( preMaster )
    buf.Write( digest )
    buf.Write( clRnd )
    buf.Write( srvRnd )

    s.info.ephmMaster = Sha256Sum( buf.Bytes() )
    s.info.thirdPub = srvPub
    atomic.StoreInt32( &s.handshaken, 1 )

}

func ( s *Session ) EndHandshake() ( err error ) {

    defer Catch( &err )

    //
    if sessionAuthPriv == nil {
        return 0, fmt.Errorf( "Server must have an auth private key." )
    }

    //
    prh, err := s.readRecordHeader()
    Try( err )

    //
    if prh.typ != packetTypeHandshakeBegin {
        return fmt.Errorf( "Bad handshake record header" )
    }

    //
    forDigest := bytes.NewBuffer( nil )

    bx, err := readNextPacket( s.conn )
    Try( err )
    clRnd, err := readNextPacket( bx )
    Try( err )
    clPub, err := readNextPacket( bx )
    Try( err )
    clChallenge, err := readNextPacket( bx )
    Try( err )

    forDigest.Write( bx )

    // Response
    err := s.writeRecordHeader( packetTypeHandshakeEnd )
    Try( err )

    si := NewSessionInfo()
    srvRnd := rand32Bytes()
    srvPubSig := secret.SignP256( sessionAuthPriv, si.ephmPub )
    challengeSig := secret.SignP256( sessionAuthPriv, clChallenge )

    buf := bytes.NewBuffer( nil )
    mw := io.MultiWriter( buf, s.conn )
    err = writeByteSeriesPacket( mw, [][]byte{
        srvRnd, si.ephmPub, srvPubSig, challengeSig, si.id,
    } )
    Try( err )

    bx, err := readNextPacket( buf.Bytes() )
    Try( err )
    forDigest.Write( bx )

    // Calc master secret
    // SHA256( PreMasterSecret || SHA256(digest) || clRnd || srvRnd )

    buf = bytes.NewBuffer( nil )
    preMaster := s.calculatePreMaster( clPub )
    digest := sha256Sum( forDigest.Bytes() )
    buf.Write( preMaster )
    buf.Write( digest )
    buf.Write( clRnd )
    buf.Write( srvRnd )

    si.ephmMaster = sha256Sum( buf.Bytes() )
    si.thirdPub = clPub
    atomic.StoreInt32( &s.handshaken, 1 )

}

type byteReader struct {
    r io.Reader
}

func ( br byteReader ) ReadByte() ( byte, error ) {
    p := make( []byte, 1 )
    _, err := br.r.Read( p )
    if err != nil {
        return byte( 0 ), err
    }
    return p[0], nil
}

func writeVarint( w io.Writer, n int64 ) error {
    buf := make( []byte, binary.MaxVarintLen64 )
    p := binary.PutVarint( buf, n )
    _, err := w.Write( p.Bytes() )
    return err
}

func readVarint( r io.Reader ) ( int64, error ) {
    return binary.ReadVarint( byteReader{ r } )
}

func readNextPacket( r io.Reader ) ( []byte, error ) {
    n, err := readVarint( r )
    if err != nil {
        return nil, err
    }
    p := make( []byte, n )
    n2, err = r.Read( p )
    if err != nil || n != n2 {
        return nil, fmt.Errorf( "Bad packet" )
    }
    return p, nil
}

func writeByteSeriesPacket( w io.Writer bx [][]byte ) error {
    buf := bytes.NewBuffer( nil )
    for _, p := range bx {
        writeVarint( buf, len( p ) )
        buf.Write( p )
    }
    all := buf.Bytes()
    buf = bytes.NewBuffer( nil )
    writeVarint( buf, len( all ) )
    buf.Write( all )
    _, err := w.Write( buf.Bytes() )
    return err
}

type PacketRecordHeader struct {
    vMinor, vMajor, typ byte
}

func ( s *Session ) readRecordHeader() ( PacketRecordHeader, error ) {
    prhl := packeRecordtHeaderLen
    p := make( []byte, prhl )
    n, err := s.conn.Read( p )
    if err != nil || n != prhl {
        return PacketRecordHeader{}, fmt.Errorf( "Bad record header" )
    }

    // Check
    if p[:10] != packetRecordHeaderStart ||
        p[10] != packetVersionMajor ||
        p[11] != packetVersionMinor {
        return PacketRecordHeader{}, fmt.Errorf( "Bad record header" )
    }

    return PacketRecordHeader{
        vMajor: p[10], vMinor: p[11],  typ: p[12],
    }, nil
}

func ( s *Session ) WriteResponse( rp Response ) error {

}

func ( s *Session ) NextResponse() ( Response, error ) {

}

func ( s *Session ) Handshaken() bool {
    i := atomic.LoadInt32( &s.handshaken )
    return i == 1
}

type Response struct {
    name string
    headers map[string] []string
    args map[string] interface{}
}

var commaSplitRegexp = regexp.MustCompile( "\\s*,\\s*" )
func SplitComma( s string ) []string {
    return commaSplitRegexp.Split( s, -1 )
}
var headerSplitRegexp = regexp.MustCompile( "\\s*:\\s*" )
func SplitResponseHeader( s string ) []string {
    return headerSplitRegexp.Split( s, 2 )
}

func ReadResponse( r io.Reader ) ( Response, error ) {

    br := bufio.NewReader( r )
    
    start, err := br.ReadString( '\n' )
    if err != nil {
        return Response{}, err
    }

    startCols := strings.SplitN( start, " ", 2 )
    if startCols[0] != "TELESCRIBE" || len( startCols ) != 2 {
        return Response{}, fmt.Errorf( "Bad response block" )
    }
    name := startCols[1]

    headers := make( map[string] []string )
    for {
        l, err := br.ReadString( '\n' )
        if err != nil {
            return Response{}, err
        }
        headerCols := SplitResponseHeader( l )
        if len( headerCols ) != 2 {
            break
        }

        key := strings.ToLower( headerCols[0] )
        headers[key] = SplitComma( headerCols[1] )
    }

    contentLength, ok := headers["content-length"]
    if !ok {
        return Response{}, fmt.Errorf( "Bad response block" )
    }

    cl, err := strconv.Atoi( contentLength )
    if err != nil {
        return Response{}, err
    }

    body := io.LimitReader( br, cl )
    args := make( map[string] interface{} )
    dec := json.NewDecoder( body )
    err = dec.Decode( &args )
    if err != nil {
        return Response{}, err
    }

    return Response{
        name: name,
        headers: headers,
        args: args,
    }, nil

}

func NewResponse( name string ) Response {
    return Response{
        name: name,
        headers: make( map[string] []string ),
        args: make( map[string] interface{} ),
    }
}

func ( rp *Response ) Arg( key string ) interface{} {
    v, _ := rp.args[key]
    return v
}

func ( rp *Response ) Args() map[string] interface{} {
    return rp.args
}

func ( rp *Response ) SetArg( key string, val interface{} ) {
    rp.args[key] = val
}

func ( rp *Response ) SetArgs( args map[string] interface{} ) {
    rp.args = args
}

func ( rp *Response ) SetHeader( key, val string ) {
    key = strings.ToLower( key )
    rp.headers[key] = []string{ val }
}

func ( rp *Response ) AddHeader( key, val string ) {
    key = strings.ToLower( key )
    h, ok := rp.headers[key]
    if !ok {
        rp.headers[key] = []string{ val }
        return
    }
    rp.headers[key] = append( []string{ val }, h... )
}

func ( rp *Response ) SetHeaders( headers map[string] []string ) {
    rp.headers = make( map[string] []string )
    for key, h := range headers {
        key = strings.ToLower( key )
        for _, v := range h {
            rp.AddHeader( key, v )
        }
    }
}

func ( rp *Response ) Headers() map[string] []string {
    return rp.headers
}

func ( rp *Response ) Header( key string ) string {
    h, ok := rp.headers[key]
    if !ok || len( h ) == 0 {
        return ""
    }
    return h[0]
}