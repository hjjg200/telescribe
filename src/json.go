package main

import (
    "encoding/json"
)

func encodeJson( v interface{} ) ( []byte, error ) {
    return json.Marshal( v )
}

func decodeJson( j []byte, v interface{} ) error {
    return json.Unmarshal( j, v )
}

func UnmarshalJsonResponse( j []byte ) ( map[string] interface{}, error ) {
    rsp := make( map[string] interface{} )
    return rsp, decodeJson( j, &rsp )
}

func MarshalJsonResponse( name string, args map[string] interface{} ) ( []byte, error ) {

    rsp := make( map[string] interface{} )
    
    rsp["name"] = name
    for k, v := range args {
        rsp[k] = v
    }

    return encodeJson( rsp )

}