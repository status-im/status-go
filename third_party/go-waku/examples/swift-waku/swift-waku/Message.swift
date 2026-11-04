//
//  Message.swift
//  swift-waku
//

import Foundation
import Gowaku

class Message: Codable {
    var payload: [UInt8] = [];
    var contentTopic: String? = "";
    var version: Int? = 0;
    var timestamp: Int64? = nil;
    
    func decodeAsymmetric(_ privateKey: String) throws -> DecodedPayload {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(self)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuDecodeAsymmetric(jsonMsg, privateKey)
        return try handleResponse(response)
    }

    func decodeSymmetric(symmetricKey: String) throws -> DecodedPayload {
        let jsonEncoder = JSONEncoder()
        let jsonData = try! jsonEncoder.encode(self)
        let jsonMsg = String(data: jsonData, encoding: String.Encoding.utf8)
        let response = GowakuDecodeSymmetric(jsonMsg, symmetricKey)
        return try handleResponse(response)
    }
}
