//go:build cgo && !nocgo

package cgo

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/../../../ftml-capi/target/release -lftml_capi
#cgo windows LDFLAGS: -lws2_32 -luserenv -lbcrypt -lntdll -ladvapi32 -lsynchronization
*/
import "C"
