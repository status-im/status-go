package com.example.waku

import com.example.waku.events.BaseEvent
import com.example.waku.events.EventHandler
import com.example.waku.events.EventType
import com.example.waku.events.MessageEvent
import com.example.waku.messages.Message
import com.example.waku.store.StoreQuery
import com.example.waku.store.StoreResponse
import com.example.waku.filter.FilterSubscription
import gowaku.Gowaku
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json


/**
 * @param c Config containing the options used to initialize a node. It can be `null` to use
 *          defaults. All the keys from the configuration are optional
 */
class Node(c: Config? = null) {
    var running: Boolean = false
    lateinit var signalHandler: gowaku.SignalHandler
    lateinit var eventHandler: EventHandler

    init {
        val configJson = Json.encodeToString(c)
        val response = Gowaku.newNode(configJson)
        handleResponse(response)

        signalHandler = DefaultEventHandler()
        Gowaku.setMobileSignalHandler(signalHandler)
    }

    inner class DefaultEventHandler : gowaku.SignalHandler {
        override fun handleSignal(signalJson: String) {
            if (eventHandler != null) {
                val evt = Json {
                    ignoreUnknownKeys = true; coerceInputValues = true
                }.decodeFromString<BaseEvent>(signalJson)
                when (evt.type) {
                    EventType.Message -> {
                        try {
                            val msgEvt = Json.decodeFromString<MessageEvent>(signalJson)
                            eventHandler.handleEvent(msgEvt)
                        } catch (e: Exception) {
                            // TODO: do something
                        }
                    }
                    else -> {
                        // TODO: do something with invalid message type
                    }
                }
            }
        }
    }
}

/**
 * Register callback to act as event handler and receive application signals which are used to
 * react to asyncronous events in waku.
 * @param handler event handler
 */
fun Node.setEventHandler(handler: EventHandler) {
    eventHandler = handler
}

/**
 * Initialize a node mounting all the protocols that were enabled during the node instantiation.
 */
fun Node.start() {
    if (running) {
        return
    }

    val response = Gowaku.start()
    handleResponse(response)
    running = true
}

/**
 * Stops a node
 */
fun Node.stop() {
    if (!running) {
        return
    }

    val response = Gowaku.stop()
    handleResponse(response)
    running = false
}

/**
 * Obtain the peer ID of the go-waku node.
 * @return The base58 encoded peer Id
 */
fun Node.peerID(): String {
    val response = Gowaku.peerID()
    return handleResponse<String>(response)
}

/**
 * Obtain number of connected peers
 * @return The number of peers connected to this node
 */
fun Node.peerCnt(): Int {
    val response = Gowaku.peerCnt()
    return handleResponse<Int>(response)
}

/**
 * Obtain the multiaddresses the wakunode is listening to
 * @return List of multiaddresses
 */
fun Node.listenAddresses(): List<String> {
    val response = Gowaku.listenAddresses()
    return handleResponse<List<String>>(response)
}

/**
 * Add node multiaddress and protocol to the wakunode peerstore
 * @param address multiaddress of the peer being added
 * @param protocolID protocol supported by the peer
 * @return Base58 encoded peer Id
 */
fun Node.addPeer(address: String, protocolID: String): String {
    val response = Gowaku.addPeer(address, protocolID)
    return handleResponse<String>(response)
}

/**
 * Connect to peer at multiaddress
 * @param address multiaddress of the peer being dialed
 * @param ms max duration in milliseconds this function might take to execute. If the function
 *           execution takes longer than this value, the execution will be canceled and an error
 *           returned. Use 0 for unlimited duration
 */
fun Node.connect(address: String, ms: Long = 0) {
    val response = Gowaku.connect(address, ms)
    handleResponse(response)
}

/**
 * Close connection to a known peer by peerID
 * @param peerID Base58 encoded peer ID to disconnect
 */
fun Node.disconnect(peerID: String) {
    val response = Gowaku.disconnect(peerID)
    handleResponse(response)
}

/**
 * Subscribe to a WakuRelay topic to receive messages
 * @param topic Pubsub topic to subscribe to. Use "" for subscribing to the default pubsub topic
 */
fun Node.relaySubscribe(topic: String = "") {
    val response = Gowaku.relaySubscribe(topic)
    handleResponse(response)
}

/**
 * Publish a message using waku relay
 * @param msg Message to broadcast
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.relayPublish(msg: Message, topic: String = "", ms: Long = 0): String {
    val jsonMsg = Json.encodeToString(msg)
    val response = Gowaku.relayPublish(jsonMsg, topic, ms)
    return handleResponse<String>(response)
}

/**
 * Publish a message using waku lightpush
 * @param msg Message to broadcast
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param peerID ID of a peer supporting the lightpush protocol. Use "" to automatically select a node
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.lightpushPublish(msg: Message, topic: String = "", peerID: String = "", ms: Long = 0): String {
    val jsonMsg = Json.encodeToString(msg)
    val response = Gowaku.lightpushPublish(jsonMsg, topic, peerID, ms)
    return handleResponse<String>(response)
}

/**
 * Publish a message encrypted with an secp256k1 public key using waku relay
 * @param msg Message to broadcast
 * @param publicKey Secp256k1 public key
 * @param optionalSigningKey Optional secp256k1 private key for signing the message
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.relayPublishEncodeAsymmetric(
    msg: Message,
    publicKey: String,
    optionalSigningKey: String = "",
    topic: String = "",
    ms: Long = 0
): String {
    val jsonMsg = Json.encodeToString(msg)
    val response =
        Gowaku.relayPublishEncodeAsymmetric(jsonMsg, topic, publicKey, optionalSigningKey, ms)
    return handleResponse<String>(response)
}

/**
 * Publish a message encrypted with an secp256k1 public key using waku lightpush
 * @param msg Message to broadcast
 * @param publicKey Secp256k1 public key
 * @param optionalSigningKey Optional secp256k1 private key for signing the message
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param peerID ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.lightpushPublishEncodeAsymmetric(
    msg: Message,
    publicKey: String,
    optionalSigningKey: String = "",
    topic: String = "",
    peerID: String = "",
    ms: Long = 0
): String {
    val jsonMsg = Json.encodeToString(msg)
    val response =
        Gowaku.lightpushPublishEncodeAsymmetric(jsonMsg, topic, peerID, publicKey, optionalSigningKey, ms)
    return handleResponse<String>(response)
}

/**
 * Publish a message encrypted with a 32 byte symmetric key using waku relay
 * @param msg Message to broadcast
 * @param symmetricKey 32 byte hex string containing a symmetric key
 * @param optionalSigningKey Optional secp256k1 private key for signing the message
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.relayPublishEncodeSymmetric(
    msg: Message,
    symmetricKey: String,
    optionalSigningKey: String = "",
    topic: String = "",
    ms: Long = 0
): String {
    val jsonMsg = Json.encodeToString(msg)
    val response =
        Gowaku.relayPublishEncodeSymmetric(jsonMsg, topic, symmetricKey, optionalSigningKey, ms)
    return handleResponse<String>(response)
}

/**
 * Publish a message encrypted with a 32 byte symmetric key using waku lightpush
 * @param msg Message to broadcast
 * @param symmetricKey 32 byte hex string containing a symmetric key
 * @param optionalSigningKey Optional secp256k1 private key for signing the message
 * @param topic Pubsub topic. Set to "" to use the default pubsub topic
 * @param peerID ID of a peer supporting the lightpush protocol. Use "" to automatically select a node
 * @param ms If ms is greater than 0, the broadcast of the message must happen before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return message id
 */
fun Node.lightpushPublishEncodeSymmetric(
    msg: Message,
    symmetricKey: String,
    optionalSigningKey: String = "",
    topic: String = "",
    peerID: String = "",
    ms: Long = 0
): String {
    val jsonMsg = Json.encodeToString(msg)
    val response =
        Gowaku.lightpushPublishEncodeSymmetric(jsonMsg, topic, peerID, symmetricKey, optionalSigningKey, ms)
    return handleResponse<String>(response)
}

/**
 * Determine if there are enough peers to publish a message on a topic
 * @param topic pubsub topic to verify. Use "" to verify the number of peers in the default pubsub topic
 * @return boolean indicating if there are enough peers or not
 */
fun Node.relayEnoughPeers(topic: String = ""): Boolean {
    val response = Gowaku.relayEnoughPeers(topic)
    return handleResponse<Boolean>(response)
}

/**
 * Closes the pubsub subscription to a pubsub topic
 * @param topic Pubsub topic to unsubscribe. Use "" for unsubscribe from the default pubsub topic
 */
fun Node.relayUnsubscribe(topic: String = "") {
    val response = Gowaku.relayUnsubscribe(topic)
    handleResponse(response)
}

/**
 * Get peers
 * @return Retrieve list of peers and their supported protocols
 */
fun Node.peers(): List<Peer> {
    val response = Gowaku.peers()
    return handleResponse<List<Peer>>(response)
}

/**
 * Query message history
 * @param query Query
 * @param peerID PeerID to ask the history from. Use "" to automatically select a peer
 * @param ms If ms is greater than 0, response must be received before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 * @return Response containing the messages and cursor for pagination. Use the cursor in further queries to retrieve more results
 */
fun Node.storeQuery(query: StoreQuery, peerID: String = "", ms: Long = 0): StoreResponse{
    val queryJSON = Json.encodeToString(query)
    val response = Gowaku.storeQuery(queryJSON, peerID, ms)
    return handleResponse<StoreResponse>(response)
}

/**
 * Creates a subscription in a lightnode for messages
 * @param filter Filter criteria
 * @param peerID PeerID to subscribe to. Use "" to automatically select a peer
 * @param ms If ms is greater than 0, the subscription must be done before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 */
func Node.filterSubscribe(filter: FilterSubscription, peerID: String = "", ms: Long = 0) {
    val filterJSON = Json.encodeToString(filter)
    val response = Gowaku.filterSubscribe(filterJSON, peerID, ms)
    handleResponse(response)
}

/**
 * Removes subscriptions in a light node
 * @param filter Filter criteria
 * @param ms If ms is greater than 0, the unsubscription must be done before the timeout
 *           (in milliseconds) is reached, or an error will be returned
 */
func Node.filterUnsubscribe(filter: FilterSubscription, ms: Long = 0) {
    val filterJSON = Json.encodeToString(filter)
    val response = Gowaku.filterUnsubscribe(filterJSON, ms)
    handleResponse(response)
}
