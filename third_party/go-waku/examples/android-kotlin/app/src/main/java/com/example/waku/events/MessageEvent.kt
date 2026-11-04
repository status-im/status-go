package com.example.waku.events

import kotlinx.serialization.Serializable

@Serializable
data class MessageEvent(override val type: EventType, val event: MessageEventData) : Event
