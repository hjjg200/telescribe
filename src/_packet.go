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
    "./secret"
)

type Instance interface {
    MyPrivateKey() *rsa.PrivateKey
    TheirPublicKey() *rsa.PublicKey
    SetTheirPublicKey( *rsa.PublicKey )
}

type Response struct {
    Name string
    Args map[string] interface{}
}

const (
    flResponseEncrypted = 1 << iota
)





func ReadNextJsonResponse( i Instance, conn net.Conn ) ( rsp Response, err error ) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf( "%v", r )
        }
    }()

    // New Response Block
    // TELESCRIBE\r\n
    // Content-Length: ...\r\n
    // \r\n
    // ...
    //

    rd := bufio.NewReader( conn )
    startLine, err := rd.ReadString( '\n' )
    if startLine[:10] != "TELESCRIBE" || err != nil {
        return Response{}, fmt.Errorf( "Bad response block" )
    }

    nextHeader := func() ( string, string, bool ) {
        l, err := rd.ReadString( '\n' )
        if err != nil {
            return "", "", false
        }
        l = strings.Trim( l, "\r\n" )
        sp := SplitResponseHeader( l )
        if len( sp ) < 2 {
            return "", "", false
        }
        return sp[0], sp[1], true
    }

    headers := make( map[string] string )
    for {
        key, val, ok := nextHeader()
        if !ok {
            break
        }
        headers[key] = val
    }

    // Vals
    contentLength, err := strconv.Atoi( headers["Content-Length"] )
    if err != nil {
        return
    }
    contentType := headers["Content-Type"]

    // Body
    buf := bytes.NewBuffer( nil )
    _, err = io.CopyN( buf, rd, int64( contentLength ) )
    if err != nil {
        return
    }
    body := buf.Bytes()

    switch contentType {
    case "application/json":
    case "application/json+encrypted":
        body, err = secret.Decrypt( i.MyPrivateKey(), body )
        if err != nil {
            return
        }
    default:
        err = fmt.Errorf( "Bad content type" )
        return
    }

    // Json
    m, err := UnmarshalJsonResponse( body )
    if err != nil {
        return
    }
    rsp, err = MapToResponse( m )
    return

}

func WriteJsonResponse( i Instance, conn net.Conn, r Response ) error {
    return writeJsonResponse( i, conn, r, false )
}

func WriteEncryptedJsonResponse( i Instance, conn net.Conn, r Response ) error {
    return writeJsonResponse( i, conn, r, true )
}

func writeJsonResponse( i Instance, conn net.Conn, r Response, encrypted bool ) error {

    //
    rsp, err := MarshalJsonResponse( r.Name, r.Args )
    if err != nil {
        return err
    }

    contentType := "application/json"

    // Encrypted
    if encrypted {
        contentType += "+encrypted"
        rsp = secret.Encrypt( i.TheirPublicKey(), rsp )
    }

    contentLength := len( rsp )

    // Wrtie

    body := "TELESCRIBE\r\n"
    body += "Content-Type: " + contentType + "\r\n"
    body += "Content-Length:" + fmt.Sprint( contentLength ) + "\r\n"
    body += "\r\n"
    body += string( rsp )

    _, err = conn.Write( []byte( body ) )
    if err != nil {
        return err
    }

    return nil

}

func MapToResponse( m map[string] interface{} ) ( r Response, err error ) {
    defer func() {
        rc := recover()
        if rc != nil {
            err = fmt.Errorf( "%v", rc )
        }
    }()
    r.Name = m["name"].( string )
    delete( m, "name" )
    r.Args = m
    return
}

func ( r *Response ) SetName( s string ) {
    r.Name = s
}

func ( r *Response ) Set( key string, val interface{} ) {
    r.Args[key] = val
}

func ( r Response ) String( key string ) ( str string ) {
    str, _ = r.Args[key].( string )
    return
}

func ( r Response ) Int( key string ) ( i int ) {
    i, _ = r.Args[key].( int )
    return
}

func ( r Response ) Float64( key string ) ( f float64 ) {
    f, _ = r.Args[key].( float64 )
    return
}

func ( r Response ) Bytes( key string ) []byte {
    // Json uses base64 to encode []byte
    b, err := base64.StdEncoding.DecodeString( r.String( key ) )
    if err != nil {
        return nil
    }
    return b
}

func packStringPacket( str string ) []byte {
    b := make( []byte, binary.MaxVarintLen64 )
    n := binary.PutVarint( b, int64( len( []byte( str ) ) ) )
    b = b[:n]
    return append( b, []byte( str )... )
}

type byteReader struct {
    rd io.Reader
}
func ( br byteReader ) ReadByte() ( byte, error ) {
    b := make( []byte, 1 )
    _, err := br.rd.Read( b )
    if err != nil {
        return byte( 0 ), err
    }
    return b[0], nil
}

func readStringPacket( rd io.Reader ) ( string, error ) {
    rb := byteReader{ rd }
    l, err := binary.ReadVarint( rb )
    if err != nil {
        return "", err
    }
    b := make( []byte, l )
    _, err = rd.Read( b )
    if err != nil {
        return "", err
    }
    return string( b ), nil
}