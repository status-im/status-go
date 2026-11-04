package com.example.waku.events

import com.example.waku.messages.Message
import kotlinx.serialization.Serializable

@Serializable
data class MessageEventData(val messageID: String, val pubsubTopic: String, val wakuMessage: Message)
