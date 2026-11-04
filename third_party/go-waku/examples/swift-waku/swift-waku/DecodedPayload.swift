//
//  DecodedPayload.swift
//  swift-waku
//

import Foundation

class DecodedPayload: Codable {
    var pubkey: String?
    var signature: String?
    var data: [UInt8]
    var padding: String?
}
