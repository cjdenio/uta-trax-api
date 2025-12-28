#!/bin/bash

set -eo pipefail

curl --output gtfs.zip --location https://gtfsfeed.rideuta.com/GTFS.zip
gtfs-import --gtfsPath gtfs.zip --sqlitePath uta-gtfs.db