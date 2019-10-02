package socks5

import (
	"errors"
)

var (
	ErrSocksVersionNotSupport   = errors.New("socks version not support")
	ErrNotSupportClientMethod   = errors.New("not support client method")
	ErrAddrFormatInvalid        = errors.New("address format invalid")
	ErrSocksServerFailure       = errors.New("socks server failure")
	ErrRemoteConnectionNotAllow = errors.New("remote connection not allow")
	ErrRemoteNetworkUnreachable = errors.New("remote network unreachable")
	ErrRemoteHostUnreachable    = errors.New("remote host unreachable")
	ErrRemoteConnectionRefused  = errors.New("remote connection refused")
	ErrRemoteTTLExpired         = errors.New("remote ttl expired")
	ErrCommandNotSupported      = errors.New("command not supported")
	ErrAddressTypeNotSupported  = errors.New("address type not supported")
)
