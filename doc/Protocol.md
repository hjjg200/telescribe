# Protocol

This documentation contains the specifications for protocols. The below articles include a table of the origins, rounds, and details for protocols; the origin specifies the originator for that round, and the round specifies a certain part of that protocol and when the round index is not just a number, it means it initiates and switches to a new protocol; i.e., the round index of a round named "2a Version Mismatch" is "2a" and therefore, it means it starts a new protocol and it is now exempt from the prior protocol as it switched to the initiated protocol.

## Hello

Hello is a procedure where the server and the client create shared secret and the server provides the client with configuration and such.

|Origin|Round|Details|
|-|-|-|
|Client|1 Hello Server|Gives its version and alias|
|Server|2 Hello Client|Gives config for the client|
|Server|2a Version Mismatch|When version does not match, initiates **Version Mismatch**|
|Server|2b Not Whitelisted|When the client is not whitelisted, initiates **Not Whitelisted**|
|Client|3 Terminate|Terminate connection|


## Version Mismatch

Version Mismatch is a procedure where the version of the client does not match with that of the server. The server here provides the client with its executable's content.

|Origin|Round|Details|
|-|-|-|
|Server|1 Version Mismatch|Gives **Executable Bytes**|
|Client|2 Terminate|Terminate connection|


## Monitor Record

Monitor Record is a procedure where the client sends the monitored items to the server.

|Origin|Round|Details|
|-|-|-|
|Client|1 Monitor Record|Gives **Version, Config Version, Alias, Timestamp, Value Map, and Per**|
|Server|2 OK|Ok|
|Server|2a Version Mismatch|When version does not match, initiates **Version Mismatch**|
|Server|2b Reconfigure|When client config version does not match, initiates **Reconfigure**|
|Server|2c Not Whitelisted|When the client is not whitelisted, initiates **Not Whitelisted**|
|Client|3 Terminate|Terminate connection|


## Reconfigure

Reconfigure is a procedure where the server notifies the client that the client config for that client is updated and thus the client needs to reconfigure.

|Origin|Round|Details|
|-|-|-|
|Server|1 Reconfigure|Gives **Config in JSON and Config Version**|
|Client|2 Terminate|Terminate connection|


## Not Whitelisted

Not Whitelisted is a procedure where the server notifies the client that the client is not on the whitelist.

|Origin|Round|Details|
|-|-|-|
|Server|1 Not Whitelisted|Notifies the client that it is not whitelisted|
|Client|2 Terminate|Terminate connection|