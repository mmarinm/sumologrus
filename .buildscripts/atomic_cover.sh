#!/bin/bash

set -e
echo "" > ../tmp/coverage.txt

cd ..

go test -race -coverprofile=profile.out -covermode=atomic
if [ -f profile.out ]; then
	cat profile.out >> ./tmp/coverage.txt
	rm profile.out
fi
