package secret

import (
    "encoding/base64"
)

func EncodeBase64( data []byte ) string {
    return base64.StdEncoding.EncodeToString( data )
}

func DecodeBase64( str string ) ( []byte, error ) {
    return base64.StdEncoding.DecodeString( str )
}