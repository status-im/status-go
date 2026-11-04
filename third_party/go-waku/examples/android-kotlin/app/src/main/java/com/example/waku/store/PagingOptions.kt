package com.example.waku.store

import kotlinx.serialization.Serializable

@Serializable
data class PagingOptions(val pageSize: Int, val cursor: Cursor?, val forward: Boolean)