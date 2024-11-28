package socketrpc

var (
	socketAddr        = "/tmp/immich-sync.sock"
	CmdStatus         = byte(0x1)
	CmdScanAll        = byte(0x2)
	CmdAddDir         = byte(0x3)
	CmdRmDir          = byte(0x4)
	CmdUploadFile     = byte(0x5)
	CmdExit           = byte(0xFF)
	ErrOk             = byte(0x0)
	ErrGeneric        = byte(0x1)
	ErrUnknownCmd     = byte(0x2)
	ErrUnsupportedCmd = byte(0x3)
	ErrWrongArgs      = byte(0x4)
	ErrFileNotFound   = byte(0x5)
)
