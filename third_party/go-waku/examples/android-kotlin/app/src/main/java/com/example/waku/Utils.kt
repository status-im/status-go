package com.example.waku

import gowaku.Gowaku

/**
 * Get default pubsub topic
 * @return Default pubsub topic used for exchanging waku messages defined in RFC 10
 */
fun DefaultPubsubTopic(): String {
    return Gowaku.defaultPubsubTopic()
}

/**
 * Create a content topic string
 * @param applicationName
 * @param applicationVersion
 * @param contentTopicName
 * @param encoding
 * @return Content topic string according to RFC 23
 */
fun ContentTopic(applicationName: String, applicationVersion: Long, contentTopicName: String, encoding: String): String{
    return Gowaku.contentTopic(applicationName, applicationVersion, contentTopicName, encoding)
}

