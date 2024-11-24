package socketrpc

var (
	socketAddr        = "/tmp/immich-sync.sock"
	CmdScanAll        = byte(0x1)
	CmdAddDir         = byte(0x2)
	ErrOk             = byte(0x0)
	ErrGeneric        = byte(0x1)
	ErrUnknownCmd     = byte(0x2)
	ErrUnsupportedCmd = byte(0x3)
)
