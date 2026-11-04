package com.example.waku.events

interface EventHandler {
    fun handleEvent(evt: Event)
}
