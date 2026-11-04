package com.example.waku.store

import kotlinx.serialization.Serializable

@Serializable
data class ContentFilter(val contentTopic: String)