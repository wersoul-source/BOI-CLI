#!/bin/bash
for f in bin/boi-*; do
    sha256sum "$f" >> checksums.txt
done
