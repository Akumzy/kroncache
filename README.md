# kroncache [WIP]
Have you ever wanted a cache management system that notifies you whenever a record expires automatically? like saving an upload records and clearing it uploads after some time if failed and so on? That's how *kroncache* was born since I couldn't find any software that does that.

*kroncache* provides API to perform simple CRUD operation for managing cache and as well notifying you once a record expires. It uses [badgerhold](https://github.com/timshannon/badgerhold) as database layer and Websocket for communication with *kroncache* clients.


# Usage 

See NodeJs client [kroncache-node](https://github.com/Akumzy/kroncache-node)
