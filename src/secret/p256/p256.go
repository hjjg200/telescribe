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

const Prefix = "ECC_P256"
var p256 = elliptic.P256()

type PublicKey struct {
    X, Y *big.Int
}

type PrivateKey struct {
    PublicKey
    D *big.Int // Plain private key
}

func SerializePublicKey(pub *PublicKey) string {
    b := elliptic.Marshal(p256, pub.X, pub.Y)
    return secret.EncodeBase64(b)
}

func DeserializePublicKey(str string) (*PublicKey, error) {
    b, err := secret.DecodeBase64(str)
    if err != nil {
        return nil, err
    }
    X, Y := elliptic.Unmarshal(p256, b)
    return &PublicKey{
        X: X,
        Y: Y,
    }, nil
}

func SerializePrivateKey(priv *PrivateKey) string {
    return secret.EncodeBase64(priv.D.Bytes())
}

func DeserializePrivateKey(str string) (*PrivateKey, error) {
    b, err := secret.DecodeBase64(str)
    if err != nil {
        return nil, err
    }
    D := new(big.Int).SetBytes(b)
    X, Y := p256.ScalarBaseMult(b)
    return &PrivateKey{
        PublicKey: PublicKey{
            X: X,
            Y: Y,
        },
        D: D,
    }, nil
}

//
// PRIVATE KEY
//

func (priv *PrivateKey) Ecdsa() *ecdsa.PrivateKey {
    pub := priv.PublicKey.Ecdsa()
    return &ecdsa.PrivateKey{
        PublicKey: *pub,
        D: priv.D,
    }
}

func (priv *PrivateKey) Fingerprint() string {
    return Prefix + " " + secret.Sha256Fingerprint(priv)
}

func (priv *PrivateKey) Base64() string {
    return secret.EncodeBase64(priv.Bytes())
}

func (priv *PrivateKey) Bytes() []byte {
    return priv.D.Bytes()
}

func (priv *PrivateKey) SetBase64(b64 string) (*PrivateKey, bool) {
    b, err := secret.DecodeBase64(b64)
    if err != nil {
        return priv, false
    }
    return priv.SetBytes(b)
}

func (priv *PrivateKey) SetBytes(b []byte) (*PrivateKey, bool) {
    priv.D.SetBytes(b)
    return priv, true
}

func (priv *PrivateKey) Sign(data []byte) []byte {
    return Sign(priv, data)
}

//
// PUBLIC KEY
//

func (pub *PublicKey) Ecdsa() *ecdsa.PublicKey {
    return &ecdsa.PublicKey{
        Curve: p256,
        X: pub.X,
        Y: pub.Y,
    }
}

func (pub *PublicKey) Fingerprint() string {
    return Prefix + " " + secret.Sha256Fingerprint(pub)
}

func (pub *PublicKey) Base64() string {
    return secret.EncodeBase64(pub.Bytes())
}

func (pub *PublicKey) Bytes() []byte {
    b := elliptic.Marshal(p256, pub.X, pub.Y)
    return b
}

func (pub *PublicKey) SetBase64(b64 string) (*PublicKey, bool) {
    b, err := secret.DecodeBase64(b64)
    if err != nil {
        return pub, false
    }
    return pub.SetBytes(b)
}

func (pub *PublicKey) SetBytes(p []byte) (*PublicKey, bool) {
    x, y := elliptic.Unmarshal(p256, p)
    if x == nil {
        return pub, false
    }
    pub.X, pub.Y = x, y
    return pub, true
}

func (pub *PublicKey) Verify(data []byte, signature []byte) bool {
    return Verify(pub, data, signature)
}

func GenerateKey() *PrivateKey {
    priv, _ := ecdsa.GenerateKey(p256, rand.Reader)
    return &PrivateKey{
        PublicKey: PublicKey{
            X: priv.PublicKey.X,
            Y: priv.PublicKey.Y,
        },
        D: priv.D,
    }
}

func Sign(priv *PrivateKey, data []byte) []byte {

    // Hash
    h := sha256.New()
    h.Write(data)
    r, s, _ := ecdsa.Sign(rand.Reader, priv.Ecdsa(), h.Sum(nil)[:])

    // Encode
    buf := bytes.NewBuffer(nil)
    enc := gob.NewEncoder(buf)
    enc.Encode(r)
    enc.Encode(s)

    return buf.Bytes()
}

func Verify(pub *PublicKey, data []byte, signature []byte) bool {

    var r, s *big.Int
    dec := gob.NewDecoder(bytes.NewReader(signature))
    err := dec.Decode(&r)
    if err != nil {
        return false
    }
    err = dec.Decode(&s)
    if err != nil {
        return false
    }

    h := sha256.New()
    h.Write(data)

    return ecdsa.Verify(pub.Ecdsa(), h.Sum(nil)[:], r, s)

}