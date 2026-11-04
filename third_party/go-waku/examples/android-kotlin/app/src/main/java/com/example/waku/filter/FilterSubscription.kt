package com.example.waku.filter

import kotlinx.serialization.Serializable

@Serializable
data class FilterSubscription(
    var contentFilters: List<ContentFilter>,
    var topic: String?,
)