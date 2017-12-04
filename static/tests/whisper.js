var chai = require("chai");
var expect = chai.expect;
var assert = chai.assert;
var Web3 = require('web3');

describe.skip('Whisper Tests', function () {
    var node1 = new Web3();
    var node2 = new Web3();
    var web3 = node1;
    node1.setProvider(new web3.providers.HttpProvider('http://localhost:8645'));
    node2.setProvider(new web3.providers.HttpProvider('http://localhost:8745'));

    console.log('Node is expected: wnode-status -datadir app1 wnode -http -httpport 8645');
    console.log('Node is expected: wnode-status -datadir app2 wnode -http -httpport 8745');
    console.log('Node is expected: wnode-status -datadir wnode1 wnode -notify -injectaccounts=false -identity ./static/keys/wnodekey -firebaseauth ./static/keys/firebaseauthkey');

    // some common vars
    var topic1 = '0xdeadbeef'; // each topic 4 bytes, as hex
    var topic2 = '0xbeefdead'; // each topic 4 bytes, as hex
    var topic3 = '0xbebebebe'; // each topic 4 bytes, as hex
    var topic4 = '0xdadadada'; // each topic 4 bytes, as hex
    var identity1 = '0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3';
    var identity2 = '0x0490161b00f2c47542d28c2e8908e77159b1720dccceb6393d7c001850122efc3b1709bcea490fd8f5634ba1a145aa0722d86b9330b0e39a8d493cb981fd459da2';

    // watchFilter makes sure that we halt the filter on first message received
    var watchFilter = function (filter, done) {
        var messageReceived = false;
        filter.watch(function (error, message) {
            if (messageReceived)  return; // avoid double calling
            messageReceived = true; // no need to watch for the filter any more
            filter.stopWatching();
            done(error, message);
        });
    };

    // makeTopic generates random topic (4 bytes, in hex)
    var makeTopic = function () {
        var min = 1;
        var max = Math.pow(16, 8);
        var randInt = Math.floor(Math.random() * (max - min + 1)) + min;
        return web3.toHex(randInt);
    };

    context('shh/5 API verification', function () {
        it('statusd node is running', function () {
            var web3 = new Web3();
            var provider = new web3.providers.HttpProvider('http://localhost:8645');
            var result = provider.send({});
            assert.equal(typeof result, 'object');
        });

        it('shh.version()', function () {
            var version = node1.shh.version();
            assert.equal(version, '0x5', 'Whisper version does not match');
        });

        it('shh.info()', function () {
            var info = node1.shh.info();
            if (info == "") {
                throw new Error('no Whisper info provided')
            }
        });

        context('symmetric key management', function () {
            var keyId = ''; // symmetric key ID (to be populated)
            var keyVal = ''; // symmetric key value (to be populated)

            it('shh.generateSymmetricKey()', function () {
                keyId = node1.shh.generateSymmetricKey();
                assert.lengthOf(keyId, 64, 'invalid keyId length');
            });

            it('shh.getSymmetricKey(keyId)', function () {
                keyVal = node1.shh.getSymmetricKey(keyId);
                assert.lengthOf(keyVal, 66, 'invalid key value length'); // 2 bytes for "0x"
            });

            it('shh.hasSymmetricKey(keyId)', function () {
                expect(node1.shh.hasSymmetricKey(keyId)).to.equal(true);
            });

            it('shh.deleteSymmetricKey(keyId)', function () {
                expect(node1.shh.hasSymmetricKey(keyId)).to.equal(true);
                node1.shh.deleteSymmetricKey(keyId);
                expect(node1.shh.hasSymmetricKey(keyId)).to.equal(false);
            });

            it('shh.addSymmetricKeyDirect(keyVal)', function () {
                keyIdOriginal = keyId;
                keyId = node1.shh.addSymmetricKeyDirect(keyVal);
                assert.notEqual(keyId, keyIdOriginal);
                assert.lengthOf(keyId, 64, 'invalid keyId length');
                expect(node1.shh.hasSymmetricKey(keyId)).to.equal(true);
            });

            it('shh.addSymmetricKeyFromPassword(password)', function () {
                var password = 'foobar';
                var keyId = node1.shh.addSymmetricKeyFromPassword(password);
                var keyVal = node1.shh.getSymmetricKey(keyId);

                assert.lengthOf(keyId, 64, 'invalid keyId length');
                expect(node1.shh.hasSymmetricKey(keyId)).to.equal(true);
                assert.equal(keyVal, '0xa582720d74d463589df14c11538189a1c07778c47e86f70bab7b5ba27e2de3cc');
            });
        });

        context('assymmetric key management', function () {
            var keyId = ''; // to be populated
            var pubKey = ''; // to be populated

            it('shh.newKeyPair()', function () {
                keyId = node1.shh.newKeyPair();
                assert.lengthOf(keyId, 64);
            });

            it('shh.hasKeyPair(id)', function () {
                expect(node1.shh.hasKeyPair(keyId)).to.equal(true);
            });

            it('shh.getPublicKey(id)', function () {
                pubKey = node1.shh.getPublicKey(keyId);
                assert.lengthOf(pubKey, 132);
            });

            it('shh.hasKeyPair(pubKey)', function () {
                expect(node1.shh.hasKeyPair(pubKey)).to.equal(true);
            });

            it('shh.getPrivateKey(id)', function () {
                var prvkey = node1.shh.getPrivateKey(keyId);
                assert.lengthOf(prvkey, 66);
            });

            it('shh.deleteKeyPair(id)', function () {
                expect(node1.shh.hasKeyPair(pubKey)).to.equal(true);
                expect(node1.shh.hasKeyPair(keyId)).to.equal(true);
                node1.shh.deleteKeyPair(keyId);
                expect(node1.shh.hasKeyPair(pubKey)).to.equal(false);
                expect(node1.shh.hasKeyPair(keyId)).to.equal(false);

                // re-create
                keyId = node1.shh.newKeyPair();
                assert.lengthOf(keyId, 64);
                pubKey = node1.shh.getPublicKey(keyId);
                assert.lengthOf(pubKey, 132);
            });

            it('shh.deleteKeyPair(pubKey)', function () {
                expect(node1.shh.hasKeyPair(pubKey)).to.equal(true);
                expect(node1.shh.hasKeyPair(keyId)).to.equal(true);
                node1.shh.deleteKeyPair(pubKey);
                expect(node1.shh.hasKeyPair(pubKey)).to.equal(false);
                expect(node1.shh.hasKeyPair(keyId)).to.equal(false);

                // re-create
                keyId = node1.shh.newKeyPair();
                assert.lengthOf(keyId, 64);
                pubKey = node1.shh.getPublicKey(keyId);
                assert.lengthOf(pubKey, 132);
            });
        });

        context('subscribe and manually get messages', function () {
            // NOTE: you can still use shh.filter to poll for messages automatically, see other examples

            var filterid1 = ''; // sym filter, to be populated
            var filterid2 = ''; // asym filter, to be populated
            var keyId = ''; // symkey, to be populated
            var uniqueTopic = makeTopic();

            var payloadBeforeSymFilter = 'sent before filter was active (symmetric)';
            var payloadAfterSymFilter = 'sent after filter was active (symmetric)';
            var payloadBeforeAsymFilter = 'sent before filter was active (asymmetric)';
            var payloadAfterAsymFilter = 'sent after filter was active (asymmetric)';

            it('shh.subscribe(filterParams) - symmetric filter', function () {
                keyId = node1.shh.generateSymmetricKey();
                assert.lengthOf(keyId, 64);

                // send message, which will be floating around *before* filter is even created
                var message = {
                    type: "sym",
                    key: keyId,
                    topic: uniqueTopic,
                    payload: payloadBeforeSymFilter
                };
                expect(node1.shh.post(message)).to.equal(null);

                // symmetric filter
                filterid1 = node1.shh.subscribe({
                    type: "sym",
                    key: keyId,
                    sig: identity1,
                    topics: [topic1, topic2, uniqueTopic]
                });
                assert.lengthOf(filterid1, 64);
            });

            it('shh.subscribe(filterParams) - asymmetric filter', function () {
                // send message, which will be floating around *before* filter is even created
                var message = {
                    type: "asym",
                    key: identity2,
                    topic: uniqueTopic,
                    payload: payloadBeforeAsymFilter
                };
                expect(node1.shh.post(message)).to.equal(null);

                // asymmetric filter
                filterid2 = node1.shh.subscribe({
                    type: "asym",
                    key: identity2,
                    sig: identity1,
                    topics: [topic1, topic2, uniqueTopic]
                });
                assert.lengthOf(filterid1, 64);
            });

            it('shh.getFloatingMessages(filterID) - symmetric filter', function () {
                // let's try to capture message that was there *before* filter is created
                var messages = node1.shh.getFloatingMessages(filterid1);
                assert.typeOf(messages, 'array');
                assert.lengthOf(messages, 1);
                assert.equal(web3.toAscii(messages[0].payload), payloadBeforeSymFilter);

                // send message, after the filter has been already installed
                var message = {
                    type: "sym",
                    key: keyId,
                    topic: uniqueTopic,
                    payload: payloadAfterSymFilter
                };
                expect(node1.shh.post(message)).to.equal(null);
            });

            it('shh.getFloatingMessages(filterID) - asymmetric filter', function () {
                // let's try to capture message that was there *before* filter is created
                var messages = node1.shh.getFloatingMessages(filterid2);
                assert.typeOf(messages, 'array');
                assert.lengthOf(messages, 1);
                assert.equal(web3.toAscii(messages[0].payload), payloadBeforeAsymFilter);

                // send message, after the filter has been already installed
                var message = {
                    type: "asym",
                    key: identity2,
                    topic: uniqueTopic,
                    payload: payloadAfterAsymFilter
                };
                expect(node1.shh.post(message)).to.equal(null);
            });

            it('shh.getNewSubscriptionMessages(filterID) - symmetric filter', function (done) {
                // allow some time for message to propagate
                setTimeout(function () {
                    // now let's try to capture new messages from our last capture
                    var messages = node1.shh.getNewSubscriptionMessages(filterid1);
                    assert.typeOf(messages, 'array');
                    assert.lengthOf(messages, 1);
                    assert.equal(web3.toAscii(messages[0].payload), payloadAfterSymFilter);

                    // no more messages should be returned
                    messages = node1.shh.getNewSubscriptionMessages(filterid1);
                    assert.typeOf(messages, 'array');
                    assert.lengthOf(messages, 0);

                    done();
                }, 200);
            });

            it('shh.getNewSubscriptionMessages(filterID) - asymmetric filter', function () {
                // allow some time for message to propagate
                setTimeout(function () {
                    // now let's try to capture new messages from our last capture
                    var messages = node1.shh.getNewSubscriptionMessages(filterid2);
                    assert.typeOf(messages, 'array');
                    assert.lengthOf(messages, 1);
                    assert.equal(web3.toAscii(messages[0].payload), payloadAfterAsymFilter);

                    // no more messages should be returned
                    messages = node1.shh.getNewSubscriptionMessages(filterid2);
                    assert.typeOf(messages, 'array');
                    assert.lengthOf(messages, 0);

                    done();
                }, 200);
            });

            it.skip('shh.unsubscribe(filterID)', function () {
                node1.shh.unsubscribe(filterid1);
                node1.shh.unsubscribe(filterid2);
            });
        });
    });

    context('symmetrically encrypted messages send/recieve', function () {
        this.timeout(0);

        var keyId = ''; // symmetric key ID (to be populated)
        var keyVal = ''; // symmetric key value (to be populated)
        var payload = 'here come the dragons';

        it('default test identity is present', function () {
            if (!node1.shh.hasKeyPair(identity1)) {
                throw new Error('identity not found in whisper: ' + identity1);
            }
        });

        it('ensure symkey exists', function () {
            keyId = node1.shh.generateSymmetricKey();
            assert.lengthOf(keyId, 64);
            expect(node1.shh.hasSymmetricKey(keyId)).to.equal(true);
        });

        it('read the generated symkey', function () {
            keyVal = node1.shh.getSymmetricKey(keyId);
            assert.lengthOf(keyVal, 66); // 2 bytes for "0x"
        });

        it('send/receive symmetrically encrypted message', function (done) {
            // start watching for messages
            watchFilter(node1.shh.filter({
                type: "sym",
                key: keyId,
                sig: identity1,
                topics: [topic1, topic2]
            }), function (err, message) {
                done(err);
            });

            // send message
            var message = {
                type: "sym",
                key: keyId,
                sig: identity1,
                topic: topic1,
                payload: web3.fromAscii(payload),
                ttl: 20,
                powTime: 2,
                powTarget: 0.001
            };
            expect(node1.shh.post(message)).to.equal(null);
        });

        it('send the minimal symmetric message possible', function (done) {
            var uniqueTopic = makeTopic();

            // start watching for messages
            watchFilter(node1.shh.filter({
                type: "sym",
                key: keyId,
                topics: [uniqueTopic]
            }), function (err, message) {
                done(err);
            });

            // send message
            var message = {
                type: "sym",
                key: keyId,
                topic: uniqueTopic
            };
            expect(node1.shh.post(message)).to.equal(null);
        });
    });

    context('message travelling from one node to another', function () {
        this.timeout(0);

        var keyId1 = ''; // symmetric key ID on node 1 (to be populated)
        var keyId2 = ''; // symmetric key ID on node 2 (to be populated)

        it('statusd node1 is running', function () {
            var web3 = new Web3();
            var provider = new web3.providers.HttpProvider('http://localhost:8645');
            var result = provider.send({});
            assert.equal(typeof result, 'object');
        });

        it('statusd node2 is running', function () {
            var web3 = new Web3();
            var provider = new web3.providers.HttpProvider('http://localhost:8745');
            var result = provider.send({});
            assert.equal(typeof result, 'object');
        });

        it('test identities injected', function () {
            if (!node1.shh.hasKeyPair(identity1)) {
                throw new Error('identity not found in whisper (node1): ' + identity1);
            }
            if (!node1.shh.hasKeyPair(identity2)) {
                throw new Error('identity not found in whisper (node1): ' + identity2);
            }
            if (!node2.shh.hasKeyPair(identity1)) {
                throw new Error('identity not found in whisper (node2): ' + identity1);
            }
            if (!node2.shh.hasKeyPair(identity2)) {
                throw new Error('identity not found in whisper (node2): ' + identity2);
            }
        });

        it('ensure symkey exists', function () {
            keyId1 = node1.shh.generateSymmetricKey();
            assert.lengthOf(keyId1, 64);
            expect(node1.shh.hasSymmetricKey(keyId1)).to.equal(true);

            // obtain key value
            var keyVal = node1.shh.getSymmetricKey(keyId1);
            assert.lengthOf(keyVal, 66); // 2 bytes of "0x"

            // share the value with the node2
            keyId2 = node2.shh.addSymmetricKeyDirect(keyVal);
            assert.lengthOf(keyId2, 64);
            expect(node2.shh.hasSymmetricKey(keyId2)).to.equal(true);
        });

        it('send symmetrically encrypted, signed message (node1 -> node2)', function (done) {
            var payload = 'send symmetrically encrypted, signed message (node1 -> node2)';
            var topic = makeTopic();
            // start watching for messages
            watchFilter(node2.shh.filter({
                type: "sym",
                sig: identity1,
                key: keyId2,
                topics: [topic]
            }), function (err, message) {
                done(err);
            });

            // send message
            var message = {
                type: "sym",
                sig: identity1,
                key: keyId1,
                topic: topic,
                payload: payload,
                ttl: 20
            };
            expect(node1.shh.post(message)).to.equal(null);
        });

        it('send asymmetrically encrypted, signed message (node1.id1 -> node2.id2)', function (done) {
            var payload = 'send asymmetrically encrypted, signed message (node1.id1 -> node2.id2)';
            var topic = makeTopic();
            // start watching for messages
            watchFilter(node2.shh.filter({
                type: "asym",
                sig: identity1,
                key: identity2
            }), function (err, message) {
                done(err);
            });

            // send message
            var message = {
                type: "asym",
                sig: identity1,
                key: identity2,
                topic: topic,
                payload: payload,
                ttl: 20
            };
            expect(node1.shh.post(message)).to.equal(null);
        });
    });

    context('push notifications', function () {
        this.timeout(5000);
        var discoveryPubKey = '0x040edb0d71a3dbe928e154fcb696ffbda359b153a90efc2b46f0043ce9f5dbe55b77b9328fd841a1db5273758624afadd5b39638d4c35b36b3a96e1a586c1b4c2a';
        var discoverServerTopic = '0x268302f3'; // DISCOVER_NOTIFICATION_SERVER
        var proposeServerTopic = '0x08e3d8c0'; // PROPOSE_NOTIFICATION_SERVER
        var acceptServerTopic = '0x04f7dea6'; // ACCEPT_NOTIFICATION_SERVER
        var ackClientSubscriptionTopic = '0x93dafe28'; // ACK_NOTIFICATION_SERVER_SUBSCRIPTION
        var sendNotificationTopic = '0x69915296'; // SEND_NOTIFICATION
        var newChatSessionTopic = '0x509579a2'; // NEW_CHAT_SESSION
        var ackNewChatSessionTopic = '0xd012aae8'; // ACK_NEW_CHAT_SESSION
        var newDeviceRegistrationTopic = '0x14621a51'; // NEW_DEVICE_REGISTRATION
        var ackDeviceRegistrationTopic = '0x424358d6'; // ACK_DEVICE_REGISTRATION
        var checkClientSessionTopic = '0x8745d931'; // CHECK_CLIENT_SESSION
        var confirmClientSessionTopic = '0xd3202c5f'; // CONFIRM_CLIENT_SESSION
        var dropClientSessionTopic = '0x3a6656bb'; // DROP_CLIENT_SESSION

        // ensures that message had payload (which is HEX-encoded JSON)
        var extractPayload = function (message) {
            expect(message).to.have.property('payload');
            return JSON.parse(web3.toAscii(message.payload));
        };

        var identity1 = ''; // pub key of device 1
        var identity2 = ''; // pub key of device 2
        var chatKeySharingTopic = makeTopic(); // topic used by device1 to send chat key to device 2

        context('prepare devices', function () {
            it('create key pair to be used as main identity on device1', function () {
                var keyId = node1.shh.newKeyPair();
                assert.lengthOf(keyId, 64);

                identity1 = node1.shh.getPublicKey(keyId);
                assert.lengthOf(identity1, 132);

                expect(node1.shh.hasKeyPair(identity1)).to.equal(true);
                expect(node1.shh.hasKeyPair(identity2)).to.equal(false);
            });

            it('create key pair to be used as main identity on device2', function () {
                var keyId = node2.shh.newKeyPair();
                assert.lengthOf(keyId, 64);

                identity2 = node2.shh.getPublicKey(keyId);
                assert.lengthOf(identity1, 132);

                expect(node2.shh.hasKeyPair(identity1)).to.equal(false);
                expect(node2.shh.hasKeyPair(identity2)).to.equal(true);
            });
        });

        context('run device1', function () {
            var serverId = ''; // accepted/selected server id
            var subscriptionKeyId = ''; // symkey provided by server, and used to configure client-server subscription
            var chatKeyId = ''; // symkey provided by server, and shared among clients so that they can trigger notifications
            var appChatId = ''; // chat id that identifies device1-device2 interaction session on RN app level


            it('start discovery by sending discovery request', function () {
                var message = {
                    type: "asym",
                    sig: identity1,
                    key: discoveryPubKey,
                    topic: discoverServerTopic,
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);
            });

            it('watch for server proposals', function (done) {
                watchFilter(node1.shh.filter({
                    type: "asym",
                    sig: discoveryPubKey,
                    key: identity1,
                    topics: [proposeServerTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    serverId = payload.server;

                    done();
                });
            });

            it('client accepts server', function () {
                var message = {
                    type: "asym",
                    sig: identity1,
                    key: discoveryPubKey,
                    topic: acceptServerTopic,
                    payload: '{"server": "' + serverId + '"}',
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);
            });

            it('watch for server ACK response and save provided subscription key', function (done) {
                watchFilter(node1.shh.filter({
                    type: "asym",
                    key: identity1,
                    topics: [ackClientSubscriptionTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');

                    // save subscription key
                    subscriptionKeyId = node1.shh.addSymmetricKeyDirect(payload.key);
                    assert.lengthOf(subscriptionKeyId, 64);
                    expect(node1.shh.hasSymmetricKey(subscriptionKeyId)).to.equal(true);

                    done();
                });
            });

            it('create chat session', function () {
                appChatId = makeTopic(); // globally unique chat id
                var message = {
                    type: "sym",
                    sig: identity1,
                    key: subscriptionKeyId,
                    topic: newChatSessionTopic,
                    payload: '{"chat": "' + appChatId + '"}',
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);
            });

            it('watch for server to respond with chat key', function (done) {
                watchFilter(node1.shh.filter({
                    type: "asym",
                    key: identity1,
                    topics: [ackNewChatSessionTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');

                    // save subscription key
                    chatKeyId = node1.shh.addSymmetricKeyDirect(payload.key);
                    assert.lengthOf(chatKeyId, 64);
                    expect(node1.shh.hasSymmetricKey(chatKeyId)).to.equal(true);

                    done();
                });
            });

            it('register device with a given chat', function (done) {
                // this obtained from https://status-sandbox-c1b34.firebaseapp.com/
                var deviceId = 'ca5pRJc6L8s:APA91bHpYFtpxvXx6uOayGmnNVnktA4PEEZdquCCt3fWR5ldLzSy1A37Tsbzk5Gavlmk1d_fvHRVnK7xPAhFFl-erF7O87DnIEstW6DEyhyiKZYA4dXFh6uy323f9A3uw5hEtT_kQVhT';
                var message = {
                    type: "sym",
                    sig: identity1,
                    key: chatKeyId,
                    topic: newDeviceRegistrationTopic,
                    payload: '{"device": "' + deviceId + '"}',
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);

                // watch for server server ACK
                watchFilter(node1.shh.filter({
                    type: "asym",
                    key: identity1,
                    topics: [ackDeviceRegistrationTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');

                    done();
                });
            });

            it('share chat key, so that another device can send us notifications', function () {
                var chatKey = node1.shh.getSymmetricKey(chatKeyId);
                assert.lengthOf(chatKey, 66);
                var message = {
                    type: "asym",
                    sig: identity1,
                    key: identity2,
                    topic: chatKeySharingTopic,
                    payload: '{"chat": "' + appChatId + '", "key": "' + chatKey + '"}',
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);
            });
        });

        context('run device2', function () {
            var chatKeyId = '';

            it('watch for device1 to send us chat key', function (done) {
                watchFilter(node2.shh.filter({
                    type: "asym",
                    key: identity2,
                    topics: [chatKeySharingTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('chat');
                    expect(payload).to.have.property('key');

                    // persist chat key
                    chatKeyId = node2.shh.addSymmetricKeyDirect(payload.key);
                    assert.lengthOf(chatKeyId, 64);
                    expect(node2.shh.hasSymmetricKey(chatKeyId)).to.equal(true);

                    done();
                });
            });

            it('trigger notification (from device2, on device1)', function () {
                var message = {
                    type: "sym",
                    sig: identity2,
                    key: chatKeyId,
                    topic: sendNotificationTopic,
                    payload: '{' // see https://firebase.google.com/docs/cloud-messaging/http-server-ref
                    + '"notification": {'
                    + '"title": "status.im notification",'
                    + '"body": "Hello this is test notification!",'
                    + '"icon": "https://status.im/img/logo.png",'
                    + '"click_action": "https://status.im"'
                    + '},'
                    + '"to": "{{ ID }}"' // this get replaced by device id your've registered
                    + '}',
                    ttl: 20
                };
                expect(node2.shh.post(message)).to.equal(null);
            });
        });

        context('misc methods and cleanup', function () {

            it('check client session', function (done) {
                // request status
                var message = {
                    type: "asym",
                    sig: identity1,
                    key: discoveryPubKey,
                    topic: checkClientSessionTopic,
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);

                // process server's response
                watchFilter(node1.shh.filter({
                    type: "asym",
                    key: identity1,
                    topics: [confirmClientSessionTopic]
                }), function (err, message) {
                    if (err) return done(err);

                    // process payload
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');

                    done();
                });
            });

            it('remove client session', function () {
                var message = {
                    type: "asym",
                    sig: identity1,
                    key: discoveryPubKey,
                    topic: dropClientSessionTopic,
                    ttl: 20
                };
                expect(node1.shh.post(message)).to.equal(null);
            });
        });
    });
});
