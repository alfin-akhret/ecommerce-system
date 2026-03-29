# Development Notes

**Development notes:**

`CartRepository` unit testing (actually this is mini integration test now) is using ephemeral local redis server. Make sure to have redis binary installed in ci/dev environment. 
Other options (if we don't want to make an actual call to redis):
1. update the code (and the test), make repo to use interface for redis. This can test the repo but cannot make sure if the data really stored in redis.
2. use in-memory mock servers (eg: miniredis, minisentinel)

TODO: 
- ~~set cart TTL: 24 hours~~
- ~~change all price to int64, including in DB~~
- ~~add engineering notes: topic: price data type~~
- CreateOrder flow , integrate with cart and checkout.