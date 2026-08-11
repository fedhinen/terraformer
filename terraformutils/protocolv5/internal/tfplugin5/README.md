# Canonical Terraform Plugin Protocol v5.10 stubs

These files are copied without modification from
`github.com/hashicorp/terraform-plugin-go@v0.31.0/tfprotov5/internal/tfplugin5`.
They are used privately by Terraformer's protocol-v5 client and do not import
the upstream Go `internal` package.

Upstream license: Mozilla Public License 2.0. The generated files retain their
upstream copyright and generation headers. See the repository's third-party
license inventory before distribution.

Expected SHA-256 checksums:

```text
3d9975526164cce8479755469220d54e7400e461b24a21997ccccef79e3f1e90  tfplugin5.proto
b36b11ab6ea0ebe8ecbdad93a26b9026cf7298460d6182120a6ef6473290d083  tfplugin5.pb.go
1b572a4c9eb0edc01c3c609f86c0d270509f886418d6c9fdaa0fdd3751027c7d  tfplugin5_grpc.pb.go
```

Update procedure:

1. update the pinned `terraform-plugin-go` module;
2. copy the three files from the module version recorded above;
3. run `shasum -a 256` and update this file;
4. run `go test ./terraformutils/protocolv5/...`; and
5. review the protobuf and generated API diff before committing.
