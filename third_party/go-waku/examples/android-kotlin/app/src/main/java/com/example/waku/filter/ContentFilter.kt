package com.example.waku.filter

import kotlinx.serialization.Serializable

@Serializable
data class ContentFilter(val contentTopic: String)