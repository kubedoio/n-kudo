module github.com/n-kudo/n-kudo-edge

go 1.23.0

toolchain go1.23.4

require go.etcd.io/bbolt v1.3.11

replace go.etcd.io/bbolt => ./third_party/go.etcd.io/bbolt

replace golang.org/x/sys => ./third_party/golang.org/x/sys

require golang.org/x/sys v0.34.0 // indirect
