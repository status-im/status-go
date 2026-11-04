//
//  Response.swift
//  swift-waku
//

import Foundation

struct WakuError: Error {
    let message: String

    init(_ message: String) {
        self.message = message
    }

    public var localizedDescription: String {
        return message
    }
}

func handleResponse<T: Codable>(_ response: String) throws -> T {
    let decoder = JSONDecoder()
    let jsonResult = try! decoder.decode(JsonResult<T>.self, from: response.data(using: .utf8)!)
    
    if (jsonResult.error != nil) {
        throw WakuError(jsonResult.error!)
    }
    
    if (jsonResult.result == nil) {
        throw WakuError("no result in response")
    }
    
    return jsonResult.result!;
}

func handleResponse(_ response: String) throws {
    let decoder = JSONDecoder()
    let jsonResult = try! decoder.decode(JsonResult<String>.self, from: response.data(using: .utf8)!)
    
    if jsonResult.error != nil {
        throw WakuError("no result in response")
    }
}
