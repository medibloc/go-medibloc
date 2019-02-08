#!/usr/bin/env bash

CACHE_DIR="${TRAVIS_BUILD_DIR}/cache/rocksdb/v5.15.10"

if [[ -e ${CACHE_DIR}/lib/librocksdb.so ]]; then
    ls -R ${CACHE_DIR}/
    sudo cp -r --preserve=links ${CACHE_DIR}/lib/librocksdb.* /usr/lib/
    sudo cp -r ${CACHE_DIR}/include/* /usr/include/
    exit
fi

git clone https://github.com/facebook/rocksdb.git ${TRAVIS_BUILD_DIR}/rocksdb
cd ${TRAVIS_BUILD_DIR}/rocksdb
git reset --hard v5.15.10
make shared_lib

mkdir -p ${CACHE_DIR}/lib
mkdir -p ${CACHE_DIR}/include

sudo cp --preserve=links ./librocksdb.* ${CACHE_DIR}/lib/
sudo cp -r ./include/rocksdb/ ${CACHE_DIR}/include/
ls -R ${CACHE_DIR}/
sudo cp -r --preserve=links ${CACHE_DIR}/lib/librocksdb.* /usr/lib/
sudo cp -r ${CACHE_DIR}/include/* /usr/include/
