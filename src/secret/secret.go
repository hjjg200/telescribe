package secret

import (
    "crypto/rand"
)

func RandomBytes(l int) []byte {
    p := make([]byte, l)
    rand.Read(p)
    return p
}
