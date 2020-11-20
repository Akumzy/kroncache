# kroncache [WIP]
Have you ever wanted a cache management system that notifies you whenever a record expires automatically? like saving upload records and clearing it uploads after some time if failed and so on? That's how *kroncache* was born since I couldn't find any software that does that.

*kroncache* provides API to perform simple CRUD operation for managing cache and as well notifying you once a record expires. It uses [badgerhold](https://github.com/timshannon/badgerhold) as database layer and Websocket for communication with *kroncache* clients.

# Setup
Download the tar file that matches your platform and architecture [here](https://github.com/Akumzy/kroncache/releases)
example for Linux amd64 `kroncache_x.x.x_linux_amd64.tar.gz`
then unzip into your desired directory

```sh
$ tar -xf kroncache_x.x.x_Linux_64-bit.tar.gz
```
Then run `kroncache` with the default port `5093`
```sh
$ cd kroncache_x.x.x_Linux_64-bit
$ ./kroncache
```
or change the port by setting the PORT environment variable
```sh
$ PORT=5599 ./kroncache
```

# Clients 

- NodeJs: [kroncache-node](https://github.com/Akumzy/kroncache-node)
