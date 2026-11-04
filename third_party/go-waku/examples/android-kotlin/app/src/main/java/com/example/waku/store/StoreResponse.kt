package com.example.waku.store

import com.example.waku.messages.Message
import kotlinx.serialization.Serializable

@Serializable
data class StoreResponse(val messages: List<Message>, val pagingOptions: PagingOptions?)