package com.example.waku

import android.os.Bundle
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import com.example.waku.events.Event
import com.example.waku.events.EventHandler
import com.example.waku.events.EventType
import com.example.waku.events.MessageEvent
import com.example.waku.messages.Message
import com.example.waku.messages.decodeAsymmetric
import gowaku.Gowaku.defaultPubsubTopic

val alicePrivKey: String = "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e"
val alicePubKey: String =
    "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302"
val bobPrivKey: String = "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453"
val bobPubKey: String =
    "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795dd7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd"

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val lbl = findViewById<TextView>(R.id.lbl)

        // This configuration and its attributes are optional
        var c = Config()
        c.relay = true

        var node = Node(c)

        // A callback must be registered to receive events
        class MyEventHandler(var lbl: TextView) : EventHandler {
            override fun handleEvent(evt: Event) {
                lbl.text =
                    (lbl.text.toString() + ">>> Received a signal: " + evt.type.toString() + "\n")
                if (evt.type == EventType.Message) {
                    val m = evt as MessageEvent
                    val decodedPayload = m.event.wakuMessage.decodeAsymmetric(bobPrivKey)
                    lbl.text =
                        (lbl.text.toString() + ">>> Message: " + decodedPayload.data.toString(
                            Charsets.UTF_8
                        ) + "\n")
                }
            }
        }
        node.setEventHandler(MyEventHandler(lbl))

        node.start()

        lbl.text = (lbl.text.toString() + ">>> The node peer ID is " + node.peerID() + "\n")

        node.listenAddresses().forEach {
            lbl.text = (lbl.text.toString() + ">>> Listening on " + it + "\n")
        }

        lbl.text = (lbl.text.toString() + ">>> Default pubsub topic: " + defaultPubsubTopic() + "\n");

        try {
            node.connect("/dns4/node-01.gc-us-central1-a.waku.test.status.im/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG")
            lbl.text = (lbl.text.toString() + ">>> Connected to Peer" + "\n")

            node.peers().forEach {
                lbl.text = (lbl.text.toString() + ">>> Peer: " + it.peerID + "\n")
                lbl.text =
                    (lbl.text.toString() + ">>> Protocols: " + it.protocols.joinToString(",") + "\n")
                lbl.text =
                    (lbl.text.toString() + ">>> Addresses: " + it.addrs.joinToString(",") + "\n")
            }

            /*var q = StoreQuery();
            q.pubsubTopic = defaultPubsubTopic();
            q.pagingOptions = new(3, null, false);
            val response = node.StoreQuery(q);
            println(">>> Retrieved " + response.messages.Count + " messages from store");*/

        } catch (ex: Exception) {
            lbl.text = (lbl.text.toString() + ">>> Could not connect to peer: " + ex.message)
        }

        node.relaySubscribe()

        for (i in 1..5) {
            val payload = ("Hello world! - " + i).toByteArray(Charsets.UTF_8)
            val timestamp = System.currentTimeMillis() * 1000000
            val contentTopic = ContentTopic("example", 1, "example", "rfc26")
            val msg = Message(payload, contentTopic, timestamp = timestamp)
            val messageID = node.relayPublishEncodeAsymmetric(msg, bobPubKey, alicePrivKey)
            Thread.sleep(1_000)
        }

        node.relayUnsubscribe()

        node.stop()
    }
}