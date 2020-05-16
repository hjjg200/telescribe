
export default class {
  constructor() {
    this.queued = [];
    this.running = false;
  }
  queue(asyncFunc) {
    let $ = this;
    return new Promise(resolve => {
      $.queued.push({asyncFunc, resolve});
      if(!$.running) {
        $.running = true;
        $._run();
      }
    });
  }
  _run() {
    let queued = this.queued.slice(0);
    this.queued = [];

    (async() => {
      for(let i = 0; i < queued.length; i++) {
        let q = queued[i];
        await q.asyncFunc().then(q.resolve);
      }
    })().then(this._post.bind(this));
  }
  _post() {
    if(this.queued.length) {
      this._run();
    } else {
      this.running = false;
    }
  }
}