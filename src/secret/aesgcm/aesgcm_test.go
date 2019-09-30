package aesgcm

import (
    "testing"
)

func TestEncryption( t *testing.T ) {

    key := GenerateKey()
    data := []byte( "Secret" )
    encrypted := Encrypt( key, data )
    t.Logf( "%v\n", encrypted )
    decrypted, err := Decrypt( key, encrypted )
    if err != nil {
        t.Error( err )
        return
    }
    t.Logf( "%s\n", decrypted )

}
