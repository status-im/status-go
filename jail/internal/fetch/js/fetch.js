const Request = require('./request');
const Response = require('./response');

export default function fetch(input, init) {
  const req = new Request(input, init);
  const res = new Response();

  return new Promise((resolve, reject) => {
    return __private__fetch_execute(req, res, err => {
      if (err) {
        return reject(err);
      }

      return resolve(res);
    });
  });
}
