package secret

type Key interface {
    Fingerprint() string
    Base64() string
    Bytes() []byte
}

type Signer interface {
    Key
    Sign([]byte) []byte
}

type Verifier interface {
    Key
    Verify([]byte, []byte) bool
}