module github.com/tofchaliss/themis

go 1.24

require github.com/spf13/cobra v1.9.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
)

require github.com/tofchaliss/themis-app v0.0.0

replace github.com/tofchaliss/themis-app => ../themis
