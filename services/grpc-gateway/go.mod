module metargb/grpc-gateway

go 1.24.0

toolchain go1.24.3

require (
	github.com/joho/godotenv v1.5.1
	google.golang.org/grpc v1.76.0
	metargb/shared v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace metargb/shared => /workspace/metargb/shared
