package main

import (
    "bytes"
    "crypto/sha256"
    "crypto/rand"
    "fmt"
    "io"
    "os"
    "strings"
    "regexp"
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

func Sha256Sum(bx ...[]byte) []byte {
    h := sha256.New()
    for _, b := range bx {
        h.Write(b)
    }
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

//
// FILE
//

func ReadFile(fn string, mode os.FileMode) ([]byte, error) {
    f, err := os.OpenFile(fn, os.O_RDONLY, mode)
    if err != nil {
        return nil, err
    }
    buf := bytes.NewBuffer(nil)
    _, err = io.Copy(buf, f)
    if err != nil {
        return nil, err
    }
    f.Close()
    return buf.Bytes(), nil
}

func TouchFile(fn string, mode os.FileMode) error {
    f, err := os.OpenFile(fn, os.O_RDONLY | os.O_CREATE, mode)
    if err != nil {
        return err
    }
    f.Close()
    return nil
}

//
// REGEXP
//

var whitespaceRegexp = regexp.MustCompile("[\\s\\t ]+")
func SplitWhitespace(str string) []string {
    return whitespaceRegexp.Split(str, -1)
}
func SplitWhitespaceN(str string, i int) []string {
    return whitespaceRegexp.Split(str, i)
}

var linebreakRegexp = regexp.MustCompile("\\r\\n|\\n")
func SplitLines(str string) []string {
    return linebreakRegexp.Split(str, -1)
}

//
// FULLNAME
//

func parseFullName(fn string) (string, string) {
    defer recover() // Possible index panic
    i := strings.Index(fn, "(")
    return fn[i + 1:len(fn) - 1], fn[:i]
}

func formatFullName(host, alias string) string {
    return alias + "(" + host + ")"
}

//
// SORT
//

type Int64Slice []int64
func (s Int64Slice) Len() int { return len(s) }
func (s Int64Slice) Less(i, j int) bool { return s[i] < s[j] }
func (s Int64Slice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }