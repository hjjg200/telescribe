
export default class {
  constructor() {
    this.queued = [];
    this.running = false;
  }
  queue(promise) {
    let $ = this;
    return new Promise(resolve => {
      $.queued.push({promise, resolve});
      if(!$.running) {
        $.running = true;
        $._run();
      }
    });
  }
  _run() {
    let queued = this.queued.slice(0);
    this.queued = [];

    Promise.all(queued.map(
      d => d.promise.then(d.resolve)
    )).then(this._post.bind(this));
  }
  _post() {
    if(this.queued.length) {
      this._run();
    } else {
      this.running = false;
    }
  }
}