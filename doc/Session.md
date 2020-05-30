# Session

This documentation contains the specifications for sessions. It is about the underlying protocols that are used for the entire communications between clients and servers. It is the most crucial part of Telescribe not only because it is the core of communications but because it is also responsible for the integrity and security of transferred data.

## Long-lived Items

The below items are stored as files for future identifications. These files must not be modified and so they are created with 600 permission.

### Known Hosts

Known hosts file contains the public keys that were identified and thus considered authentic.

### Authentication Private Key

## Info

Session info is a structure that contains the basic information about sessions.

|Item|Description|
|-|-|
|ID|Session ID is a unique ID that is used to identify sessions|
|Ephemeral Private Key|An ephemeral private key that is used for master secret creation. It is newly created for each session|
|Ephemeral Public Key|The public key for the ephemeral private key|
|Ephemeral Master Secret|The master secret that is shared between two parties. It is simultaneously created using the keys of both parties|
|Third Party Public Key|The public key of the other party|
|Expiry|The deadline by which the session is considered expired|

## Session

