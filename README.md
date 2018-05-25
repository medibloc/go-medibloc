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

## Running a blockchain node
### Test Configuration
* Genesis Block Configuration : [genesis.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/genesis.conf)
* Node Configuratoin : [node1.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node1.conf), [node2.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node2.conf), [node3.conf](https://github.com/medibloc/go-medibloc/blob/master/conf/test/3nodes/node3.conf)

### Running
```bash
$ cd $GOPATH/src/github.com/medibloc/go-medibloc

$ build/medi conf/test/3nodes/node1.conf
INFO[2018-05-18T06:55:28Z] Start medibloc...                             file=main.go func=main.runMedi line=48
INFO[2018-05-18T06:55:28Z] Setting up Medlet...                          file=medlet.go func="medlet.(*Medlet).Setup" line=92
INFO[2018-05-18T06:55:28Z] Set up Medlet.                                file=medlet.go func="medlet.(*Medlet).Setup" line=115
INFO[2018-05-18T06:55:28Z] Starting MedService...                        file=net_service.go func="net.(*MedService).Start" line=41
INFO[2018-05-18T06:55:28Z] Starting MedService Dispatcher...             file=dispatcher.go func="net.(*Dispatcher).Start" line=67
INFO[2018-05-18T06:55:28Z] Starting MedService Node...                   file=node.go func="net.(*Node).Start" line=81
INFO[2018-05-18T06:55:28Z] Starting MedService StreamManager...          file=stream_manager.go func="net.(*StreamManager).Start" line=56
INFO[2018-05-18T06:55:28Z] Starting MedService RouteTable Sync...        file=route_table.go func="net.(*RouteTable).Start" line=79
INFO[2018-05-18T06:55:28Z] Started MedService Node.                      file=node.go func="net.(*Node).Start" id=12D3KooWJkTULyR1Eb3Ps4dwm968fsGZXDDET4MNv8Xb6E29aSgX line=97 listening address="[/ip4/127.0.0.1/tcp/9900 /ip4/127.0.0.1/tcp/9910]"
INFO[2018-05-18T06:55:28Z] Started MedService.                           file=net_service.go func="net.(*MedService).Start" line=57
INFO[2018-05-18T06:55:28Z] GRPC Server is running...                     file=server.go func="rpc.(*Server).Start" line=39
INFO[2018-05-18T06:55:28Z] GRPC HTTP Gateway is running...               file=server.go func="rpc.(*Server).RunGateway" line=63
INFO[2018-05-18T06:55:28Z] Starting BlockManager...                      file=block_manager.go func="core.(*BlockManager).Start" line=96
INFO[2018-05-18T06:55:28Z] Starting TransactionManager...                file=transaction_manager.go func="core.(*TransactionManager).Start" line=46 size=262144
INFO[2018-05-18T06:55:28Z] Started Medlet.                               file=medlet.go func="medlet.(*Medlet).Start" line=145
INFO[2018-05-18T06:55:28Z] Started Dpos Mining.                          file=dpos.go func="dpos.(*Dpos).loop" line=415
INFO[2018-05-18T06:55:28Z] Started NewService Dispatcher.                file=dispatcher.go func="net.(*Dispatcher).loop" line=75
INFO[2018-05-18T06:55:28Z] Started MedService StreamManager.             file=stream_manager.go func="net.(*StreamManager).loop" line=128
INFO[2018-05-18T06:55:28Z] Started MedService RouteTable Sync.           file=route_table.go func="net.(*RouteTable).syncLoop" line=101
INFO[2018-05-18T06:55:30Z] New block is minted.                          block="<Height:2, Hash:53f8e720dc9636544807e0b06fd3fb1901952405dfecc75e212cdf0cfba63833, ParentHash:0000000000000000000000000000000000000000000000000000000000000000>" file=dpos.go func="dpos.(*Dpos).mintBlock" line=234 proposer=02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c
INFO[2018-05-18T06:55:30Z] Block pushed.                                 block="<Height:2, Hash:53f8e720dc9636544807e0b06fd3fb1901952405dfecc75e212cdf0cfba63833, ParentHash:0000000000000000000000000000000000000000000000000000000000000000>" file=block_manager.go func="core.(*BlockManager).push" lib="<Height:1, Hash:0000000000000000000000000000000000000000000000000000000000000000, ParentHash:0000000000000000000000000000000000000000000000000000000000000000>" line=235 tail="<Height:2, Hash:53f8e720dc9636544807e0b06fd3fb1901952405dfecc75e212cdf0cfba63833, ParentHash:0000000000000000000000000000000000000000000000000000000000000000>"
```

## Running a Local Testnet

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
  "chain_id": 1,
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
    * [Document](https://medjs.readthedocs.io/en/latest/)

## Testing
```bash
cd $GOPATH/src/github.com/medibloc/go-medibloc

make test
```

## License
```
Copyright (C) 2018  MediBloc

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
