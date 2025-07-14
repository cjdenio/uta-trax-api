proto:
	protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mproto/gtfs-realtime.proto=github.com/cjdenio/uta-trax-api/proto proto/*.proto
