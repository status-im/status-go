/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};
/******/
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/
/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId])
/******/ 			return installedModules[moduleId].exports;
/******/
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			exports: {},
/******/ 			id: moduleId,
/******/ 			loaded: false
/******/ 		};
/******/
/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/
/******/ 		// Flag the module as loaded
/******/ 		module.loaded = true;
/******/
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/
/******/
/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;
/******/
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;
/******/
/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";
/******/
/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(0);
/******/ })
/************************************************************************/
/******/ ([
/* 0 */
/*!******************!*\
  !*** ./index.js ***!
  \******************/
/***/ function(module, exports, __webpack_require__) {

	'use strict';
	
	__webpack_require__(/*! expose?fetch!./fetch */ 1);
	__webpack_require__(/*! expose?Headers!./headers */ 6);
	__webpack_require__(/*! expose?Request!./request */ 7);
	__webpack_require__(/*! expose?Response!./response */ 8);

/***/ },
/* 1 */
/*!******************************************!*\
  !*** ./~/expose-loader?fetch!./fetch.js ***!
  \******************************************/
/***/ function(module, exports, __webpack_require__) {

	/* WEBPACK VAR INJECTION */(function(global) {module.exports = global["fetch"] = __webpack_require__(/*! -!./~/babel-loader?stage=0!./fetch.js */ 2);
	/* WEBPACK VAR INJECTION */}.call(exports, (function() { return this; }())))

/***/ },
/* 2 */
/*!*******************************************!*\
  !*** ./~/babel-loader?stage=0!./fetch.js ***!
  \*******************************************/
/***/ function(module, exports, __webpack_require__) {

	'use strict';
	
	Object.defineProperty(exports, '__esModule', {
	  value: true
	});
	exports['default'] = fetch;
	var Request = __webpack_require__(/*! ./request */ 3);
	var Response = __webpack_require__(/*! ./response */ 5);
	
	function fetch(input, init) {
	  var req = new Request(input, init);
	  var res = new Response();
	
	  return new Promise(function (resolve, reject) {
	    return __private__fetch_execute(req, res, function (err) {
	      if (err) {
	        return reject(err);
	      }
	
	      return resolve(res);
	    });
	  });
	}
	
	module.exports = exports['default'];

/***/ },
/* 3 */
/*!*********************************************!*\
  !*** ./~/babel-loader?stage=0!./request.js ***!
  \*********************************************/
/***/ function(module, exports, __webpack_require__) {

	'use strict';
	
	Object.defineProperty(exports, '__esModule', {
	  value: true
	});
	
	function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError('Cannot call a class as a function'); } }
	
	var Headers = __webpack_require__(/*! ./headers */ 4);
	
	var Request = function Request(input) {
	  var _ref = arguments.length <= 1 || arguments[1] === undefined ? {} : arguments[1];
	
	  var method = _ref.method;
	  var headers = _ref.headers;
	  var redirect = _ref.redirect;
	  var body = _ref.body;
	
	  _classCallCheck(this, Request);
	
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
	};
	
	exports['default'] = Request;
	module.exports = exports['default'];

/***/ },
/* 4 */
/*!*********************************************!*\
  !*** ./~/babel-loader?stage=0!./headers.js ***!
  \*********************************************/
/***/ function(module, exports) {

	'use strict';
	
	Object.defineProperty(exports, '__esModule', {
	  value: true
	});
	
	var _createClass = (function () { function defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ('value' in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } } return function (Constructor, protoProps, staticProps) { if (protoProps) defineProperties(Constructor.prototype, protoProps); if (staticProps) defineProperties(Constructor, staticProps); return Constructor; }; })();
	
	function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError('Cannot call a class as a function'); } }
	
	var Headers = (function () {
	  function Headers(init) {
	    var _this = this;
	
	    _classCallCheck(this, Headers);
	
	    this._headers = {};
	
	    if (init instanceof Headers) {
	      init = init._headers;
	    }
	
	    if (typeof init === 'object' && init !== null) {
	      for (var k in init) {
	        var v = init[k];
	        if (!Array.isArray(v)) {
	          v = [v];
	        }
	
	        v.forEach(function (e) {
	          return _this.append(k, e);
	        });
	      }
	    }
	  }
	
	  _createClass(Headers, [{
	    key: 'append',
	    value: function append(name, value) {
	      var normalisedName = Headers.normaliseName(name);
	
	      if (!Object.hasOwnProperty.call(this._headers, normalisedName)) {
	        this._headers[normalisedName] = [];
	      }
	
	      this._headers[normalisedName].push(value);
	    }
	  }, {
	    key: 'delete',
	    value: function _delete(name) {
	      delete this._headers[Headers.normaliseName(name)];
	    }
	  }, {
	    key: 'get',
	    value: function get(name) {
	      var normalisedName = Headers.normaliseName(name);
	
	      if (this._headers[normalisedName]) {
	        return this._headers[normalisedName][0];
	      }
	    }
	  }, {
	    key: 'getAll',
	    value: function getAll(name) {
	      return this._headers[Headers.normaliseName(name)] || [];
	    }
	  }, {
	    key: 'has',
	    value: function has(name) {
	      var normalisedName = Headers.normaliseName(name);
	
	      return Array.isArray(this._headers[normalisedName]);
	    }
	  }, {
	    key: 'set',
	    value: function set(name, value) {
	      var normalisedName = Headers.normaliseName(name);
	
	      this._headers[normalisedName] = [value];
	    }
	  }], [{
	    key: 'normaliseName',
	    value: function normaliseName(name) {
	      return name.toLowerCase();
	    }
	  }]);
	
	  return Headers;
	})();
	
	exports['default'] = Headers;
	module.exports = exports['default'];

/***/ },
/* 5 */
/*!**********************************************!*\
  !*** ./~/babel-loader?stage=0!./response.js ***!
  \**********************************************/
/***/ function(module, exports, __webpack_require__) {

	'use strict';
	
	Object.defineProperty(exports, '__esModule', {
	  value: true
	});
	
	var _createClass = (function () { function defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ('value' in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } } return function (Constructor, protoProps, staticProps) { if (protoProps) defineProperties(Constructor.prototype, protoProps); if (staticProps) defineProperties(Constructor, staticProps); return Constructor; }; })();
	
	function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError('Cannot call a class as a function'); } }
	
	var Headers = __webpack_require__(/*! ./headers */ 4);
	
	var Response = (function () {
	  function Response(body) {
	    var _ref = arguments.length <= 1 || arguments[1] === undefined ? {} : arguments[1];
	
	    var _ref$status = _ref.status;
	    var status = _ref$status === undefined ? 200 : _ref$status;
	    var _ref$statusText = _ref.statusText;
	    var statusText = _ref$statusText === undefined ? 'OK' : _ref$statusText;
	    var _ref$headers = _ref.headers;
	    var headers = _ref$headers === undefined ? {} : _ref$headers;
	
	    _classCallCheck(this, Response);
	
	    this.bodyUsed = true;
	    this._body = null;
	
	    this.headers = new Headers(headers);
	    this.ok = status >= 200 && status < 300;
	    this.status = status;
	    this.statusText = statusText;
	    this.type = this.headers.get('content-type');
	  }
	
	  _createClass(Response, [{
	    key: 'text',
	    value: function text() {
	      var _this = this;
	
	      return new Promise(function (resolve) {
	        return resolve(_this._body);
	      });
	    }
	  }, {
	    key: 'json',
	    value: function json() {
	      return this.text().then(function (d) {
	        return JSON.parse(d);
	      });
	    }
	  }]);
	
	  return Response;
	})();
	
	exports['default'] = Response;
	module.exports = exports['default'];

/***/ },
/* 6 */
/*!**********************************************!*\
  !*** ./~/expose-loader?Headers!./headers.js ***!
  \**********************************************/
/***/ function(module, exports, __webpack_require__) {

	/* WEBPACK VAR INJECTION */(function(global) {module.exports = global["Headers"] = __webpack_require__(/*! -!./~/babel-loader?stage=0!./headers.js */ 4);
	/* WEBPACK VAR INJECTION */}.call(exports, (function() { return this; }())))

/***/ },
/* 7 */
/*!**********************************************!*\
  !*** ./~/expose-loader?Request!./request.js ***!
  \**********************************************/
/***/ function(module, exports, __webpack_require__) {

	/* WEBPACK VAR INJECTION */(function(global) {module.exports = global["Request"] = __webpack_require__(/*! -!./~/babel-loader?stage=0!./request.js */ 3);
	/* WEBPACK VAR INJECTION */}.call(exports, (function() { return this; }())))

/***/ },
/* 8 */
/*!************************************************!*\
  !*** ./~/expose-loader?Response!./response.js ***!
  \************************************************/
/***/ function(module, exports, __webpack_require__) {

	/* WEBPACK VAR INJECTION */(function(global) {module.exports = global["Response"] = __webpack_require__(/*! -!./~/babel-loader?stage=0!./response.js */ 5);
	/* WEBPACK VAR INJECTION */}.call(exports, (function() { return this; }())))

/***/ }
/******/ ]);
//# sourceMappingURL=bundle.js.map