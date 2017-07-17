const Headers = require('./headers');

export default class Request {
  constructor(input, {method, headers, redirect, body}={}) {
    this.method = 'GET';
    this.headers = new Headers({});
    this.redirect = 'manual';
    this.body = null;

    if (input instanceof Request) {
      this.url = input.url;
      this.method = input.method;
      this.headers = new Headers(input.headers);
      this.redirect = input.redirect;
    } else {
      this.url = input;
    }

    if (method) {
      this.method = method;
    }

    if (headers) {
      this.headers = new Headers(headers);
    }

    if (redirect) {
      this.redirect = redirect;
    }

    if (body) {
      this.body = body;
    }
  }
}