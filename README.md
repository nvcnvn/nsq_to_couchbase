# nsq_to_couchbase

Similar to nsq_tail (infact alot of copy/paste from that). This tool is a consumer that catch and save message in Couchbase


### Build step

Requirement:
 * Install Go (https://golang.org/doc/install#install)
 * Install `gb` tool (https://getgb.io/)

Copy to terminal:
```
git clone https://github.com/nvcnvn/nsq_to_couchbase.git
cd nsq_to_couchbase
gb build
```
Then you will find an excutable in `bin` folder name `cmd`.

### Usage

Arguments:
 * connStr: Couchbase connection string - required (example: "couchbase://192.168.46.10")
 * bucket: Couchbase bucket - required
 * bucketPwd: Couchbase bucket password - optional
 * topic: NSQ topic - required
 * channel: NSQ channel - optional
 * max-in-flight: max number of messages to allow in flight, default 200
 * nsqd-tcp-address: nsqd TCP address (may be given multiple times)
 * lookupd-http-address: lookupd HTTP address (may be given multiple times)
 
 
 * json: determine if message encoded with JSO, default true
 * key-fields: if message encoded with JSON, use this list to lookup the key field (may be given multiple times)

Example:
```
./bin/cmd --topic=example --lookupd-http-address=192.168.46.10:4161 --bucket=example --connStr=couchbase://192.168.46.10

```

#### Behavior of `json` and `key-fields` arguments

In short, by default we expect that message payload encoded with JSON and have a field `messageId` contain string type data can be use for Couchbase document key.

You can use `key-fields` argument to specific the field name intead of `messageId`, if nothing given and the default field not found, we will generate an uuid for document key.

 * If message not encoded with JSON we generate an uuid for Couchbase document key.
 * If message encoded with JSON
   * If a list of `key-fields` given
     * If found a field on the list with string data, use that data for Couchbase document key
     * If not found, generate an uuid for Couchbase docuemnt key
   * If no `key-fields` given
     * If a field name `messageId` with string data, use that data for Couchbase document key
     * If not found, generate an uuid for CouchBase document key

### Licence: MIT
