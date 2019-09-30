package aesgcm

import (
    "crypto/aes"
    "crypto/cipher"
    ".."
)

type Key struct {
    key []byte
}

func NewKey(k []byte) *Key {
    if len(k) != 32 {
        return nil
    }
    return &Key{
        key: k,
    }
}

func GenerateKey() *Key {
    return &Key{
        key: secret.RandomBytes( 32 ),
    }
}

func Encrypt( key *Key, data []byte ) []byte {

    // key.key is guaranteed to be 32 bytes long
    aesBlock, _ := aes.NewCipher( key.key )
    aesGcm, _ := cipher.NewGCM( aesBlock )
    nonce := secret.RandomBytes( aesGcm.NonceSize() )
    // As the key is always generated randomly, made nonce always the same
    encrypted := aesGcm.Seal( nil, nonce, data, nil )

    return append( nonce, encrypted... )

}

func Decrypt( key *Key, block []byte ) ( []byte, error ) {
    
    // Decrypt
    aesBlock, err := aes.NewCipher( key.key )
    if err != nil { return nil, err }
    aesGcm, err := cipher.NewGCM( aesBlock )
    if err != nil { return nil, err }
    nsz := aesGcm.NonceSize()
    nonce := block[:nsz]
    encrypted := block[nsz:]

    return aesGcm.Open( nil, nonce, encrypted, nil )

}