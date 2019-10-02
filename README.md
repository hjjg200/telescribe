# Telescribe

Telescribe is a standalone app that acts as both a server and a client, which is used for monitoring server machines. When it acts as a client, it monitors the machine that it is on and send the data to its designated server. And for the configuration of each client is all stored in the server and is given to each client at handshake, each server machine has to have nothing but the executable file.

And when it acts as a server, it handles connections from telescribe clients and general http clients, handling both connections on the same port. When you use a browser to connect, you'll see graphs and status of the monitored clients.

Telescribe is designed in such a way that the only thing you have to worry about is the machine that acts as a server; client configurations can be modified on the server and they are to be delivered and applied at the next handshake; when an update was made to the server executable and the version does not match with that of a client, the server will give the client its executable and the client will update itself and restart, provided the client is configured to restart on its own. And in order to prevent any MITM attack that may modify the configuration or the executable file, the server has its private key to sign the data. Clients, therefore, have its known hosts list that contain the public key fingerprints of the servers they have identified and thus consider authentic.

## Components

- **Data Decimation:** For compressing all the monitoring data into a set of the most relevant and concise data
- **Asymmetric Encryption:** App also uses ECC to simultaenously create master secret for symmertic encryption.
- **Symmetric Encryption:** App uses AES 256 GCM to encrypt and decrypt packets. Each encrypted packet includes a nonce and encrypted data.
- **Port Forwarding:** HTTP runs on a random port that is only bound to 127.0.0.1. The main listener recognizes the protocol for each connection and it forwards to HTTP if it is an HTTP request and calls Telescribe methods if it is a relevant request.
- **Monitoring:** A telescribe client uses files such as `/proc/stat` and `/proc/meminfo` to analyze the current status of the machine
- **Private Key Signing:** App uses ECDSA P256 to sign and verify packets from clients and servers. A telescribe server has its own private key to sign data that must not be modified. And the client stores the public keys of the servers and use them to verify the servers and the data they give.
- **Packet Packing:** Each packet has a record header and several varints to hint the app how many bytes to read.
- **Gob Encoding:** To serialize the monitored data and cache them as files.
- **~~Hybrid Encryption:~~**(removed) Due to the limit of data length of RSA encryption, it first encrypts data with a key that is randomly generated at every instance using AES. Secondly, it encrypts the AES key with the given RSA public key.

## TODO

Telescribe is currently at alpha stage. When all of the followings get done, it will be its beta stage.

1. ~~Better handling of responses and requests~~
1. ~~Elliptic curve encryption~~
1. Detect config change in server and notify the client
1. Alias-to-host-and-role instead of current host-to-role to prevent redundant configs
1. Putting static data into binary
1. I/O monitoring
1. Disk monitoring
1. Per-process monitoring
1. Network monitoring
1. JS overhaul
1. (maybe) React.js
1. Numeral.js format for monitor data
1. Combining all the data into one single graph and letting users to select which data to view
1. Custom bash scripts in client configs
1. Overall overhaul

## External Libraries
- Chartist.js
- Chartist Tooltip Plugin
- Numeral.js
- Moment.js