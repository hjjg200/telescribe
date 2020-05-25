# Telescribe

![](https://img.shields.io/badge/-mini%20project-orange) ![](https://img.shields.io/badge/created-‘19%20Sep%2016-9cf)

Telescribe is standalone application that helps remotely monitor client machines.


## Install

Note that this project is currently in **the alpha stage**; things are liable to change and there is no guarantee of backward compatibility in the alpha stage, which means that using data files or assets created by one version in the alpha stage may not be compatible with another version.

The install procedure is as follows:

1. Run `build.sh` which will produce necessary things in `bin` directory.
1. In the bin folder, do `./telescribe -server` to create config files.
1. Modify the configuration to your taste.
1. Distribute the binary file `telescribe` to client machines.
1. You can start the application on client machines by running `./telescribe -host <IP_ADDRESS> -alias <ALIAS_NAME|DEFAULT:default> -port <TARGET_PORT|DEFAULT:1226>`

Post-installation process:

1. You can simply edit the config files on the server machine in order to make changes to the configuration.


## Introduction

Telescribe is a standalone app that acts as both a server and a client, which is used for monitoring server machines. When it acts as a client, it monitors the machine that it is on and sends the data to its designated server. And for the configuration of each client is all stored in the server and is given to each client at handshake, each client machine has to have nothing but the executable file.

And when it acts as a server, it handles connections from telescribe clients and general http clients, handling both connections on the same port. When you use a browser to connect, you'll see graphs and status of the monitored clients.

Telescribe is designed in such a way that the only thing you have to worry about is the machine that acts as a server; client configurations can be modified on the server and they are to be delivered and applied at the next handshake; when an update was made to the server executable and the version does not match with that of a client, the server will give the client its executable and the client will update itself and restart, provided the client is configured to restart on its own. And in order to prevent any MITM attack that may modify the configuration or the executable file, the server has its private key to sign the data. Clients, therefore, have its known hosts list that contain the public key fingerprints of the servers they have identified and thus consider authentic.


## Components

- **Data Decimation:** For compressing all the monitoring data into a set of the most relevant and concise data
- **Elliptic Curve:** App uses elliptic curve to simultaenously create master secret for symmertic encryption.
- **Symmetric Encryption:** App uses AES 256 GCM to encrypt and decrypt packets. Each encrypted packet includes a nonce and encrypted data.
- **Port Forwarding:** HTTP runs on a random port that is only bound to 127.0.0.1. The main listener recognizes the protocol for each connection and it forwards to HTTP if it is an HTTP request and calls Telescribe methods if it is a telescribe request.
- **Monitoring:** A telescribe client uses files such as `/proc/stat` and `/proc/meminfo` to analyze the current status of the machine
- **Private Key Signing:** App uses ECDSA P256 to sign and verify packets from clients and servers. A telescribe server has its own private key to sign data whose integrity is crucial. And the client stores the public keys of the servers and use them to verify the servers and the data they give.
- **Packet Packing:** Each packet has a record header and several varints to hint the app how many bytes to read.
- **Gob Encoding:** To serialize the monitored data and cache them as files.
- **~~Hybrid Encryption:~~**(removed) Due to the limit of data length of RSA encryption, it first encrypts data with a key that is randomly generated at every instance using AES. Secondly, it encrypts the AES key with the given RSA public key.


## Web Design

The exterior-wise design is mainly done on the following Figma document:

> [https://www.figma.com/file/jVXjr7BLJLdOHWe2TSPuZW/Telescribe](https://www.figma.com/file/jVXjr7BLJLdOHWe2TSPuZW/Telescribe)

Note that, however, the above document is not a faithful representation of the web design; rather, it must be taken as a prototype for it.


## Stable Release Checklist

1. Project requirements satisfied
1. Maintainable
1. Scalable
1. Well-documented


## TODO (beta)

1. IPv6 support
1. Config validators
1. Custom executables in client configs, which are sent from the server to clients' machines for custom metrics
1. Web anchors(#) or queries for fullName, timestamp, and selected items
1. Client config mixins
1. Realtime graph updating using WebSockets


## TODO (alpha)

Telescribe is currently at alpha stage. When all of the followings get done, it will be its beta stage.

1. ~~Better handling of responses and requests~~
1. ~~Elliptic curve encryption~~
1. ~~Host-to-alias-and-role instead of current host-to-role to prevent redundant configs~~
1. ~~Vue.js~~
1. ~~Allow multi clients from same host with host-to-alias-and-role map~~
1. ~~Combining all the data into one single graph and letting users to select which data to view~~
1. ~~Vibrant colors for graph legend~~
1. ~~Fix auto update procedure~~
1. ~~Run client as daemon and spawn sub process so as to auto update would not terminate the app~~
1. ~~I/O monitoring~~
1. ~~Disk monitoring~~
1. ~~Network monitoring~~
1. ~~D3.js~~
1. ~~Detect config change in server and notify the client~~
1. ~~RESTful webhook for fatal status of clients~~
1. ~~Allow users to select time frame of the shown data in web page~~
1. ~~Make another div for tooltip~~
1. ~~Proper handling of mouse events for mobile devices~~
1. ~~Window resize event handler~~
1. ~~Restore the scrollLeft and the hand location when changing duration~~
1. ~~JS overhaul~~
1. ~~Shorten monitor keys~~
1. ~~Compact view~~
1. ~~Big data to csv rather than json~~
1. ~~App.vue implementation~~
1. ~~Intuitive type names~~
1. ~~Roles as tags: "bar": "minecraft-server cpu memory"~~
1. ~~Various http users with different permissions~~
1. ~~Prevent the server from being shutdown when it is flushing caches: use go-together and signal waiting~~
1. ~~Web: Custom number format~~
1. ~~Log files like latest.log, 20191210.1.log.gz...~~
1. ~~Log file separation: access, events~~
1. ~~I/O wait monitoring~~
1. Protocol documentation
1. Monitor documentation
1. Per-process monitoring
1. Overall overhaul
1. Compatibility test for Debian, CentOS(Red Hat), Fedora, Ubuntu, Mint Linux, macOS(maybe) using LightSail

## External Libraries
- Vue.js
- D3.js
- Moment.js
