Topics flags
============

This module provides 2 helpers to parse collections of topics.

1. List of topics, such as:

```
statusd -topic.register=whisper -topic.register=les
```

2. List of topics with limits per topic. Main use case is to define per-protocol
peer limits:

```
statusd -topic.search=whisper=7,9 -topic.search=mailserver=1,1 -topic.search=les=1,2
```