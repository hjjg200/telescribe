
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
  // Encode twice as net/url unencodes automatically
  args = args.map(arg => encodeURIComponent(encodeURIComponent(arg)));
  let suffix = args.length ? `/${args.join("/")}` : "";
  return `/${apiName}/${key}${suffix}`;
}

async function apiFetch(method, typ, body, key, ...args) {
  let uri = formatURI(key, ...args);
  let info = {method};
  if(body !== undefined) info.body = JSON.stringify(body);
  let rsp = await fetch(uri, info);
  if(rsp.status !== 200) {
    throw new apiError(uri, rsp.status);
  }
  return typ ? await rsp[typ]() : undefined;
}

export default {

  keyClientInfoMap: "clientInfoMap",
  async getClientInfoMap() {
    return await apiFetch(
      "GET", "json", undefined, this.keyClientInfoMap
    );
  },

  keyClientRule: "clientRule",
  async getClientRule(clientId) { 
    return await apiFetch(
      "GET", "json", undefined, this.keyClientRule, clientId
    );
  },

  keyClientItemStatus: "clientItemStatus",
  async getClientItemStatus(clientId) { 
    return await apiFetch(
      "GET", "json", undefined, this.keyClientItemStatus, clientId
    );
  },

  keyMcfg: "monitorConfig",
  async getMonitorConfig(clientId, monitorKey) {
    return await apiFetch(
      "GET", "json", undefined, this.keyMcfg, clientId, monitorKey
    );
  },

  keyMdb: "monitorDataBoundaries",
  async getMonitorDataBoundaries(clientId) {
    return await apiFetch(
      "GET", "text", undefined, this.keyMdb, clientId
    );
  },

  keyMdc: "monitorDataCsv",
  async getMonitorDataCsv(clientId, monitorKey, filter) {
    return await apiFetch(
      "POST", "text", filter, this.keyMdc, clientId, monitorKey
    );
  },

  keyMdt: "monitorDataTable",
  async getMonitorDataTable(clientId, monitorKey) {
    return await apiFetch(
      "GET", "text", undefined, this.keyMdt, clientId, monitorKey
    );
  },
  async deleteMonitorDataTable(clientId, monitorKey) {
    return await apiFetch(
      "DELETE", undefined, undefined, this.keyMdt, clientId, monitorKey
    );
  },

  keyWebCfg: "webConfig",
  async getWebConfig() {
    return await apiFetch(
      "GET", "json", undefined, this.keyWebCfg
    );
  },

  keyVersion: "version",
  async getVersion() {
    return await apiFetch(
      "GET", "json", undefined, this.keyVersion
    );
  }

}