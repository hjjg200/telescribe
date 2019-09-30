package secret

import (
    "crypto/rand"
)


/*

type Session
Listener
Dialer

Client
- Random P256 Public Key
- Client Random

Server
- Session ID
- Random P256 Public Key
- Signature of Public Key signed with preexisting P256 private key
- Server Random

Client
* Generate master secret
- Send message

Server
* Master secret
- Send message


*/


func RandomBytes( l int ) []byte {
    p := make( []byte, l )
    rand.Read( p )
    return p
}

/*

func Rand32Bytes() []byte {
    key := make( []byte, 32 )
    _, _ = rand.Read( key )
    // https://golang.org/pkg/math/rand/#Read
    // Read generates len(p) random bytes from the default Source and writes them into p.
    // It always returns len(p) and a nil error.
    return key
}

func RandomPrivateKey() *rsa.PrivateKey {
    rsaPriv, _ := rsa.GenerateKey( rand.Reader, 2048 )
    return rsaPriv
}

func FingerprintPublicKey( rsaPub *rsa.PublicKey ) string {
    h := sha256.New()
    enc := gob.NewEncoder( h )
    if err := enc.Encode( rsaPub ); err != nil {
        panic( err )
    }
    hashed := h.Sum( nil )[:]
    fp := fmt.Sprintf( "%d SHA256", rsaPub.N.BitLen() )
    for _, b := range hashed[:] {
        fp += fmt.Sprintf( ":%02x", b )
    }
    return fp
}

func SerializePublicKey( rsaPub *rsa.PublicKey ) string {
    buf := bytes.NewBuffer( nil )
    enc := gob.NewEncoder( buf )
    if err := enc.Encode( rsaPub ); err != nil {
        panic( err )
    }
    return base64.StdEncoding.EncodeToString( buf.Bytes() )
}

func DeserializePublicKey( b64 string ) ( *rsa.PublicKey, error ) {
    gobBytes, err := base64.StdEncoding.DecodeString( b64 )
    if err != nil {
        return nil, err
    }
    rd := bytes.NewReader( gobBytes )
    dec := gob.NewDecoder( rd )
    var rsaPub rsa.PublicKey
    if err = dec.Decode( &rsaPub ); err != nil {
        return nil, err
    }
    return &rsaPub, nil
}

func SerializePrivateKey( rsaPriv *rsa.PrivateKey ) string {
    buf := bytes.NewBuffer( nil )
    enc := gob.NewEncoder( buf )
    if err := enc.Encode( rsaPriv ); err != nil {
        panic( err )
    }
    return base64.StdEncoding.EncodeToString( buf.Bytes() )
}

func DeserializePrivateKey( b64 string ) ( *rsa.PrivateKey, error ) {
    gobBytes, err := base64.StdEncoding.DecodeString( b64 )
    if err != nil {
        return nil, err
    }
    rd := bytes.NewReader( gobBytes )
    dec := gob.NewDecoder( rd )
    var rsaPriv rsa.PrivateKey
    if err = dec.Decode( &rsaPriv ); err != nil {
        return nil, err
    }
    return &rsaPriv, nil
}

func Encrypt( rsaPub *rsa.PublicKey, data []byte ) []byte {

    // Block
    // | RSA-Encrypted AES Key(256) | AES Encrypted Data(variable) |

    // AES PART
    aesSymKey := Rand32Bytes()
    aesBlock, _ := aes.NewCipher( aesSymKey )
    // symKey is guaranteed to be 32 bytes long
    aesGcm, _ := cipher.NewGCM( aesBlock )
    nonce := make( []byte, aesGcm.NonceSize() )
    // As the key is always generated randomly, made nonce always the same
    aesEncryptedData := aesGcm.Seal( nil, nonce, data, nil )

    // RSA PART
    rsaEncryptedAesSymKey, _ := rsa.EncryptOAEP( sha256.New(), rand.Reader, rsaPub, aesSymKey, nil )

    return append( rsaEncryptedAesSymKey, aesEncryptedData... )

}

func Decrypt( rsaPriv *rsa.PrivateKey, block []byte ) ( []byte, error ) {

    // Len
    if len( block ) <= 256 {
        return nil, fmt.Errorf( "Bad encrypted block" )
    }

    // The first 256 bytes are the rsa-encrypted aes key
    rsaEncryptedAesSymKey := block[:256]
    aesSymKey, err := rsa.DecryptOAEP(
        sha256.New(), rand.Reader, rsaPriv, rsaEncryptedAesSymKey, nil,
    )
    if err != nil {
        return nil, err
    }

    // Decrypt
    aesEncryptedData := block[256:]
    aesBlock, err := aes.NewCipher( aesSymKey )
    if err != nil { return nil, err }
    aesGcm, err := cipher.NewGCM( aesBlock )
    if err != nil { return nil, err }
    nonce := make( []byte, aesGcm.NonceSize() )

    return aesGcm.Open( nil, nonce, aesEncryptedData, nil )

}

func Sign( rsaPriv *rsa.PrivateKey, data []byte ) []byte {
    h := sha256.New()
    h.Write( data )
    signed, _ := rsa.SignPSS( rand.Reader, rsaPriv, crypto.SHA256, h.Sum( nil )[:], nil )
    return signed
}

func Verify( rsaPub *rsa.PublicKey, data, signed []byte ) bool {
    h := sha256.New()
    h.Write( data )
    err := rsa.VerifyPSS( rsaPub, crypto.SHA256, h.Sum( nil )[:], signed[:], nil )
    if err != nil {
        return false
    }
    return true
}

*/