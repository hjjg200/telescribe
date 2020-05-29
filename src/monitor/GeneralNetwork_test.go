package monitor

import (
    "encoding/json"
    "testing"
    "time"
    "os"
)

func TestNetwork(t *testing.T) {

    enc := json.NewEncoder(os.Stderr)
    enc.SetIndent("", "  ")
    
    for {
        enc.Encode(GetNetworkIn())
        time.Sleep(3 * time.Second)
    }

}