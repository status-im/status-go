package com.example.waku.store

import com.example.waku.DefaultPubsubTopic
import com.example.waku.serializers.ByteArrayBase64Serializer
import kotlinx.serialization.Serializable

@Serializable
data class Cursor(
    @Serializable(with = ByteArrayBase64Serializer::class) val digest: ByteArray,
    val pubsubTopic: String = DefaultPubsubTopic(),
    val receiverTime: Long = 0,
    val senderTime: Long = 0
)
