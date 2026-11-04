//
//  ContentView.swift
//  swift-waku
//

import SwiftUI
import Gowaku

struct ContentView: View {
    var body: some View {
        Text(GowakuDefaultPubsubTopic())
            .padding()
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
