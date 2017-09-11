const Headers = require('./headers');

export default class Response {
  bodyUsed = true;

  _body = null;

  constructor(body, {status=200, statusText='OK', headers={}}={}) {
    this.headers = new Headers(headers);
    this.ok = status >= 200 && status < 300;
    this.status = status;
    this.statusText = statusText;
    this.type = this.headers.get('content-type');
  }

  text() {
    return new Promise(resolve => resolve(this._body));
  }

  json() {
    return this.text().then(d => JSON.parse(d));
  }
}
