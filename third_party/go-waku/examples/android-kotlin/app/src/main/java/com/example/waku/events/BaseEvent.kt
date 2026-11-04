package com.example.waku.events

import kotlinx.serialization.Serializable

@Serializable
data class BaseEvent(override val type: EventType) : Event
