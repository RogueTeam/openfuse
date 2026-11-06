#!/bin/sh

build_example() (
    cd examples/$1
    GOOS=openbsd go build -o ../../build/ .
)

for example in $(ls examples); do
    if [ "$example" != "README.md" ]; then
        build_example $example
    fi
done

tar -cf build.tar.gz build/
