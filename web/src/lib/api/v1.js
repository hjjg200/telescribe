
const apiName = "api/v1";

class apiError {
  constructor(uri, status) {
    this.uri = uri;
    this.status = status;
  }
  toString() {
    return `${apiName}: ${this.uri} returned a non-200 status code: ${this.status}`;
  }
}

function formatURI(key, ...args) {
  let suffix = args.length ? `/${args.join("/")}` : "";
  return `/${apiName}/${key}${suffix}`;
}

async function apiFetch(method, typ, key, ...args) {
  let uri = formatURI(key, ...args);
  let rsp = await fetch(uri, {method});
  if(rsp.status !== 200) {
    throw new apiError(uri, rsp.status);
  }
  return typ ? await rsp[typ]() : undefined;
}

export default {

  keyClientInfoMap: "clientInfoMap",
  async getClientInfoMap() {
    return await apiFetch(
      "GET", "json", this.keyClientInfoMap
    );
  },

  keyClientRule: "clientRule",
  async getClientRule(clientId) { 
    return await apiFetch(
      "GET", "json", this.keyClientRule, clientId
    );
  },

  keyClientItemStatus: "clientItemStatus",
  async getClientItemStatus(clientId) { 
    return await apiFetch(
      "GET", "json", this.keyClientItemStatus, clientId
    );
  },

  keyMcfg: "monitorConfig",
  async getMonitorConfig(clientId, monitorKey) {
    return await apiFetch(
      "GET", "json", this.keyMcfg, clientId, monitorKey
    );
  },

  keyMdb: "monitorDataBoundaries",
  async getMonitorDataBoundaries(clientId) {
    return await apiFetch(
      "GET", "text", this.keyMdb, clientId
    );
  },

  keyMdt: "monitorDataTable",
  async getMonitorDataTable(clientId, monitorKey) {
    return await apiFetch(
      "GET", "text", this.keyMdt, clientId, monitorKey
    );
  },
  async deleteMonitorDataTable(clientId, monitorKey) {
    return await apiFetch(
      "DELETE", undefined, this.keyMdt, clientId, monitorKey
    );
  },

  keyWebCfg: "webConfig",
  async getWebConfig() {
    return await apiFetch(
      "GET", "json", this.keyWebCfg
    );
  },

  keyVersion: "version",
  async getVersion() {
    return await apiFetch(
      "GET", "json", this.keyVersion
    );
  }

}