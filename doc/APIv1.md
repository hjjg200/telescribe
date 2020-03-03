# API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

## Specification

|Item|Details|
|-|-|
|Base URL|`/api/v1/`|

## clientMap

#### URL

`clientMap`

#### Permission

`api/v1.<method>.clientMap.<fullName>`

#### GET

**200:** Provides the user with a JSON object that contains client list.

```json
{
    "clientMap": {
        "<fullName>": {
            "latestMap": {
                "<mdKey>": {
                    "timestamp": ...,
                    "value": ...,
                    "status": ...
                }
            },
            "configMap": {
                "<mdKey>": {
                    "fatalRange": ...,
                    "warningRange": ...,
                    "format": ...
                }
            }
        }
    }
}
```

**403:** No permission

## monitorDataTableBox

#### URL

`monitorDataTableBox/<fullName>/<mdKey>`

#### Permission

`api/v1.<method>.monitorDataTableBox.<fullName>.<mdKey>`

#### GET

Provides the user with the content of the specified monitorDataTableBox in the CSV form.

#### DELETE

Immediately deletes the specified monitor data's cache.