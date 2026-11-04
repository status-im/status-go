package com.example.waku

import kotlinx.serialization.Serializable

@Serializable
data class Config(
    var host: String? = null,
    var result: Int? = null,
    var advertiseAddr: String? = null,
    var nodeKey: String? = null,
    var keepAliveInterval: Int? = null,
    var relay: Boolean? = null,
    var minPeersToPublish: Int? = null
)
