package com.example.waku.store

import com.example.waku.DefaultPubsubTopic
import kotlinx.serialization.Serializable

@Serializable
data class StoreQuery(
    var pubsubTopic: String? = DefaultPubsubTopic(),
    var startTime: Long? = null,
    var endTime: Long? = null,
    var contentFilters: List<ContentFilter>?,
    var pagingOptions: PagingOptions?
)