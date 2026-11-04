package com.example.waku.messages

import com.example.waku.serializers.ByteArrayBase64Serializer
import kotlinx.serialization.Serializable

@Serializable
data class DecodedPayload(
    val pubkey: String?,
    val signature: String?,
    @Serializable(with = ByteArrayBase64Serializer::class) val data: ByteArray,
    @Serializable(with = ByteArrayBase64Serializer::class) val padding: ByteArray,
)
