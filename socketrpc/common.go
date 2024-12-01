package socketrpc

var (
	socketAddr        = "/tmp/immich-sync.sock"
	CmdStatus         = byte(0x1)
	CmdScanAll        = byte(0x2)
	CmdUploadFile     = byte(0x5)
	CmdAddDir         = byte(0x10)
	CmdRmDir          = byte(0x11)
	CmdCreateAlbum    = byte(0x20)
	CmdShowAlbum      = byte(0x21)
	CmdAddAlbum       = byte(0x22)
	CmdExit           = byte(0xFF)
	ErrOk             = byte(0x0)
	ErrGeneric        = byte(0x1)
	ErrUnknownCmd     = byte(0x2)
	ErrUnsupportedCmd = byte(0x3)
	ErrWrongArgs      = byte(0x4)
	ErrFileNotFound   = byte(0x5)
)
