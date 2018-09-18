# MediBloc Testnet

## Explorer
* Testnet Explorer
    * https://explorer.medibloc.org
* Source Repo
    * https://github.com/medibloc/explorer

## Wallet
* Web Wallet
    * https://wallet.medibloc.org/

## Claim Testnet Coin
* [Claim](https://goo.gl/forms/UjbYcGzA9YvjiEiI3)

## API Endpoint
* Testnet API Endpoint
```
https://node.medibloc.org
```
* Using medjs library
    * https://github.com/medibloc/medjs
```js
var Medjs = require('medjs');
var medjs = Medjs.init(['https://node.medibloc.org']);
```

## Running a Testnet Node
* Public Node Config : [testnet/public.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/testnet/public.conf)
* Genesis Config : [testnet/genesis.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/testnet/genesis.conf)
* Seed Address
```
"/ip4/13.124.155.229/tcp/9900/ipfs/12D3KooWFVij7DgP88xxGKsRwKTLYBv1F7v4QaXqNBxpi99iVpfk"
```
* Running
```
cd $GOPATH/src/github.com/medibloc/go-medibloc

# Build a medi binary
make build

# Run
build/medi conf/testnet/public.conf
```
