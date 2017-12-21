const crypto = require('crypto');
const { spawn } = require('child_process');
const { expect } = require('chai');
const axios = require('axios');
const rimraf = require('rimraf');
const Web3 = require('web3');

describe('Whisper MailServer', () => {
    const topic = `0x${crypto.randomBytes(4).toString('hex')}`;
    const sharedSymKey = '0x6c32583c0bc13ef90a10b36ed6f66baaa0e537d0677619993bfd72c819cba6f3';
    const mailServerEnode = 'enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@127.0.0.1:8549';
    const messageTTL = 5;

    describe('Check prerequisites', () => {
        console.log('Expecting MailServer running.')
        console.log('./build/bin/wnode-status -mailserver -passwordfile=./static/keys/wnodepassword -http -httpport 8540 -listenaddr=127.0.0.1:8549 -identity=./static/keys/wnodekey')

        it('MailServer should be running', () => {
            const mailServer = new Web3(new Web3.providers.HttpProvider('http://localhost:8540'));
            const version = mailServer.shh.version();
            expect(version).to.equal("5.0");
        });
    });

    describe('NodeA', () => {
        let nodeA;
        let nodeAProcess;

        before((done) => {
            nodeAProcess = spawn(
                './build/bin/wnode-status',
                ['-datadir', 'wnode-data-1', '-http', '-httpport', '8590']
            );
            nodeA = new Web3(new Web3.providers.HttpProvider('http://localhost:8590'));

            // need to wait a bit until the node is up and running
            setTimeout(done, 500);
        });

        after((done) => {
            nodeAProcess.kill('SIGTERM');
            nodeAProcess.on('exit', (code, signal) => {
                expect(code).to.be.null;
                expect(signal).to.equal('SIGTERM');
                rimraf('wnode-data-1', done);
            });
        });

        it('Should add MailServer as a peer', (done) => {
            // add MailServer as a peer
            axios.post(nodeA.currentProvider.host, {
                method: 'admin_addPeer',
                params: [mailServerEnode],
                id: 1
            }).then((resp) => {
                expect(resp.data.id).to.equal(1);
                expect(resp.data.result).to.equal(true);
                done();
            }).catch(done);
        })

        it('Should send a message', (done) => {
            const symKeyId = nodeA.shh.addSymKey(sharedSymKey);
            const result = nodeA.shh.post({
                symKeyID: symKeyId,
                topic: topic,
                payload: nodeA.toHex('hello!'),
                ttl: messageTTL,
                powTime: 10,
                powTarget: 2.5
            });
            expect(result).to.be.true;

            // give it some time to propagate before the node is shut down
            setTimeout(done, 500);
        });
    });

    describe('NodeB', () => {
        let nodeBProcess;
        let nodeB;

        before((done) => {
            nodeBProcess = spawn(
                './build/bin/wnode-status',
                ['-datadir', 'wnode-data-2', '-http', '-httpport', '8591', '-log', 'INFO', '-logfile', 'wnode-data-2/wnode.log']
            );
            nodeB = new Web3(new Web3.providers.HttpProvider('http://localhost:8591'));

            // need to wait a bit until the node is up and running
            setTimeout(done, 500);
        });

        after((done) => {
            nodeBProcess.kill('SIGTERM');
            nodeBProcess.on('exit', (code, signal) => {
                expect(code).to.be.null;
                expect(signal).to.equal('SIGTERM');
                rimraf('wnode-data-2', done);
            });
        });

        it('Should add MailServer as a peer', (done) => {
            // add MailServer as a peer
            axios.post(nodeB.currentProvider.host, {
                method: 'admin_addPeer',
                params: [mailServerEnode],
                id: 1
            }).then((resp) => {
                expect(resp.data.id).to.equal(1);
                expect(resp.data.result).to.equal(true);
                done();
            }).catch(done);
        })

        it('Should request and receive old messages', (done) => {
            const mailServerSymKeyID = nodeB.shh.generateSymKeyFromPassword('status-offline-inbox');
            const symKeyId = nodeB.shh.addSymKey(sharedSymKey);

            let requestedForMessages = false;

            // wait until the message expires before setting up a filter
            setTimeout(() => {
                let counter = 0;
                nodeB.shh.newMessageFilter({
                    topics: [topic],
                    symKeyID: symKeyId,
                    allowP2P: true
                }, (err, data) => {
                    if (err) {
                        done(err);
                        return;
                    }

                    expect(nodeB.toAscii(data.payload)).to.equal('hello!');

                    if (requestedForMessages) {
                        done();
                    } else {
                        done('should not receive the message before requesting it');
                    }
                }, done);
            }, (messageTTL + 1) * 1000);

            // request messages after the filter is set up and give it some addotional time
            // so we are sure that the message was received after requesting it
            setTimeout(() => {
                // send a request for old messages
                axios.post(nodeB.currentProvider.host, {
                    method: 'shh_requestMessages',
                    params: [{
                        mailServerPeer: mailServerEnode,
                        topic: topic,
                        symKeyID: mailServerSymKeyID
                    }],
                    id: 2
                }).then((resp) => {
                    requestedForMessages = true;

                    expect(resp.data.id).to.equal(2);
                    expect(resp.data.result).to.equal(true);
                }).catch(done);
            }, (messageTTL + 5) * 1000);
        });
    });
});
