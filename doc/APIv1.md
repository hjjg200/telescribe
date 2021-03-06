# API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

## clientInfoMap

#### URL

`/api/v1/clientInfoMap`

#### Permission

`api/v1.get.<Client.ID>`

#### Sub-Permission

* `clientInfoMap.host`: Whether the **Client.Info** object should include host address

#### GET

* **200:** Provides the user with a JSON object that contains a **Client.ID** to **Client.Info** map. The user will only have access to the clients he or she has access to, as defined in their permissions.

```text
{
    "clientInfoMap": {
        "<Client.ID>": {
            Client.Info
        }
    }
}
```

* **403:** No permission


## clientRule

#### URL

`/api/v1/clientRule/<Client.ID>`

#### Permission

`api/v1.get.<Client.ID>`

#### GET

* **200:** Provides the user with a **Client.Rule** object. Note that rule objects generally hold raw **Monitor.Config**.

```text
{
    "clientRule": {
        Client.Rule
    }
}
```

* **403:** No permission


## clientItemStatus

#### URL

`/api/v1/clientItemStatus/<Client.ID>`

#### Permission

`api/v1.get.<Client.ID>.monitor.<Monitor.Key>`

#### GET

* **200:** Provides the user with a **Monitor.Key** to **Client.ItemStatus** object.

```text
{
    "clientItemStatus": {
        "<Monitor.Key>": {
            Client.ItemStatus
        }
    }
}
```

* **400:** Not found

* **403:** No permission


## monitorConfig

#### URL

`/api/v1/monitorConfig/<Client.ID>/<Monitor.Key>`

#### Permission

`api/v1.get.<Client.ID>.monitor.<Monitor.Key>`

#### GET

* **200:** Provides the user with a **Monitor.Config** for the given **Monitor.Key**.

```text
{
    "monitorConfig": {
        Monitor.Config
    }
}
```

* **400:** Not found

* **403:** No permission


## monitorDataBoundaries

#### URL

`/api/v1/monitorDataBoundaries/<Client.ID>`

#### Permission

`api/v1.get.<Client.ID>`

#### GET

* **200:** Provides the user with a csv table that contains timestamp boundaries of a client.

```text
timestamp
...
...
```

* **403:** No permission


## monitorDataTable

#### URL

`/api/v1/monitorDataTable/<Client.ID>/<Monitor.Key>`

#### Permission

`api/v1.<method>.<Client.ID>.monitor.<Monitor.Key>`

#### GET

* **200:** Provides the user with a csv table that contains timestamps and values.

```text
timestamp,value
...,...
...,...
```

* **400:** Not found

* **403:** No permission

#### DELETE

* **200:** Immediately deletes the specified monitor data's cache.

* **400:** Not found

* **403:** No permission

* **500:** Internal error; most likely an I/O error


## webConfig

#### URL

`/api/v1/webConfig`

#### Permission

`api/v1.get`

#### GET

* **200:** Provides the user with a **Web.Config** object

```text
{
    "webConfig": {
        Web.Config
    }
}
```

* **403:** No permission


## version

#### URL

`/api/v1/version`

#### Permission

`api/v1.get`

#### GET

* **200:** JSON object that contains the version
```text
{
    "version": "telescribe-..."
}
```