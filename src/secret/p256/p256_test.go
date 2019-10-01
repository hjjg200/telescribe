package p256

import (
    "testing"
    ".."
)

func TestInterface(t *testing.T) {
    priv := GenerateKey()
    var sPriv secret.Signer = priv
    var sPub secret.Verifier = &priv.PublicKey
    _ = sPriv
    _ = sPub
}

func TestSigning(t *testing.T) {

    priv := GenerateKey()
    data := []byte("Secret")
    signature := Sign(priv, data)
    t.Logf("Signature: %v\n", signature)
    verified := Verify(&priv.PublicKey, data, signature)
    t.Logf("Verified: %t\n", verified)
    if !verified {
        t.Error("Failed to verify")
        return
    }

}