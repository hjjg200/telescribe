package p256

// P256 is the most common and go has asm implementaion of P256 and thus it is the fastest.

import (
    "bytes"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "encoding/gob"
    "math/big"
    ".."
)

var p256 = elliptic.P256()

type PublicKey struct {
    elliptic.Curve
    X, Y *big.Int
}

type PrivateKey struct {
    PublicKey
    D *big.Int // Plain private key
}

func SerializePublicKey( pub *PublicKey ) string {
    b := elliptic.Marshal( p256, pub.X, pub.Y )
    return secret.EncodeBase64( b )
}

func DeserializePublicKey( str string ) ( *PublicKey, error ) {
    b, err := secret.DecodeBase64( str )
    if err != nil {
        return nil, err
    }
    X, Y := elliptic.Unmarshal( p256, b )
    return &PublicKey{
        Curve: p256,
        X: X,
        Y: Y,
    }, nil
}

func SerializePrivateKey( priv *PrivateKey ) string {
    return secret.EncodeBase64( priv.D.Bytes() )
}

func DeserializePrivateKey( str string ) ( *PrivateKey, error ) {
    b, err := secret.DecodeBase64( str )
    if err != nil {
        return nil, err
    }
    D := new( big.Int ).SetBytes( b )
    X, Y := p256.ScalarBaseMult( b )
    return &PrivateKey{
        PublicKey: PublicKey{
            Curve: p256,
            X: X,
            Y: Y,
        },
        D: D,
    }, nil
}

func ( priv *PrivateKey ) Ecdsa() *ecdsa.PrivateKey {
    return &ecdsa.PrivateKey{
        PublicKey: *( priv.PublicKey.Ecdsa() ),
        D: priv.D,
    }
}

func ( pub *PublicKey ) Ecdsa() *ecdsa.PublicKey {
    return &ecdsa.PublicKey{
        Curve: pub.Curve,
        X: pub.X,
        Y: pub.Y,
    }
}

func GenerateKey() *PrivateKey {
    priv, _ := ecdsa.GenerateKey( p256, rand.Reader )
    return &PrivateKey{
        PublicKey: PublicKey{
            Curve: p256,
            X: priv.PublicKey.X,
            Y: priv.PublicKey.Y,
        },
        D: priv.D,
    }
}

func Sign( priv *PrivateKey, data []byte ) []byte {

    // Hash
    h := sha256.New()
    h.Write( data )
    r, s, _ := ecdsa.Sign( rand.Reader, priv.Ecdsa(), h.Sum( nil )[:] )

    // Encode
    buf := bytes.NewBuffer( nil )
    enc := gob.NewEncoder( buf )
    enc.Encode( r )
    enc.Encode( s )

    return buf.Bytes()
}

func Verify( pub *PublicKey, data []byte, signature []byte ) bool {

    var r, s *big.Int
    dec := gob.NewDecoder( bytes.NewReader( signature ) )
    err := dec.Decode( &r )
    if err != nil {
        return false
    }
    err = dec.Decode( &s )
    if err != nil {
        return false
    }

    h := sha256.New()
    h.Write( data )

    return ecdsa.Verify( pub.Ecdsa(), h.Sum( nil )[:], r, s )

}