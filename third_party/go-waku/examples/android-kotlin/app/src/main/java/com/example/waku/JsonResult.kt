package com.example.waku

import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json

@Serializable
data class JsonResult<T>(val error: String? = null, val result: T? = null)

inline fun <reified T> handleResponse(response: String): T {
    val jsonResult = Json.decodeFromString<JsonResult<T>>(response)

    if (jsonResult.error != null)
        throw Exception(jsonResult.error)

    if (jsonResult.result == null)
        throw Exception("no result in response")

    return jsonResult.result
}

inline fun handleResponse(response: String) {
    val jsonResult = Json.decodeFromString<JsonResult<String>>(response)

    if (jsonResult.error != null)
        throw Exception(jsonResult.error)
}
