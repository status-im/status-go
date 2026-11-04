//
//  Config.swift
//  swift-waku
//

import Foundation

class Config: Codable {
    var host: String? = nil;
    var port: Int? = nil;
    var advertiseAddr: String? = nil;
    var nodeKey: String? = nil;
    var keepAliveInterval: Int? = nil;
    var relay: Bool? = nil;
    var minPeersToPublish: Int? = nil;
}
