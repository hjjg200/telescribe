package main

import (
    "crypto/sha256"
    "crypto/rand"
    "fmt"
    "os"
)

func EnsureDirectory(p string) error {
    st, err := os.Stat(p)
    if err != nil {
        if os.IsNotExist(err) {
            err = os.MkdirAll(p, 0755)
            if err != nil {
                return err
            }
        }
        return err
    }
    if !st.IsDir() {
        return fmt.Errorf("Server cache directory path does not resolve to a directory!")
    }
    return nil
}

func Sha256Sum(b []byte) []byte {
    h := sha256.New()
    h.Write(b)
    return h.Sum(nil)[:]
}

const alphaNumSeries = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
func RandomAlphaNum(l int) string {
    b := make([]byte, l)
    rand.Read(b)
    for i := range b {
        b[i] = alphaNumSeries[int(b[i]) % len(alphaNumSeries)]
    }
    return string(b)
}