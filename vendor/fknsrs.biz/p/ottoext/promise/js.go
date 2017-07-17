package promise

const src = `'use strict';

/**
 * @constructor
 */
function Promise(resolver) {
  if (typeof resolver !== 'function' && typeof resolver !== 'undefined') {
    throw new TypeError();
  }

  if (typeof this !== 'object' || (this && this.then)) {
    throw new TypeError();
  }

  var self = this;

  // states
  // 0: pending
  // 1: resolving
  // 2: rejecting
  // 3: resolved
  // 4: rejected
  var state = 0;
  var val = 0;
  var next = [];
  var fn = null;
  var er = null;

  this.promise = this;

  this.resolve = function resolve(v) {
    fn = self.fn;
    er = self.er;

    if (!state) {
      val = v;
      state = 1;

      setImmediate(fire);
    }

    return self;
  };

  this.reject = function reject(v) {
    fn = self.fn;
    er = self.er;

    if (!state) {
      val = v;
      state = 2;

      setImmediate(fire);
    }

    return self;
  };

  this._p = 1;

  this.then = function then(_fn, _er) {
    if (!(this._p === 1)) {
      throw new TypeError();
    }

    var p = new Promise();

    p.fn = _fn;
    p.er = _er;

    switch (state) {
    case 3:
      p.resolve(val);
      break;
    case 4:
      p.reject(val);
      break;
    default:
      next.push(p);
      break;
    }

    return p;
  };

  this.catch = function _catch(_er) {
    return self.then(null, _er);
  };

  var finish = function finish(type) {
    state = type || 4;

    next.map(function(p) {
      state === 3 && p.resolve(val) || p.reject(val);
    });
  };

  try {
    if (typeof resolver === 'function') {
      resolver(this.resolve, this.reject);
    }
  } catch (e) {
    this.reject(e);
  }

  return this;

  // ref: reference to 'then' function
  // cb, ec, cn: successCallback, failureCallback, notThennableCallback
  function thennable (ref, cb, ec, cn) {
    if ((typeof val === 'object' || typeof val === 'function') && typeof ref === 'function') {
      try {
        // cnt protects against abuse calls from spec checker
        var cnt = 0;
        ref.call(val, function(v) {
          if (cnt++) {
            return;
          }

          val = v;

          cb();
        }, function(v) {
          if (cnt++) {
            return;
          }

          val = v;

          ec();
        })
      } catch (e) {
        val = e;

        ec();
      }
    } else {
      cn();
    }
  }

  function fire() {
    // check if it's a thenable
    var ref;

    try {
      ref = val && val.then;
    } catch (e) {
      val = e;
      state = 2;

      return fire();
    }

    thennable(ref, function() {
      state = 1;

      fire();
    }, function() {
      state = 2;

      fire();
    }, function() {
      try {
        if (state === 1 && typeof fn === 'function') {
          val = fn(val);
        } else if (state === 2 && typeof er === 'function') {
          val = er(val);

          state = 1;
        }
      } catch (e) {
        val = e;

        return finish();
      }

      if (val === self) {
        val = TypeError();

        finish();
      } else {
        thennable(ref, function() {
          finish(3);
        }, finish, function() {
          finish(state === 1 && 3);
        });
      }
    });
  }
}

Promise.resolve = function resolve(value) {
  if (!(this._p === 1)) {
    throw new TypeError();
  }

  if (value instanceof Promise) {
    return value;
  }

  return new Promise(function(resolve) {
    resolve(value);
  });
};

Promise.reject = function reject(value) {
  if (!(this._p === 1)) {
    throw new TypeError();
  }

  return new Promise(function(resolve, reject) {
    reject(value);
  });
};

Promise.all = function all(arr) {
  if (!(this._p === 1)) {
    throw new TypeError();
  }

  if (!(arr instanceof Array)) {
    return Promise.reject(TypeError());
  }

  var p = new Promise();

  function done(e, v) {
    if (v) {
      return p.resolve(v);
    }

    if (e) {
      return p.reject(e);
    }

    var unresolved = arr.reduce(function(cnt, v) {
      if (v && v.then) {
        return cnt + 1;
      }

      return cnt;
    }, 0);

    if (unresolved === 0) {
      p.resolve(arr);
    }

    arr.map(function(v, i) {
      if (v && v.then) {
        v.then(function(r) {
          arr[i] = r;
          done();
          return r;
        }, done);
      }
    });
  }

  done();

  return p;
}

Promise.race = function race(arr) {
  if (!(this._p === 1)) {
    throw TypeError();
  }

  if (!(arr instanceof Array)) {
    return Promise.reject(TypeError());
  }

  if (arr.length === 0) {
    return new Promise();
  }

  var p = new Promise();

  function done(e, v) {
    if (v) {
      return p.resolve(v);
    }

    if (e) {
      return p.reject(e);
    }

    var unresolved = arr.reduce(function(cnt, v) {
      if (v && v.then) {
        return cnt + 1;
      }

      return cnt;
    }, 0);

    if (unresolved === 0) {
      p.resolve(arr);
    }

    arr.map(function(v, i) {
      if (v && v.then) {
        v.then(function(r) {
          done(null, r);
        }, done);
      }
    });
  }

  done();

  return p;
}

Promise._p = 1;
`
