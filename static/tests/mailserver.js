const crypto = require('crypto');
const { spawn } = require('child_process');
const { expect } = require('chai');
const Web3 = require('web3');

describe('Whisper MailServer', () => {
    const identityA = '0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3';
    const identityB = '0x0490161b00f2c47542d28c2e8908e77159b1720dccceb6393d7c001850122efc3b1709bcea490fd8f5634ba1a145aa0722d86b9330b0e39a8d493cb981fd459da2';
    const topic = `0x${crypto.randomBytes(4).toString('hex')}`;
    const sharedSymKey = '0x6c32583c0bc13ef90a10b36ed6f66baaa0e537d0677619993bfd72c819cba6f3';

    describe('NodeA', () => {
        let nodeA;
        let nodeAProcess;

        before(() => {
            nodeAProcess = spawn(
                './build/bin/wnode-status',
                ['-datadir', 'wnode-data-1', '-http', '-httpport', '8590']
            );
            nodeA = new Web3(new Web3.providers.HttpProvider('http://localhost:8590'));
        });

        after((done) => {
            nodeAProcess.kill('SIGTERM');
            nodeAProcess.on('exit', (code, signal) => {
                expect(code).to.be.null;
                expect(signal).to.equal('SIGTERM');
                done();
            });
        });

        it('Should be online', () => {
            expect(nodeA).to.not.be.null;
            expect(nodeA.isConnected()).to.be.true;
        });

        it('Should use Whisper V5', () => {
            expect(nodeA.shh.version()).to.equal('5.0');
        });

        it('Should send a message', () => {
            const symKeyId = nodeA.shh.addSymKey(sharedSymKey);
            const result = nodeA.shh.post({
                symKeyID: symKeyId,
                topic: topic,
                payload: nodeA.toHex('hello!'),
                ttl: 60,
                powTime: 10,
                powTarget: 2.5
            });
            expect(result).to.be.true;
        });
    });

    describe('NodeB', () => {
        let nodeBProcess;
        let nodeB;

        before(() => {
            nodeBProcess = spawn(
                './build/bin/wnode-status',
                ['-datadir', 'wnode-data-2', '-http', '-httpport', '8591']
            );
            nodeB = new Web3(new Web3.providers.HttpProvider('http://localhost:8591'));
        });

        after((done) => {
            nodeBProcess.kill('SIGTERM');
            nodeBProcess.on('exit', (code, signal) => {
                expect(code).to.be.null;
                expect(signal).to.equal('SIGTERM');
                done();
            });
        });

        it('Should be online', () => {
            expect(nodeB).to.not.be.null;
            expect(nodeB.isConnected()).to.be.true;
        });

        it('Should use Whisper V5', () => {
            expect(nodeB.shh.version()).to.equal('5.0');
        });

        it('Should request and receive old messages', (done) => {
            const symKeyId = nodeB.shh.addSymKey(sharedSymKey);
            nodeB.shh.newMessageFilter({
                topics: [topic],
                symKeyID: symKeyId,
                allowP2P: true
            }, (err, data) => {
                if (!err) {
                    done(err);
                    return;
                }

                done();
            }, (err) => {
                done(err)
            });

            // (optional)
            // nodeB.shh.addMailServer({ ... });

            // send a request for old messages
            // nodeB.shh.requestMessages({
            //   start: timestamp,
            //   end: timestamp,
            //   mailServerID: (optional)
            // });
        });
    });
});
