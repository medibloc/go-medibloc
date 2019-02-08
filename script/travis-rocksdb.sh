#!/usr/bin/env bash

if [[ -e ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/lib/librocksdb.so ]]; then
    ls -R ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/
    exit
fi

git clone https://github.com/facebook/rocksdb.git ${TRAVIS_BUILD_DIR}/rocksdb
cd ${TRAVIS_BUILD_DIR}/rocksdb
git reset --hard v5.15.10
make shared_lib

mkdir -p ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/lib
mkdir -p ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/include

sudo cp --preserve=links ./librocksdb.* ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/lib/
sudo cp -r ./include/rocksdb/ ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/include/
ls -R ${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10/
