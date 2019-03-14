#!/bin/bash

## Env
ROCKSDB=$HOME/tmp/rocksdb
ROCKSDB_TARGET=v5.15.10
GO_MEDIBLOC_TARGET=develop
BINDIR=$HOME/bin
export GOROOT=$HOME/bin/go
export GOPATH=$HOME/go

## Directories
mkdir -p $BINDIR
mkdir -p $ROCKSDB
mkdir -p $GOPATH

## Dependencies
sudo apt-get update -y && sudo apt-get install -y libsnappy-dev zlib1g-dev libbz2-dev g++ make

## Git
sudo apt-get install -y git

## Golang
cd $BINDIR
wget https://dl.google.com/go/go1.11.5.linux-amd64.tar.gz
tar xvf go1.11.5.linux-amd64.tar.gz
mv go go1.11.5
ln -s go1.11.5 go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH
go get -u github.com/golang/dep/cmd/dep

# #RocksDB
git clone https://github.com/facebook/rocksdb.git $ROCKSDB
cd $ROCKSDB
git reset --hard $ROCKSDB_TARGET
make shared_lib
sudo cp --preserve=links ./librocksdb.* /usr/lib/
sudo cp -r ./include/rocksdb/ /usr/include

## go-medibloc
go get -u github.com/medibloc/go-medibloc
cd $GOPATH/src/github.com/medibloc/go-medibloc
git reset --hard $GO_MEDIBLOC_TARGET
make dep
make build
cp build/medi $BINDIR
