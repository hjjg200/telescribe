package secret

import (
    "crypto/sha256"
    "encoding/base64"
    "fmt"
)

func EncodeBase64(data []byte) string {
    return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(str string) ([]byte, error) {
    return base64.StdEncoding.DecodeString(str)
}

func ToByteSeries(p []byte) string {
    ret := ""
    for _, b := range p {
        ret += fmt.Sprintf(":%02x", b)
    }
    return ret[1:] // Remove first colon
}

func Sha256Fingerprint(k Key) string {
    h := sha256.New()
    h.Write(k.Bytes())
    return "SHA256:" + ToByteSeries(h.Sum(nil)[:])
}