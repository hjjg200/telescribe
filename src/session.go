package main

const packetVersionMajor = byte( 0x00 )
const packetVersionMinor = byte( 0x05 )

type Session struct {
    id []byte
    authPriv []byte // elliptic P256 very very secret; when nil, cannot act as a server
    authPrivFile string
    ephmPriv []byte // elliptic P256, ephemeral
    ephmPub []byte // elliptic P256, ephemeral
    thirdPub []byte // third party's public
    ephmMaster []byte

    knownHostsFile string
    knownHosts map[string] []byte // elliptic P256, when nil, cannot act as a client
}

var cachedSessions map[string] *Session

func init() {
    cachedSessions = make( map[string] *Session )
}

func NewSession() *Session {

    p256 := elliptic.P256()
    priv, pubx, puby, _ := elliptic.GenerateKey( p256, rand.Reader )
    pub := elliptic.Marshal( p256, pubx, puby )
    s := &Session{
        id: make( []byte, 32 ),
        ephmPriv: priv,
        ephmPub: pub,
        knownHosts: make( map[string] string ),
    }

    return s

}

func GetSession( id []byte ) ( *Session, bool ) {
    s, ok := cachedSessions[string( id )]
}

func ( s *Session ) LoadKnownHosts( fn string ) error {

    f, err := os.OpenFile( fn, os.O_RDONLY, 0400 )
    if err != nil {
        return err
    }
    buf := bytes.NewBuffer( nil )
    io.Copy( buf, f )
    f.Close()

    //
    s.knownHostsFile = fn
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
        s.knownHosts[cols[0]] = cols[1]
    }
    return nil

}

func ( s *Session ) LoadAuthPrivateKey( fn string ) error {

    f, err := os.OpenFile( fn, os.O_RDONLY, 0400 )
    if err != nil {
        return err
    }
    buf := bytes.NewBuffer( nil )
    io.Copy( buf, f )
    f.Close()

    //
    s.authPrivFile = fn
    s.authPriv = buf.Bytes()
    return nil

}

type SessionConn struct {
    handshaken int32
    conn net.Conn
    inMu sync.Mutex
    outMu sync.Mutex
    rawInput bytes.Buffer
    input bytes.Reader
    localAddr net.Addr
    remoteAddr net.Addr
    readDeadline time.Time
    writeDeadline time.Time
}

const (
    packetRecordHeaderStart = "TELESCRIBE"
    packeRecordtHeaderLen = 13
    packetTypeHandshakeBegin = byte( 0x01 )
    packetTypeHandshakeEnd = byte( 0x02 )
    packetTypeEncrypted = byte( 0x11 )
)

// Response/Request
//
//
//
//
//
//

func ( sc *SessionConn ) calculatePreMaster( pub []byte ) ( []byte, error ) {
    p256 := elliptic.P256()
    x, y, err := elliptic.Unmarshal( p256, pub )
    if err != nil {
        return nil, err
    }
    pmX, pmY := p256.ScalarMult( x, y, sc.ephmPriv )
    pm, _ := elliptic.Marshal( p256, pmX, pmY )
    return pm, nil
}

func ( sc *SessionConn ) readRawN( n int64 ) error {
    // Mutex is accessed from the parent function

    // Reset
    sc.rawInput.Reset( nil )
    rn, err := io.CopyN( sc.rawInput, sc.conn, n )
    if err != nil {
        return err
    }
    if rn != n {
        return fmt.Errorf( "Read bytes do not match n" )
    }

    return nil
}

func ( sc *SessionConn ) writeRecordHeader( typ byte ) error {

    buf := bytes.NewBuffer( nil )
    buf.Write( []byte( packetRecordHeaderStart ) )
    buf.Write( []byte{ packetVersionMajor, packetVersionMinor } )
    buf.Write( []byte{ typ } )
    _, err := sc.conn.Write( buf.Bytes() )
    return err

}

func ( sc *SessionConn ) BeginHandshake() error {

    err := sc.writeRecordHeader( packetTypeHandshakeBegin )
    if err != nil {
        return err
    }

    forDigest := bytes.NewBuffer( nil )

    // Block from client
    // | length(varint) | client random length | client random |
    // | public key length | public key |
    // | challenge length | challenge |
    clRnd := rand32Bytes()
    challenge := rand32Bytes()
    buf := bytes.NewBuffer( nil )
    mw := io.MultiWriter( buf, sc.conn )
    err = writeByteSeriesPacket( mw, [][]byte{
        clRnd, sc.ephmPub, challenge,
    } )
    if err != nil {
        return err
    }
    clHsMsg, err := readNextPacket( bytes.NewReader( buf.Bytes() ) )
    if err != nil {
        return err
    }
    forDigest.Write( clHsMsg )
    
    // Block from server
    // | length(varint) | server random length | server random |
    // | public key length | public key |
    // | signature length | public key signature signed with auth priv |
    // | challenge signature length | challenge signature |
    // | session id length | session id |

    //
    prh, err := sc.readRecordHeader()
    if err != nil {
        return err
    }

    //
    if prh.typ != packetTypeHandshakeEnd {
        return fmt.Errorf( "Bad handshake record header" )
    }

    bx, err := readNextPacket( sc.conn )
    if err != nil {
        return err
    }
    srvHs := bytes.NewReader( bx )
    srvRnd, err := readNextPacket( srvHs )
    if err != nil {
        return err
    }
    srvPub, err := readNextPacket( srvHs )
    if err != nil {
        return err
    }
    srvPubSig, err := readNextPacket( srvHs )
    if err != nil {
        return err
    }
    srvChallenge, err := readNextPacket( srvHs )
    if err != nil {
        return err
    }
    sessionId, err := readNextPacket( srvHs )
    if err != nil {
        return err
    }

    // Verify server pub
    verified := secret.Verify( sc.knownHosts[host], srvPub, srvPubSig )
    challengeResult := secret.Verify( sc.knownHosts[host], challenge, srvChallenge )

    if !( verified && challengeResult ) {
        return fmt.Errorf( "Invalid signature" )
    }

    forDigest.Write( bx )
    
    // Calc master secret
    // SHA256( PreMasterSecret || SHA256(digest) || clRnd || srvRnd )

    buf = bytes.NewBuffer( nil )
    preMaster := sc.calculatePreMaster( srvPub )
    digest := sha256Sum( forDigest.Bytes() )
    buf.Write( preMaster )
    buf.Write( digest )
    buf.Write( clRnd )
    buf.Write( srvRnd )

    sc.ephmMaster = sha256Sum( buf.Bytes() )
    sc.handshaken = true

}

func ( sc *SessionConn ) EndHandshake() error {

    //
    prh, err := sc.readRecordHeader()
    if err != nil {
        return err
    }

    //
    if prh.typ != packetTypeHandshakeBegin {
        return fmt.Errorf( "Bad handshake record header" )
    }

    //
    forDigest := bytes.NewBuffer( nil )

    bx, err := readNextPacket( sc.conn )
    if err != nil {
        return err
    }
    clRnd, err := readNextPacket( bx )
    if err != nil {
        return err
    }
    clPub, err := readNextPacket( bx )
    if err != nil {
        return err
    }
    clChallenge, err := readNextPacket( bx )
    if err != nil {
        return err
    }

    forDigest.Write( bx )

    // Response
    err := sc.writeRecordHeader( packetTypeHandshakeEnd )
    if err != nil {
        return err
    }

    sId := rand32Bytes()
    srvRnd := rand32Bytes()
    srvPubSig := secret.Sign( sc.authPriv, sc.ephmPub )
    challengeSig := secret.Sign( sc.authPriv, clChallenge )

    buf := bytes.NewBuffer( nil )
    mw := io.MultiWriter( buf, sc.conn )
    err = writeByteSeriesPacket( mw, [][]byte{
        srvRnd, sc.ephmPub, srvPubSig, challengeSig, sId,
    } )
    if err != nil {
        return err
    }

    bx, err := readNextPacket( buf.Bytes() )
    if err != nil {
        return err
    }
    forDigest.Write( bx )

    // Calc master secret
    // SHA256( PreMasterSecret || SHA256(digest) || clRnd || srvRnd )

    buf = bytes.NewBuffer( nil )
    preMaster := sc.calculatePreMaster( clPub )
    digest := sha256Sum( forDigest.Bytes() )
    buf.Write( preMaster )
    buf.Write( digest )
    buf.Write( clRnd )
    buf.Write( srvRnd )

    sc.ephmMaster = sha256Sum( buf.Bytes() )
    sc.handshaken = true

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

func ( sc *SessionConn ) readRecordHeader() ( PacketRecordHeader, error ) {
    prhl := packeRecordtHeaderLen
    p := make( []byte, prhl )
    n, err := sc.conn.Read( p )
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

func ( sc *SessionConn ) WriteResponse( rp Response ) error {

}

func ( sc *SessionConn ) NextResponse() ( Response, error ) {

}

func ( sc *SessionConn ) Handshaken() bool {
    i := atomic.LoadInt32( &sc.handshaken )
    return i == 1
}