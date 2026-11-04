package com.example.waku

import kotlinx.serialization.Serializable

@Serializable
data class Peer(
    val peerID: String?,
    val connected: Boolean,
    val protocols: List<String>,
    val addrs: List<String>,
)