# Go MediBloc
Official implementation of the medibloc blockchain.

## How to Build
### Build Requirements
* Golang v1.10 or higher
    * [Golang.org](Golang.org)
* dep
```bash
go get -u github.com/golang/dep/cmd/dep
```
* goimports
```bash
go get -u golang.org/x/tools/cmd/goimports
```
* golint
```bash
go get -u golang.org/x/lint/golint
```

### Building
```bash
# Go get medibloc repository.
go get -u github.com/medibloc/go-medibloc

# Change directory to medibloc repository.
cd $GOPATH/src/github.com/medibloc/go-medibloc

# Download dependencies.
make dep

# Build
make build
```

## Running a Local Testnet
### Configuration
* Genesis Block Configuration : [genesis.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/genesis.conf)
* Node Configuratoin : [node1.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node1.conf), [node2.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node2.conf), [node3.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node3.conf)

### Running
```bash
cd $GOPATH/src/github.com/medibloc/go-medibloc

# Run the first node
nohup build/medi conf/test/3nodes/node1.conf &> /dev/null &

# Run the second node
nohup build/medi conf/test/3nodes/node2.conf &> /dev/null &

# Run the third node
nohup build/medi conf/test/3nodes/node3.conf &> /dev/null &
```

### Endpoints of the Local Testnet
* RPC : `localhost:9720`, `localhost:9820`, `localhost:9920`
* HTTP : `localhost:9721`, `localhost:9821`, `localhost:9921`

### Check the running nodes.
```bash
# Get blockchain state
$ curl localhost:9921/v1/user/medstate | jq .
{
  "chain_id": 1010,
  "tail": "9964d1dfde18bdae9ff2122be87222a3cc0f5c665f22f7359b113310d2c4a4f5",
  "height": "2942"
}

# Get account state
$ curl localhost:9921/v1/user/accountstate?address=02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c
{"balance":"1000000000"}

# View each node's logs
$ tail -f logs/log1/medibloc.log
$ tail -f logs/log2/medibloc.log
$ tail -f logs/log3/medibloc.log
```

### Demo
* Demo video link (TODO)

## Library
* medjs
    * [Github](https://github.com/medibloc/medjs)
    * Document (TODO)
* Blockchain API
    * Doucment (TODO)

## Testing
```bash
cd $GOPATH/src/github.com/medibloc/go-medibloc

make test
```

## License (TODO)