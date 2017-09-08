export default class Headers {
  _headers = {};

  constructor(init) {
    if (init instanceof Headers) {
      init = init._headers;
    }

    if (typeof init === 'object' && init !== null) {
      for (var k in init) {
        var v = init[k];
        if (!Array.isArray(v)) {
          v = [v];
        }

        v.forEach(e => this.append(k, e));
      }
    }
  }

  append(name, value) {
    const normalisedName = Headers.normaliseName(name);

    if (!Object.hasOwnProperty.call(this._headers, normalisedName)) {
      this._headers[normalisedName] = [];
    }

    this._headers[normalisedName].push(value);
  }

  delete(name) {
    delete this._headers[Headers.normaliseName(name)];
  }

  get(name) {
    const normalisedName = Headers.normaliseName(name);

    if (this._headers[normalisedName]) {
      return this._headers[normalisedName][0];
    }
  }

  getAll(name) {
    return this._headers[Headers.normaliseName(name)] || [];
  }

  has(name) {
    const normalisedName = Headers.normaliseName(name);

    return Array.isArray(this._headers[normalisedName]);
  }

  set(name, value) {
    const normalisedName = Headers.normaliseName(name);

    this._headers[normalisedName] = [value];
  }

  static normaliseName(name) {
    return name.toLowerCase();
  }
}
