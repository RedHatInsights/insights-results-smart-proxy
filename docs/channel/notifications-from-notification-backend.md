---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `notifications-from-notification-backend`



## Type

* E-mail, web hook message etc.



## Description

Templating is done in *Notification Backend* (the event itself contains
basically a set of key+values pairs) and messages are sent to customers/users
via pre-defined channels like webhooks, e-mails, Slack (planned for the future
etc.)
