#!/bin/sh
# 데모 GIF 재생성. 레포 루트에서: demo/record.sh
# crc를 demo/bin(가짜 claude와 같은 PATH)으로 새로 빌드하고 VHS로 굽는다.
set -e
cd "$(dirname "$0")/.."
go build -o demo/bin/crc .
vhs demo/demo.tape
rm -rf demo/.state demo/bin/crc /tmp/crc-demo
echo "→ demo/demo.gif"
