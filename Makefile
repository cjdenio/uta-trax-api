proto: proto/schema.pb.go proto/gtfs-realtime.pb.go

proto/gtfs-realtime.pb.go: proto/gtfs-realtime.proto
	protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mproto/gtfs-realtime.proto=github.com/cjdenio/uta-trax-api/proto $<

proto/schema.pb.go: proto/schema.proto
	protoc --go_out=. --go_opt=paths=source_relative $<
