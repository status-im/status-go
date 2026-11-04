package com.example.waku.events

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
enum class EventType {
    @SerialName("unknown")
    Unknown,
    @SerialName("message")
    Message
}
