using System.Text;


string alicePrivKey = "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e";
string alicePubKey = "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302";

string bobPrivKey = "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453";
string bobPubKey = "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795dd7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd";



Waku.Config c = new(); // This configuration and its attributes are optional
c.relay = true;
c.host = "localhost";

Waku.Node node = new(c);


// A callback must be registered to receive events
void SignalHandler(Waku.Event evt)
{
    Console.WriteLine(">>> Received a signal: " + evt.type);
    if (evt.type == Waku.EventType.Message)
    {
        Waku.MessageEvent msgEvt = (Waku.MessageEvent)evt; // Downcast to specific event type to access the event data
        Waku.DecodedPayload decodedPayload = node.DecodeAsymmetric(msgEvt.data.wakuMessage, bobPrivKey);

        string message = Encoding.UTF8.GetString(decodedPayload.data);
        Console.WriteLine(">>> Message: " + message + " from: " + decodedPayload.pubkey);
    }
}
node.SetEventHandler(SignalHandler); 

node.Start();

Console.WriteLine(">>> The node peer ID is " + node.PeerId());

foreach (string addr in node.ListenAddresses())
{
    Console.WriteLine(">>> Listening on " + addr);
}

Console.WriteLine(">>> Default pubsub topic: " + Waku.Utils.DefaultPubsubTopic());


try
{
    node.Connect("/dns4/node-01.gc-us-central1-a.waku.test.status.im/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG");
    Console.WriteLine(">>> Connected to Peer");

    foreach (Waku.Peer peer in node.Peers())
    {
        Console.WriteLine(">>> Peer: " + peer.peerID);
        Console.WriteLine(">>> Protocols: " + String.Join(", ", peer.protocols.ToArray()));
        Console.WriteLine(">>> Addresses: " + String.Join(", ", peer.addrs.ToArray()));
    }

    Waku.StoreQuery q = new();
    q.pubsubTopic = Waku.Utils.DefaultPubsubTopic();
    q.pagingOptions = new(3, null, false);
    Waku.StoreResponse response = node.StoreQuery(q);
    Console.WriteLine(">>> Retrieved " + response.messages.Count + " messages from store");
}
catch (Exception ex)
{
    Console.WriteLine(">>> Could not connect to peer: " + ex.Message);
}


node.RelaySubscribe();

for (int i = 0; i < 5; i++)
{
    Waku.Message msg = new Waku.Message();
    msg.payload = Encoding.UTF8.GetBytes("Hello World - " + i);
    msg.timestamp = (long)(DateTime.UtcNow - new DateTime(1970, 1, 1, 0, 0, 0)).TotalMilliseconds; // Nanoseconds
    msg.contentTopic = Waku.Utils.ContentTopic("example", 1, "example", "rfc26");
    string messageID = node.RelayPublishEncodeAsymmetric(msg, bobPubKey, alicePrivKey);

    System.Threading.Thread.Sleep(1000);
}

node.RelayUnsubscribe();

Console.ReadLine();
node.Stop();