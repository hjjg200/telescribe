# Telescribe

![](https://img.shields.io/badge/-mini%20project-orange) ![](https://img.shields.io/badge/created-‘19%20Sep%2016-9cf)

Telescribe is standalone application that helps remotely monitor client machines.


## Pre-beta

This project is currently in the pre-beta stage. There is no backward compatibility guarantee for the pre-beta stage; the versions of the beta and stable stage highly likely will not be compatible with the pre-beta versions.

The development will be mainly done in the `develop` branch and only the successful and major commits will make it to the `main` branch.


## Prerequisites

The build process requires the following:

* npm
* go
* https://github.com/hjjg200/go-act
* https://github.com/hjjg200/go-together
* https://github.com/hjjg200/go-jsoncfg


## Install

The install procedure is as follows:

1. Run `build.sh` which will produce necessary things in `bin` directory.
1. In the bin folder, do `./telescribe -server` to create config files.
1. Exit the app and modify the configuration to your taste.
1. Distribute the binary file, `telescribe`, to client machines.
1. You can start the application on your client machines by: 
```shell
./telescribe -host <IP_ADDRESS/DOMAIN> \
  -alias <ALIAS_NAME|DEFAULT:default> \
  -port <TARGET_PORT|DEFAULT:1226>
```

Post-installation process:

1. You can simply edit the config files on the server machine in order to make changes to the configuration.


## Trial and Error

I started this personal project to learn about servers, monitoring, secure protocol, and web framework. These are the notes I personally made during the entire development process.

### Secure Protocol

#### ~*Hybrid Encryption*~

Hybrid encryption is currently deprecated.

* Every packet was encrypted using AES 256 GCM with a new key, and that key was encrypted using RSA public keys.
* This was done because of the content size limit of RSA encryption.

#### *Master Secret*

* A new master secret for each session was created at the handshake process using ECC P256.
* The master secret, then, was used for AES 256 GCM encryption.
* Every packet includes a nonce and AES-encrypted data.


### Web Framework

#### ~*HTML/JS/SCSS*~

In the early stage, no web framework was used.

#### *Vue.js*

Vue.js was chosen as the web framework to go due to its simplicity.

* SFCs were used for legibility and better structure.


### Chart Library

#### ~*Chartist.js*~

Because of its pro-SVG features, Chartist.js was used to plot graphs for data.

#### *D3.js*

As the data amount increased Chartist.js could not keep up with its performance, so D3.jss came in.

* Lower level APIs for graph plotting
* Better performance at high data count


### Data File Encoding

#### ~*Gob*~

Package gob was used to encode and store monitored data files.

#### *Custom Binary*

As gob encoding took too much time to encode and decode data files, just simple binary format was used to encode and decode files.


### Data via Web API

Data are transmitted via web API in csv format which D3.js can understand.

#### ~*Data Decimation*~

* [Largest Triangle Three Buckets](https://github.com/sveinn-steinarsson/flot-downsample) was used to decimate data, in order to decrease the size and stress for the web user interface.
* The problem was that the more were the data, the more abstract became the graph.

#### *Aggregate*

* Aggregation was the chosen method for reducing the performance stress.
* Also, the graph web component was modified to only fetch the visible part of data; previously, the web UI fetched the entire data when plotting.


### Disk Monitoring

#### ~*`df` Output*~

* Parsed raw `df` output to monitor device size and device usage.
* Used `-kP` flags in order to get consistent outputs from `df` across distributions.

#### *`statvfs`*

* In the `df` source code, I found it uses `statvfs` for device information
* Used `cgo` to use the function directly


## Introduction

Telescribe is a standalone app that can act either as a server or a client, which is used for monitoring machines. When it acts as a client, it monitors the machine that it is on and sends the data to its designated server. And for the configuration of each client is all stored in the server and is given to each client at handshake, client machines have to have nothing but the executable file.

And when it acts as a server, it handles connections from telescribe clients and general http clients, handling both connections on the same port. When you use a browser to connect, you'll see graphs and status of the monitored clients.

Telescribe is designed in such a way that the only thing you have to worry about is the machine that acts as a server; client configurations can be modified on the server and they are to be delivered and applied along with the very next packet; when an update was made to the server executable and the version does not match with that of a client, the server will give the client its executable and the client will update itself and restart. And in order to prevent any MITM attack that may coorupt the configuration or the executable file, the server has its private key to sign the data. Clients, therefore, have its known hosts list that contain the public key fingerprints of the servers they have identified and thus consider authentic.


## Web Design

The exterior-wise design is mainly done on the following Figma document:

> [https://www.figma.com/file/jVXjr7BLJLdOHWe2TSPuZW/Telescribe](https://www.figma.com/file/jVXjr7BLJLdOHWe2TSPuZW/Telescribe)

Note that, however, the above document is not a faithful representation of the web design; rather, it must be taken as a prototype for it.


## Stable Release Checklist

1. Project requirements satisfied
1. Maintainable
1. Scalable
1. Well-documented
1. Support for non-procfs systems


## TODO (beta)

1. IPv6 support
1. ~~Config validators~~
1. Custom executables in client configs, which are sent from the server to clients' machines for custom metrics
1. Web anchors(#) or queries for fullName, timestamp, and selected items
1. Client config mixins
1. Realtime graph updating using WebSockets
1. Cache recently viewed data aggregates


## TODO (pre-beta)

1. ~~Data aggregates~~
1. ~~Deprecate gob encoding and fully implement byte-level encoding~~
1. Proper HTTPS support
1. Protocol documentation
1. Monitor documentation
1. Compatibility test for Debian, CentOS(Red Hat), Fedora, Ubuntu, and Mint Linux using LightSail


## TODO (alpha)

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
1. ~~Per-process monitoring~~

## External Libraries
- Vue.js
- D3.js
- Moment.js
