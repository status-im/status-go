var chai = require("chai");
var expect = chai.expect;
var assert = chai.assert;
var Web3 = require('web3');

describe('Whisper Tests', function () {
    var web3 = new Web3();
    // status peer
    var node1 = new Web3(new Web3.providers.HttpProvider('http://localhost:8645'));
    // status peer
    var node2 = new Web3(new Web3.providers.HttpProvider('http://localhost:8745'));
    // notification service node
    var node3 = new Web3(new Web3.providers.HttpProvider('http://localhost:8845'));
    
    // some common vars
    var powTime = 3;            // maximal time in seconds to be spent on proof of work
    var powTarget = 0.1;        // minimal PoW target required for this message
    var ttl = 20;               // envelope time-to-live in seconds
    var topic1 = '0xdeadbeef';  // each topic 4 bytes, as hex
    var topic2 = '0xbeefdead';  // each topic 4 bytes, as hex
    var identity1 = '0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3';
    var identity2 = '0x0490161b00f2c47542d28c2e8908e77159b1720dccceb6393d7c001850122efc3b1709bcea490fd8f5634ba1a145aa0722d86b9330b0e39a8d493cb981fd459da2';
    
    // makeTopic generates random topic (4 bytes, in hex)
    var makeTopic = function () {
        var min = 1;
        var max = Math.pow(16, 8);
        var randInt = Math.floor(Math.random() * (max - min + 1)) + min;
        return web3.toHex(randInt);
    };

    // watch makes sure that we halt the filter on first message received
    var watchFilter = function (filter, callback) {
        var messageReceived = false;
        filter.watch(function (error, message) {
            if (messageReceived)  return; // avoid double calling
            messageReceived = true; // no need to watch for the filter any more
            filter.stopWatching();
            callback(error, message);
        });
    };

    console.log('Node is expected: statusd --datadir app1 wnode --http --httpport 8645');
    console.log('Node is expected: statusd --datadir app1 wnode --http --httpport 8745');
    console.log('Node is expected: statusd --datadir wnode1 wnode --notify --injectaccounts=false --identity ./static/keys/wnodekey --firebaseauth ./static/keys/firebaseauthkey --http --httpport 8845');

    if (!node1.isConnected()) throw 'node1 is not available!';
    if (!node2.isConnected()) throw 'node2 is not available!';
    if (!node3.isConnected()) throw 'notification service node is not available!';

    context('shh/5 API verification', function () {
       
        context('status', function () {
            it('shh.version()', function () {
                var version = node1.shh.version();
                assert.equal(version, '5.0', 'Whisper version does not match');
            });

            it('shh.info()', function () {
                var info = node1.shh.info();
                expect(info).to.have.property('memory');
                expect(info).to.have.property('messages');
                expect(info).to.have.property('minPow');
                expect(info).to.have.property('maxMessageSize');
            });
        });

        context('symmetric key management', function () {
            var keyId = '';  // symmetric key Id (to be populated)
            var keyVal = ''; // symmetric key value (to be populated)

            it('shh.newSymKey()', function () {
                keyId = node1.shh.newSymKey();
                assert.lengthOf(keyId, 64, 'invalid keyId length');
            });

            it('shh.getSymKey(keyId)', function () {
                keyVal = node1.shh.getSymKey(keyId);
                assert.lengthOf(keyVal, 66, 'invalid key value length'); // 2 bytes for "0x"
            });

            it('shh.hasSymKey(keyId)', function () {
                expect(node1.shh.hasSymKey(keyId)).to.equal(true);
            });

            it('shh.deleteSymKey(keyId)', function () {
                expect(node1.shh.hasSymKey(keyId)).to.equal(true);
                node1.shh.deleteSymKey(keyId);
                expect(node1.shh.hasSymKey(keyId)).to.equal(false);
            });

            it('shh.addSymKey(keyVal)', function () {
                var keyIdOriginal = keyId;
                keyId = node1.shh.addSymKey(keyVal);
                assert.notEqual(keyId, keyIdOriginal);
                assert.lengthOf(keyId, 64, 'invalid keyId length');
                expect(node1.shh.hasSymKey(keyId)).to.equal(true);
            });

            it('shh.generateSymKeyFromPassword(password)', function () {
                var password = 'foobar';
                var keyId = node1.shh.generateSymKeyFromPassword(password);
                var keyVal = node1.shh.getSymKey(keyId);
                assert.lengthOf(keyId, 64, 'invalid keyId length');
                expect(node1.shh.hasSymKey(keyId)).to.equal(true);
                assert.equal(keyVal, '0xa582720d74d463589df14c11538189a1c07778c47e86f70bab7b5ba27e2de3cc');
            });
        });

        context('asymmetric key management', function () {
            var keyId = '';  // to be populated
            var pubKey = ''; // to be populated
            var prvKey = '0x8bda3abeb454847b515fa9b404cede50b1cc63cfdeddd4999d074284b4c21e15';

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

            it('shh.addPrivateKey(prvKey)', function () {
                keyId = node1.shh.addPrivateKey(prvKey);
                assert.lengthOf(keyId,64);
            });
        });

    });

    context('message travelling from one node to another', function () {
        var keyId1 = ''; // symmetric key Id on node 1 (to be populated)
        var keyId2 = ''; // symmetric key Id on node 2 (to be populated)

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
            keyId1 = node1.shh.newSymKey();

            // obtain key value
            var keyVal = node1.shh.getSymKey(keyId1);
            assert.lengthOf(keyVal, 66); // 2 bytes of "0x"

            // share the value with the node2
            keyId2 = node2.shh.addSymKey(keyVal);
        });

        
        it('send symmetrically encrypted, signed message (node1 -> node2)', function (done) {
            var payload = web3.fromAscii('send symmetrically encrypted, signed message (node1 -> node2)');
            var topic = makeTopic();
            var onCreationError = function (error) {
                done(error);
            }
            var onMessage = function (error, message) {
                done(error);
            }

            // start watching for messages
            var params = {
                symKeyId: keyId2,
                sig: identity1,
                topics: [topic]
            }
            filter = node2.shh.newMessageFilter(params, null, onCreationError);
            watchFilter(filter, onMessage);
            
            // send message
            var message = {
                symKeyId: keyId1,
                ttl: ttl,
                sig: identity1,
                topic: topic,
                payload: payload,
                powTime: powTime,
                powTarget: powTarget        
            };
            expect(node1.shh.post(message)).to.equal(true);
        });
        
        it('send asymmetrically encrypted, signed message (node1.id1 -> node2.id2)', function (done) {
            var payload = web3.fromAscii('send asymmetrically encrypted, signed message (node1.id1 -> node2.id2)');
            var topic = makeTopic();
            var onCreationError = function (error) {
                done(error);
            }
            var onMessage = function (error, message) {
                done(error);
            }

            // start watching for messages
            var params = {
                privateKeyId: identity2,
                sig: identity1,
                topic: topic
            }
            filter = node2.shh.newMessageFilter(params, null, onCreationError);
            watchFilter(filter, onMessage);

            // send message
            var message = {
                pubKey: identity2,
                ttl: ttl,
                sig: identity1,
                topic: topic,
                payload: payload,
                powTime: powTime,
                powTarget: powTarget        
            }; 
            expect(node1.shh.post(message)).to.equal(true);
        });
    });

    context('chat messages & push notifications', function () {
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

        // ensures that message has payload (which is HEX-encoded JSON)
        var extractPayload = function (message) {
            expect(message).to.have.property('payload');
            return JSON.parse(web3.toAscii(message.payload));
        };

        var identity1 = ''; // pub key of device 1
        var identity2 = ''; // pub key of device 2
        var chatKeySharingTopic = makeTopic(); // topic used by device1 to send chat key to device 2
        var chatKeyId1 = ''; // symkey provided by server, and shared among clients so that they can trigger notifications
        var chatKeyId2 = ''; // symkey provided by server, and shared among clients so that they can trigger notifications
        
        context('prepare devices', function () {
            it('create key pair to be used as main identity on device1', function () {
                var keyId = node1.shh.newKeyPair();
                identity1 = node1.shh.getPublicKey(keyId);
            });

            it('create key pair to be used as main identity on device2', function () {
                var keyId = node2.shh.newKeyPair();
                identity2 = node2.shh.getPublicKey(keyId);
            });
        });

        context('run device1', function () {
            var serverId = ''; // accepted/selected server id
            var subscriptionKeyId = ''; // symkey provided by server, and used to configure client-server subscription
            var appChatId = ''; // chat id that identifies device1-device2 interaction session on RN app level

            it('send discovery request & watch for server proposal ', function (done) {
                // watch for server proposal
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    serverId = payload.server;
                    done();
                }
                var params = {
                    privateKeyId: identity1,
                    topics: [proposeServerTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);
                 
                // send discovery request
                var message = {
                    pubKey: discoveryPubKey,
                    ttl: ttl,
                    sig: identity1,
                    topic: discoverServerTopic,
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });
        
            it('accept server & receive server ack', function (done) {
                // watch for server ack
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) return done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');
                    // save subscription key
                    subscriptionKeyId = node1.shh.addSymKey(payload.key);
                    assert.lengthOf(subscriptionKeyId, 64);
                    expect(node1.shh.hasSymKey(subscriptionKeyId)).to.equal(true);
                    done();
                }
                var params = {
                    privateKeyId: identity1,
                    sig: discoveryPubKey,
                    topics: [ackClientSubscriptionTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);
                
                // accept server
                var message = {
                    pubKey: discoveryPubKey,
                    ttl: ttl,
                    sig: identity1,
                    topic: acceptServerTopic,
                    payload: web3.fromAscii('{"server": "' + serverId + '"}'),
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });
            
            it('create chat session & receive chat key', function (done) {
                // watch for chat key message
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) return done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');
                    chatKeyId1 = node1.shh.addSymKey(payload.key);
                    assert.lengthOf(chatKeyId1, 64);
                    expect(node1.shh.hasSymKey(chatKeyId1)).to.equal(true);
                    done();
                }
                var params = {
                    privateKeyId: identity1,
                    topics: [ackNewChatSessionTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);

                // create chat session request
                appChatId = makeTopic(); // globally unique chat id
                var message = {
                    symKeyId: subscriptionKeyId,
                    ttl: ttl,
                    sig: identity1,
                    topic: newChatSessionTopic,
                    payload: web3.fromAscii('{"chat": "' + appChatId + '"}'),
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });

            it('register device with a given chat & receive acknowledgment', function (done) {
                // watch for device registration acknowledgement message
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) return done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    done();
                }
                var params = {
                    privateKeyId: identity1,
                    topics: [ackDeviceRegistrationTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);
                
                // send device registration request
                // this obtained from https://status-sandbox-c1b34.firebaseapp.com/
                var deviceId = 'cqMiF8cwhSM:APA91bHrv2HUajq77_gKjeUxfzKjTxU7zGsq9sgr5QdfLW3w1KVFwT7GDOGcvwmHznkA3F3dZ-eohw9PVvGjAeo5gOqlDODMZqPVWzqeuslFbu5tGgTvdjQXSpWouya-63_dT67H0a29';
                var message = {
                    symKeyId: chatKeyId1,
                    ttl: ttl,
                    sig: identity1,
                    topic: newDeviceRegistrationTopic,
                    payload: web3.fromAscii('{"device": "' + deviceId + '"}'),
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });

            it('share chat key with device2 & device 2 receives message', function (done) {
                // watch for chat key message sent by device 1
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) return done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('chat');
                    expect(payload).to.have.property('key');
                    chatKeyId2 = node2.shh.addSymKey(payload.key);
                    assert.lengthOf(chatKeyId2, 64);
                    expect(node2.shh.hasSymKey(chatKeyId2)).to.equal(true);
                    done();
                }
                var params = {
                    privateKeyId: identity2,
                    topics: [chatKeySharingTopic]
                }
                filter = node2.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);
                
                // share chat key with device 2
                var chatKey = node1.shh.getSymKey(chatKeyId1);
                assert.lengthOf(chatKey, 66);
                var message = {
                    pubKey: identity2,
                    ttl: ttl,
                    sig: identity1,
                    topic: chatKeySharingTopic,
                    payload: web3.fromAscii('{"chat": "' + appChatId + '", "key": "' + chatKey + '"}'),
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });
        });

        context('run device2', function () {
        
            it('send chat message, trigger notification (from device2, on device1) & device 1 receives message', function (done) {
                // watch for chat messages sent by device 2
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    done(error);
                }
                var params = {
                    symKeyId: chatKeyId1,
                    topics: [sendNotificationTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage, done);      
                
                // send message + notification request (from device2, on device1)
                var message = {
                    symKeyId: chatKeyId2,
                    ttl: ttl,
                    sig: identity2,
                    topic: sendNotificationTopic,
                    payload: web3.fromAscii('{' // see https://firebase.google.com/docs/cloud-messaging/http-server-ref
                    + '"notification": {'
                    + '"title": "status.im notification",'
                    + '"body": "Hello this is test notification!",'
                    + '"icon": "https://status.im/img/logo.png",'
                    + '"click_action": "https://status.im"'
                    + '},'
                    + '"to": "{{ Id }}"' // this get replaced by device id your've registered
                    + '}'),
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node2.shh.post(message)).to.equal(true);
            });
        });

        context('misc methods and cleanup', function () {
            it('check client session', function (done) {
                
                // watch for replies with the client session
                var onCreationError = function (error) {
                    done(error);
                }
                var onMessage = function (error, message) {
                    if (error) return done(error);
                    var payload = extractPayload(message);
                    expect(payload).to.have.property('server');
                    expect(payload).to.have.property('key');
                    done();
                }
                var params = {
                    privateKeyId: identity1,
                    topics: [confirmClientSessionTopic]
                }
                filter = node1.shh.newMessageFilter(params, null, onCreationError);
                watchFilter(filter, onMessage);            
                
                // request status
                var message = {
                    pubKey: discoveryPubKey,
                    ttl: ttl,
                    sig: identity1,
                    topic: checkClientSessionTopic,
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });

            it('remove client session', function () {
                
                // remove client session
                var message = {
                    pubKey: discoveryPubKey,
                    ttl: ttl,
                    sig: identity1,
                    topic: dropClientSessionTopic,
                    powTime: powTime,
                    powTarget: powTarget
                };
                expect(node1.shh.post(message)).to.equal(true);
            });
        });
    });
});
