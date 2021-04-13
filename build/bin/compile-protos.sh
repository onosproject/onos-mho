#!/bin/sh

proto_imports=".:${GOPATH}/src/github.com/gogo/protobuf/protobuf:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src/github.com/envoyproxy/protoc-gen-validate:${GOPATH}/src":"${GOPATH}/src/github.com/onosproject/onos-pci/api"

# samples below
# admin.proto cannot be generated with fast marshaler/unmarshaler because it uses gnmi.ModelData
#protoc -I=$proto_imports --doc_out=docs/api  --doc_opt=markdown,admin.md  --gogo_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-e2t/api/admin,plugins=grpc:. api/admin/v1/*.proto
#protoc -I=$proto_imports --doc_out=docs/api  --doc_opt=markdown,diags.md --gogo_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mconfig/admin/admin.proto=github.com/onosproject/onos-e2t/api/admin,import_path=github.com/onosproject/onos-e2t/api/diags,plugins=grpc:. api/diags/*.proto

cp -r github.com/onosproject/onos-pci/* .
rm -rf github.com
