package com.example.waku.messages

import com.example.waku.handleResponse
import com.example.waku.serializers.ByteArrayBase64Serializer
import gowaku.Gowaku
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
data class Message(
    @Serializable(with = ByteArrayBase64Serializer::class) val payload: ByteArray,
    val contentTopic: String? = "",
    val version: Int? = 0,
    val timestamp: Long? = null
)

/**
 * Decode a waku message using an asymmetric key
 * @param msg Message to decode
 * @param privateKey Secp256k1 private key used to decode the message
 * @return DecodedPayload containing the decrypted message, padding, public key and signature (if available)
 */
fun Message.decodeAsymmetric(privateKey: String): DecodedPayload {
    val jsonMsg = Json.encodeToString(this)
    val response = Gowaku.decodeAsymmetric(jsonMsg, privateKey)
    return handleResponse<DecodedPayload>(response)
}

/**
 * Decode a waku message using a symmetric key
 * @param msg Message to decode
 * @param privateKey Symmetric key used to decode the message
 * @return DecodedPayload containing the decrypted message, padding, public key and signature (if available)
 */
fun Message.decodeSymmetric(symmetricKey: String): DecodedPayload {
    val jsonMsg = Json.encodeToString(this)
    val response = Gowaku.decodeSymmetric(jsonMsg, symmetricKey)
    return handleResponse<DecodedPayload>(response)
}
