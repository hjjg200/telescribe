# HTTP

This documentation contains the specifications of the HTTP-related elements.


## StaticCache

|Go|
|-|
|`cache`|

## Router

|Go|
|-|
|`hrt`|

## Request

|Go|
|-|
|`hreq`|

## Context

|Go|
|-|
|`hctx`|


## User

|Go|Javascript|
|-|-|
|`husr`|`httpUser`|

A user is an individual account used for basic authentication of HTTP requests.

|Item|Description|
|-|-|
|`name`|The username of the account|
|`password`|The case-insensitive string representation of sha256 hash of the password|
|`permissions`|The array of **HTTP.Permission** for the account|


## Permission Node

|Go|Javascript|
|-|-|
|`pNode`|`permissionNode`|

Permission is used to differentiate between privileged users and general users. Permission is expressed in the case-insensitive string form. When a node contains a dot, you must use double quotation marks to eliminate ambiguity.

|Example|Description|
|-|-|
|`A`|The permission to do A|
|`A.B`|The permission to do A.B|
|`A.*`|The permission to the entire permission under the node A|
|`A."some.node"`|The permission to do some.node under the node A|