//
//  JsonResult.swift
//  swift-waku
//

import Foundation

class JsonResult<T: Codable>: Codable {
    var error: String? = nil;
    var result: T? = nil;
}
