//
//  WakuNode.swift
//  swift-waku
//

import Foundation
import Gowaku

class WakuNode {
    var running: Bool = false;
    var signalHandler: GowakuSignalHandlerProtocol
    
    internal class DefaultEventHandler: NSObject, GowakuSignalHandlerProtocol {
        func handleSignal(_ signalJson: String?) {
            print("TODO: handle signal - " + signalJson!)
            /*if (eventHandler != null) {
            val evt = Json {
                ignoreUnknownKeys = true; coerceInputValues = true
            }.decodeFromString<BaseEvent>(signalJson)
            when (evt.type) {
                EventType.Message -> {
                    try {
                        val msgEvt = Json.decodeFromString<MessageEvent>(signalJson)
                        eventHandler.handleEvent(msgEvt)
                    } catch (e: Exception) {
                        // TODO: do something
                    }
                }
                else -> {
                    // TODO: do something with invalid message type
                }
            }*/
        }
    }
    
    init(_ c: Config?) throws {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(c)
        let configJson = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuNewNode(configJson)
        
        try handleResponse(response)
        
        signalHandler = DefaultEventHandler()
        GowakuSetMobileSignalHandler(signalHandler)
    }
    
    func start() throws {
        if (running) {
            return
        }
        
        let response = GowakuStart()
        try handleResponse(response)
        running = true
    }
    
    func stop() throws {
        if(!running){
            return
        }
        
        let response = GowakuStop()
        try handleResponse(response)
        running = false
    }
    
    func peerID() throws -> String {
        let response = GowakuPeerID()
        return try handleResponse(response)
    }

    func peerCnt() throws -> Int {
        let response = GowakuPeerCnt()
        return try handleResponse(response)
    }

    func listenAddresses() throws -> [String] {
        let response = GowakuListenAddresses()
        return try handleResponse(response)
    }

    func addPeer(_ address: String, _ protocolID: String) throws -> String {
        let response = GowakuAddPeer(address, protocolID)
        return try handleResponse(response)
    }

    func connect(_ address: String, _ ms: Int = 0) throws {
        let response = GowakuConnect(address, ms)
        try handleResponse(response)
    }

    func disconnect(_ peerID: String) throws {
        let response = GowakuDisconnect(peerID)
        try handleResponse(response)
    }

    func relaySubscribe(_ topic: String = "") throws {
        let response = GowakuRelaySubscribe(topic)
        try handleResponse(response)
    }

    func relayPublish(_ msg: Message, _ topic: String = "", _ ms: Int = 0) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuRelayPublish(jsonMsg, topic, ms)
        return try handleResponse(response)
    }
    
    func lightpushPublish(_ msg: Message, _ topic: String = "", _ peerID: String = "", _ ms: Int = 0) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuLightpushPublish(jsonMsg, topic, peerID, ms)
        return try handleResponse(response)
    }

    func relayPublishEncodeAsymmetric(
        _ msg: Message,
        _ publicKey: String,
        _ optionalSigningKey: String = "",
        _ topic: String = "",
        _ ms: Int = 0
    ) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuRelayPublishEncodeAsymmetric(jsonMsg, topic, publicKey, optionalSigningKey, ms)
        return try handleResponse(response)
    }

    func lightpushPublishEncodeAsymmetric(
        _ msg: Message,
        _ publicKey: String,
        _ optionalSigningKey: String = "",
        _ topic: String = "",
        _ peerID: String = "",
        _ ms: Int = 0
    ) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuLightpushPublishEncodeAsymmetric(jsonMsg, topic, peerID, publicKey, optionalSigningKey, ms)
        return try handleResponse(response)
    }

    func relayPublishEncodeSymmetric(
        msg: Message,
        symmetricKey: String,
        optionalSigningKey: String = "",
        topic: String = "",
        ms: Int = 0
    ) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuRelayPublishEncodeSymmetric(jsonMsg, topic, symmetricKey, optionalSigningKey, ms)
        return try handleResponse(response)
    }

    func lightpushPublishEncodeSymmetric(
        _ msg: Message,
        _ symmetricKey: String,
        _ optionalSigningKey: String = "",
        _ topic: String = "",
        _ peerID: String = "",
        _ ms: Int = 0
    ) throws -> String {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(msg)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuLightpushPublishEncodeSymmetric(jsonMsg, topic, peerID, symmetricKey, optionalSigningKey, ms)
        return try handleResponse(response)
    }

    func relayEnoughPeers(_ topic: String = "") throws -> Bool {
        let response = GowakuRelayEnoughPeers(topic)
        return try handleResponse(response)
    }

    func relayUnsubscribe(_ topic: String = "") throws {
        let response = GowakuRelayUnsubscribe(topic)
        try handleResponse(response)
    }
/*
    func peers() throws -> [Peer] {
        val response = Gowaku.peers()
        return try handleResponse(response)
    }

    func storeQuery(_ query: StoreQuery, _ peerID: String?, ms: Int = 0) throws -> StoreResponse {
        val queryJSON = Json.encodeToString(query)
        val response = Gowaku.storeQuery(queryJSON, peerID, ms)
        return handleResponse<StoreResponse>(response)
    }*/
    
}
