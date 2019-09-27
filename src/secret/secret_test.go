package secret

import (
    "testing"
)

func TestPublicKeySerialization( t *testing.T ) {

    rsaPriv := RandomPrivateKey()
    rsaPub := &rsaPriv.PublicKey

    // Serialize
    serialized := SerializePublicKey( rsaPub )
    deserialized, err := DeserializePublicKey( serialized )
    if err != nil {
        t.Error( err )
        return
    }

    // Check
    if rsaPub.N.Cmp( deserialized.N ) != 0 || rsaPub.E != deserialized.E {
        t.Error( "Public key serialization error" )
        return
    }

    // Print
    t.Logf( "Original public key: %v\n", *rsaPub )
    t.Logf( "Serialized public key: %s\n", serialized )
    t.Logf( "Deserialized public key: %v\n", *deserialized )

}


func TestPrivateKeySerialization( t *testing.T ) {

    rsaPriv := RandomPrivateKey()

    // Serialize
    serialized := SerializePrivateKey( rsaPriv )
    deserialized, err := DeserializePrivateKey( serialized )
    if err != nil {
        t.Error( err )
        return
    }

    // Print
    t.Logf( "Original private key: %v\n", *rsaPriv )
    t.Logf( "Serialized private key: %s\n", serialized )
    t.Logf( "Deserialized private key: %v\n", *deserialized )

}

func TestEncrpytAndDecrypt( t *testing.T ) {

    rsaPriv := RandomPrivateKey()
    rsaPub := &rsaPriv.PublicKey
    txt := "This is secret"
    enc := Encrypt( rsaPub, []byte( txt ) )
    dec, err := Decrypt( rsaPriv, enc )

    // Check
    if err != nil {
        t.Error( err )
        return
    }

    if txt != string( dec ) {
        t.Error( "Wrongly decrypted data" )
        return
    }

    // Print
    t.Logf( "Original: %s\n", txt )
    t.Logf( "Encrypted block: %x\n", enc )
    t.Logf( "Decrypted: %s\n", dec )

}

func TestSignAndVerify( t *testing.T ) {
    
    rsaPriv := RandomPrivateKey()
    rsaPub := &rsaPriv.PublicKey
    data := []byte( "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer vitae sem commodo, dignissim mi ut, bibendum tellus. Nullam vel orci efficitur." )

    signed := Sign( rsaPriv, data )
    verified := Verify( rsaPub, data, signed )

    if !verified {
        t.Errorf( "Failed verification!" )
        return
    }

    t.Logf( "Data: %s", data )
    t.Logf( "Signature: %x", signed )
    t.Logf( "Verified: %t", verified )

}

func TestPublicKeyFingerprint( t *testing.T ) {

    rsaPriv := RandomPrivateKey()
    rsaPub := &rsaPriv.PublicKey

    fp := FingerprintPublicKey( rsaPub )
    t.Logf( fp )

}