# Configure websocket transport

Websocket is currently the only Waku transport supported by browser nodes using [js-waku](https://github.com/status-im/js-waku).
Setting up websocket enables your node to directly serve browser peers.

A valid certificate is necessary to serve browser nodes,
you can use [`letsencrypt`](https://letsencrypt.org/):

```shell
sudo letsencrypt -d <your.domain.name>
```

You will need the `privkey.pem` and `fullchain.pem` files.

To enable secure websocket, pass the generated files to `waku`:
Note, the default port for websocket is 8000.

```shell
waku --websocket-secure-support=true --websocket-secure-key-path="<letsencrypt cert dir>/privkey.pem" --websocket-secure-cert-path="<letsencrypt cert dir>/fullchain.pem"
```

## Self-signed certificates

Self-signed certificates are not recommended for production setups because:

- Browsers do not accept self-signed certificates
- Browsers do not display an error when rejecting a certificate for websocket.

However, they can be used for local testing purposes:

```shell
mkdir -p ./ssl_dir/
openssl req -x509 -newkey rsa:4096 -keyout ./ssl_dir/key.pem -out ./ssl_dir/cert.pem -sha256 -nodes
waku --websocket-secure-support=true --websocket-secure-key-path="./ssl_dir/key.pem" --websocket-secure-cert-path="./ssl_dir/cert.pem"
```