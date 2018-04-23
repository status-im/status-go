This document aims to group all general guidelines to have in mind when you work with status-go.


### /geth/ path

Some status-go code lives inside `/geth` path, however this path no longer makes sense and we should get rid of it. Please don't store your new packages on `/geth` use the root path of the repo instead.


### Server as standalone app

`status-go` is used both for mobile devices and also on our nodes. We should try to split this in order to not overload single repository.
