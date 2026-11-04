using System.Runtime.InteropServices;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace Waku
{
    public class Config
    {
        public string? host { get; set; }
        public int? port { get; set; }
        public string? advertiseAddr { get; set; }
        public string? nodeKey { get; set; }
        public int? keepAliveInterval { get; set; }
        public bool? relay { get; set; }
        public int? minPeersToPublish {get; set; }
        public bool? enableFilter {get; set; }
    }

    public enum EventType
    {
        Unknown, Message
    }

    public class Message
    {
        public byte[] payload { get; set; } = new byte[0];
        public string contentTopic { get; set; } = "";
        public int version { get; set; } = 0;
        public long? timestamp { get; set; }
    }

    public class DecodedPayload
    {
        public string? pubkey { get; set; }
        public string? signature { get; set; }
        public byte[] data { get; set; } = new byte[0];
        public byte[] padding { get; set; } = new byte[0];
    }

    public class Peer
    {
        public string peerID { get; set; } = "";
        public bool connected { get; set; } = false;
        public IList<string> protocols { get; set; } = new List<string>();
        public IList<string> addrs { get; set; } = new List<string>();    
    }

    public class Event
    {
        public EventType type { get; set; } = EventType.Unknown;
    }

    public class MessageEventData
    {
        public string messageID { get; set; } = "";

        public string pubsubTopic { get; set; } = Utils.DefaultPubsubTopic();

        public Message wakuMessage { get; set; } = new Message();
    }

    public class MessageEvent : Event
    {
        [JsonPropertyName("event")]
        public MessageEventData data { get; set; } = new();
    }

    public class ContentFilter
    {
        public ContentFilter(string contentTopic)
        {
            this.contentTopic = contentTopic;
        }
        public string contentTopic { get; set; } = "";
    }
    
    public class Cursor
    {
        public byte[] digest { get; set; } = new byte[0];
        public string pubsubTopic { get; set; } = Utils.DefaultPubsubTopic();
        public long receiverTime { get; set; } = 0;
        public long senderTime { get; set; } = 0;
    }

    public class PagingOptions
    {
        public PagingOptions(int pageSize, Cursor? cursor, bool forward)
        {
            this.pageSize = pageSize;
            this.cursor = cursor;
            this.forward = forward;
        }
        public int pageSize { get; set; } = 100;
        public Cursor? cursor { get; set; } = new();
        public bool forward { get; set; } = true;
    }

    public class StoreQuery
    {
        public string? pubsubTopic { get; set; } = Utils.DefaultPubsubTopic();
        public long? startTime { get; set; }
        public long? endTime { get; set; }
        public IList<ContentFilter>? contentFilters { get; set; }
        public PagingOptions? pagingOptions { get; set; }
    }

    public class StoreResponse
    {
        public IList<Message> messages { get; set; } = new List<Message>();
        public PagingOptions? pagingInfo { get; set; }
    }

     public class FilterSubscription
    {
        public IList<ContentFilter> contentFilters { get; set; } = new List<ContentFilter>();
        public string? topic { get; set; }
    }

    public class Node
    {
        private bool _running;
        private static SignalHandlerDelegate? _signalHandler;

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_new(string configJSON);

        /// <summary>
        /// Initialize a go-waku node.
        /// </summary>
        /// <param name="c">Waku.Config containing the options used to initialize a go-waku node. It can be NULL to use defaults. All the keys from the configuration are optional </param>
        /// <returns>The node id</returns>
        public Node(Config? c = null)
        {
            string jsonString = "{}";
            if (c != null)
            {
                jsonString = JsonSerializer.Serialize(c);
            }

            IntPtr ptr = waku_new(jsonString);

            Response.HandleResponse(ptr);

            SetEventCallback();
        }

        ~Node()
        {
            if (_running)
            {
                Stop();
            }
        }

        private static void DefaultEventHandler(IntPtr signalJSON)
        {
            if (_signalHandler != null)
            {
                string signalStr = Response.PtrToStringUtf8(signalJSON, false);
                JsonSerializerOptions options = new JsonSerializerOptions();
                options.Converters.Add(new JsonStringEnumConverter(JsonNamingPolicy.CamelCase));

                Event? evt = JsonSerializer.Deserialize<Event>(signalStr, options);
                if (evt == null) return;

                switch (evt.type)
                {
                    case EventType.Message:
                        Event? msgEvt = JsonSerializer.Deserialize<MessageEvent>(signalStr, options);
                        if (msgEvt != null) _signalHandler(msgEvt);
                        break;
                }
            }
        }

        [UnmanagedFunctionPointer(CallingConvention.StdCall)]
        internal delegate void WakuEventHandlerDelegate(IntPtr signalJSON);

        [DllImport(Constants.dllName, CallingConvention = CallingConvention.Cdecl)]
        internal static extern void waku_set_event_callback(WakuEventHandlerDelegate cb);

        private void SetEventCallback()
        {
            waku_set_event_callback(DefaultEventHandler);
        }

        public delegate void SignalHandlerDelegate(Event evt);

        /// <summary>
        /// Register callback to act as signal handler and receive application signals which are used to react to asyncronous events in waku.
        /// <param name="h">SignalHandler with the signature `void SignalHandler(Waku.Event evt){ }`</param>
        public void SetEventHandler(SignalHandlerDelegate h)
        {
            _signalHandler = h;
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_start();

        /// <summary>
        /// Initialize a go-waku node mounting all the protocols that were enabled during the waku node instantiation.
        /// </summary>
        public void Start()
        {
            if(_running) {
                return
            }

            IntPtr ptr = waku_start();
            Response.HandleResponse(ptr);

            _running = true;
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_stop();

        /// <summary>
        /// Stops a go-waku node
        /// </summary>
        public void Stop()
        {
            if(!_running) {
                return
            }
            
            IntPtr ptr = waku_stop();
            Response.HandleResponse(ptr);

            _running = false;
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_peerid();

        /// <summary>
        /// Obtain the peer ID of the go-waku node.
        /// </summary>
        /// <returns>The base58 encoded peer Id</returns>
        public string PeerId()
        {
            IntPtr ptr = waku_peerid();
            return Response.HandleResponse(ptr, "could not obtain the peerId");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_peer_cnt();

        /// <summary>
        /// Obtain number of connected peers
        /// </summary>
        /// <returns>The number of peers connected to this node</returns>
        public int PeerCnt()
        {
            IntPtr ptr = waku_start();
            return Response.HandleResponse<int>(ptr, "could not obtain the peer count");
        }


        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_listen_addresses();

        /// <summary>
        /// Obtain the multiaddresses the wakunode is listening to
        /// </summary>
        /// <returns>List of multiaddresses</returns>
        public IList<string> ListenAddresses()
        {
            IntPtr ptr = waku_listen_addresses();
            return Response.HandleListResponse<string>(ptr, "could not obtain the addresses");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_add_peer(string address, string protocolId);

        /// <summary>
        /// Add node multiaddress and protocol to the wakunode peerstore
        /// </summary>
        /// <param name="address">multiaddress of the peer being added</param>
        /// <param name="protocolId">protocol supported by the peer</param>
        /// <returns>Base58 encoded peer Id</returns>
        public string AddPeer(string address, string protocolId)
        {
            IntPtr ptr = waku_add_peer(address, protocolId);
            return Response.HandleResponse(ptr, "could not obtain the peer id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_connect(string address, int ms);

        /// <summary>
        /// Connect to peer at multiaddress.
        /// </summary>
        /// <param name="address">multiaddress of the peer being dialed</param>
        /// <param name="ms">max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use 0 for unlimited duration</param>
        public void Connect(string address, int ms = 0)
        {
            IntPtr ptr = waku_connect(address, ms);
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_connect_peerid( string peerId, int ms);

        /// <summary>
        /// Connect to peer using peerID.
        /// </summary>
        /// <param name="peerId"> peerID to dial. The peer must be already known, added with `AddPeer`</param>
        /// <param name="ms">max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use 0 for unlimited duration</param>
        public void ConnectPeerId(string peerId, int ms = 0)
        {
            IntPtr ptr = waku_connect_peerid(peerId, ms);
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_disconnect(string peerID);

        /// <summary>
        /// Close connection to a known peer by peerID
        /// </summary>
        /// <param name="peerId">Base58 encoded peer ID to disconnect</param>
        public void Disconnect(string peerId)
        {
            IntPtr ptr = waku_disconnect(peerId);
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_publish(string messageJSON, string? topic, int ms);

        /// <summary>
        ///  Publish a message using waku relay.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string RelayPublish(Message msg, string? topic = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_relay_publish(jsonMsg, topic, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_lightpush_publish(string messageJSON, string? topic, string? peerID, int ms);

        /// <summary>
        ///  Publish a message using waku lightpush.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="peerID">ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string LightpushPublish(Message msg, string? topic = null, string? peerID = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_lightpush_publish(jsonMsg, topic, peerID, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_publish_enc_asymmetric(string messageJSON, string? topic, string publicKey, string? optionalSigningKey, int ms);

        /// <summary>
        /// Publish a message encrypted with an secp256k1 public key using waku relay.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="publicKey">Secp256k1 public key</param>
        /// <param name="optionalSigningKey">Optional secp256k1 private key for signing the message</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string RelayPublishEncodeAsymmetric(Message msg, string publicKey, string? optionalSigningKey = null, string? topic = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_relay_publish_enc_asymmetric(jsonMsg, topic, publicKey, optionalSigningKey, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_lightpush_publish_enc_asymmetric(string messageJSON, string? topic, string? peerID, string publicKey, string? optionalSigningKey, int ms);

        /// <summary>
        /// Publish a message encrypted with an secp256k1 public key using waku lightpush.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="publicKey">Secp256k1 public key</param>
        /// <param name="optionalSigningKey">Optional secp256k1 private key for signing the message</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="peerID">ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string LightpushPublishEncodeAsymmetric(Message msg, string publicKey, string? optionalSigningKey = null, string? topic = null, string? peerID = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_lightpush_publish_enc_asymmetric(jsonMsg, topic, peerID, publicKey, optionalSigningKey, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_publish_enc_symmetric(string messageJSON, string? topic, string symmetricKey, string? optionalSigningKey, int ms);

        /// <summary>
        /// Publish a message encrypted with a 32 bytes symmetric key using waku relay.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="symmetricKey">32 byte hex string containing a symmetric key</param>
        /// <param name="optionalSigningKey">Optional secp256k1 private key for signing the message</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string RelayPublishEncodeSymmetric(Message msg, string symmetricKey, string? optionalSigningKey = null, string? topic = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_relay_publish_enc_symmetric(jsonMsg, topic, symmetricKey, optionalSigningKey, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_lightpush_publish_enc_symmetric(string messageJSON, string? topic, string? peerID, string symmetricKey, string? optionalSigningKey, int ms);

        /// <summary>
        /// Publish a message encrypted with a 32 bytes symmetric key using waku lightpush.
        /// </summary>
        /// <param name="msg">Message to broadcast</param>
        /// <param name="symmetricKey">32 byte hex string containing a symmetric key</param>
        /// <param name="optionalSigningKey">Optional secp256k1 private key for signing the message</param>
        /// <param name="topic">Pubsub topic. Set to `null` to use the default pubsub topic</param>
        /// <param name="peerID">ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node</param>
        /// <param name="ms">If ms is greater than 0, the broadcast of the message must happen before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns></returns>
        public string RelayPublishEncodeSymmetric(Message msg, string symmetricKey, string? optionalSigningKey = null, string? topic = null, string? peerID = null, int ms = 0)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_lightpush_publish_enc_symmetric(jsonMsg, topic, peerID, symmetricKey, optionalSigningKey, ms);
            return Response.HandleResponse(ptr, "could not obtain the message id");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_decode_symmetric(string messageJSON, string symmetricKey);

        /// <summary>
        /// Decode a waku message using a symmetric key
        /// </summary>
        /// <param name="msg">Message to decode</param>
        /// <param name="symmetricKey">Symmetric key used to decode the message</param>
        /// <returns>DecodedPayload containing the decrypted message, padding, public key and signature (if available)</returns>
        public DecodedPayload DecodeSymmetric(Message msg, string symmetricKey)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_decode_symmetric(jsonMsg, symmetricKey);
            return Response.HandleDecodedPayloadResponse(ptr, "could not decode message");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_decode_asymmetric(string messageJSON, string privateKey);

        /// <summary>
        /// Decode a waku message using an asymmetric key
        /// </summary>
        /// <param name="msg">Message to decode</param>
        /// <param name="privateKey">Secp256k1 private key used to decode the message</param>
        /// <returns>DecodedPayload containing the decrypted message, padding, public key and signature (if available)</returns>
        public DecodedPayload DecodeAsymmetric(Message msg, string privateKey)
        {
            string jsonMsg = JsonSerializer.Serialize(msg);
            IntPtr ptr = waku_decode_asymmetric(jsonMsg, privateKey);
            return Response.HandleDecodedPayloadResponse(ptr, "could not decode message");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_enough_peers(string topic);

        /// <summary>
        /// Determine if there are enough peers to publish a message on a topic.
        /// </summary>
        /// <param name="topic">pubsub topic to verify. Use NULL to verify the number of peers in the default pubsub topic</param>
        /// <returns>boolean indicating if there are enough peers or not</returns>
        public bool RelayEnoughPeers(string topic = null)
        {
            IntPtr ptr = waku_relay_enough_peers(topic);
            return Response.HandleResponse<bool>(ptr, "could not determine if there are enough peers");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_subscribe(string? topic);

        /// <summary>
        /// Subscribe to a WakuRelay topic to receive messages.
        /// </summary>
        /// <param name="topic">Pubsub topic to subscribe to. Use NULL for subscribing to the default pubsub topic</param>
        /// <returns>Subscription Id</returns>
        public void RelaySubscribe(string? topic = null)
        {
            IntPtr ptr = waku_relay_subscribe(topic);
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_relay_unsubscribe(string? topic);

        /// <summary>
        /// Closes the pubsub subscription to a pubsub topic.
        /// </summary>
        /// <param name="topic">Pubsub topic to unsubscribe. Use NULL for unsubscribe from the default pubsub topic</param>
        public void RelayUnsubscribe(string? topic = null)
        {
            IntPtr ptr = waku_relay_unsubscribe(topic);
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_peers();

        /// <summary>
        /// Get Peers
        /// </summary>
        /// <returns>Retrieve list of peers and their supported protocols</returns>
        public IList<Peer> Peers()
        {
            IntPtr ptr = waku_peers();
            return Response.HandleListResponse<Peer>(ptr, "could not obtain the peers");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_store_query(string queryJSON, string? peerID, int ms);
        /// <summary>
        /// Query message history
        /// </summary>
        /// <param name="query">Query</param>
        /// <param name="peerID">PeerID to ask the history from. Use NULL to automatically select a peer</param>
        /// <param name="ms">If ms is greater than 0, the response must be received before the timeout (in milliseconds) is reached, or an error will be returned</param>
        /// <returns>Response containing the messages and cursor for pagination. Use the cursor in further queries to retrieve more results</returns>
        public StoreResponse StoreQuery(StoreQuery query, string? peerID = null, int ms = 0)
        {
            string queryJSON = JsonSerializer.Serialize(query);
            IntPtr ptr = waku_store_query(queryJSON, peerID, ms);

            return Response.HandleStoreResponse(ptr, "could not extract query response");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_filter_subscribe(string filterJSON, string? peerID, int ms);
        /// <summary>
        /// Creates a subscription in a lightnode for messages
        /// </summary>
        /// <param name="filter">Filter criteria</param>
        /// <param name="peerID">PeerID to subscribe to</param>
        /// <param name="ms">If ms is greater than 0, the subscription must be done before the timeout (in milliseconds) is reached, or an error will be returned</param>
        public FilterSubscribe(FilterSubscription filter, string? peerID = null, int ms = 0)
        {
            string filterJSON = JsonSerializer.Serialize(filter);
            IntPtr ptr = waku_filter_subscribe(filterJSON, peerID, ms);
            
            Response.HandleResponse(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_filter_unsubscribe(string filterJSON, int ms);
        /// <summary>
        /// Removes subscriptions in a light node 
        /// </summary>
        /// <param name="filter">Filter criteria</param>
        /// <param name="ms">If ms is greater than 0, the unsubscription must be done before the timeout (in milliseconds) is reached, or an error will be returned</param>
        public FilterUnsubscribe(FilterSubscription filter, int ms = 0)
        {
            string filterJSON = JsonSerializer.Serialize(filter);
            IntPtr ptr = waku_filter_unsubscribe(filterJSON, ms);
            
            Response.HandleResponse(ptr);
        }
    }
}

