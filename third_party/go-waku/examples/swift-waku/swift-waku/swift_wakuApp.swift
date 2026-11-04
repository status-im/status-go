//
//  swift_wakuApp.swift
//  swift-waku
//

import SwiftUI

@main
struct swift_wakuApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
    
    init(){
        let n: WakuNode = try! WakuNode(nil)
        try! n.start()
        
        try! n.relaySubscribe()
        
        let msg:Message = Message()
        msg.contentTopic = "abc"
        msg.timestamp = 1
        msg.version = 0
        msg.payload = [UInt8]("Hello World".utf8)
        
        let id = try! n.relayPublish(msg)
        print(id)
        
        
    }
}
