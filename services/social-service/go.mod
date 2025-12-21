module metargb/social-service

go 1.24.0

toolchain go1.24.3

require (
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require metargb/shared v0.0.0

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
)

replace metargb/shared => ../../shared
