// +build ignore

package scard

/*
#include <PCSC/winscard.h>
#include <PCSC/reader.h>
*/
import "C"

type Attrib uint32

const (
	AttrVendorName           Attrib = C.SCARD_ATTR_VENDOR_NAME
	AttrVendorIfdType        Attrib = C.SCARD_ATTR_VENDOR_IFD_TYPE
	AttrVendorIfdVersion     Attrib = C.SCARD_ATTR_VENDOR_IFD_VERSION
	AttrVendorIfdSerialNo    Attrib = C.SCARD_ATTR_VENDOR_IFD_SERIAL_NO
	AttrChannelId            Attrib = C.SCARD_ATTR_CHANNEL_ID
	AttrAsyncProtocolTypes   Attrib = C.SCARD_ATTR_ASYNC_PROTOCOL_TYPES
	AttrDefaultClk           Attrib = C.SCARD_ATTR_DEFAULT_CLK
	AttrMaxClk               Attrib = C.SCARD_ATTR_MAX_CLK
	AttrDefaultDataRate      Attrib = C.SCARD_ATTR_DEFAULT_DATA_RATE
	AttrMaxDataRate          Attrib = C.SCARD_ATTR_MAX_DATA_RATE
	AttrMaxIfsd              Attrib = C.SCARD_ATTR_MAX_IFSD
	AttrSyncProtocolTypes    Attrib = C.SCARD_ATTR_SYNC_PROTOCOL_TYPES
	AttrPowerMgmtSupport     Attrib = C.SCARD_ATTR_POWER_MGMT_SUPPORT
	AttrUserToCardAuthDevice Attrib = C.SCARD_ATTR_USER_TO_CARD_AUTH_DEVICE
	AttrUserAuthInputDevice  Attrib = C.SCARD_ATTR_USER_AUTH_INPUT_DEVICE
	AttrCharacteristics      Attrib = C.SCARD_ATTR_CHARACTERISTICS
	AttrCurrentProtocolType  Attrib = C.SCARD_ATTR_CURRENT_PROTOCOL_TYPE
	AttrCurrentClk           Attrib = C.SCARD_ATTR_CURRENT_CLK
	AttrCurrentF             Attrib = C.SCARD_ATTR_CURRENT_F
	AttrCurrentD             Attrib = C.SCARD_ATTR_CURRENT_D
	AttrCurrentN             Attrib = C.SCARD_ATTR_CURRENT_N
	AttrCurrentW             Attrib = C.SCARD_ATTR_CURRENT_W
	AttrCurrentIfsc          Attrib = C.SCARD_ATTR_CURRENT_IFSC
	AttrCurrentIfsd          Attrib = C.SCARD_ATTR_CURRENT_IFSD
	AttrCurrentBwt           Attrib = C.SCARD_ATTR_CURRENT_BWT
	AttrCurrentCwt           Attrib = C.SCARD_ATTR_CURRENT_CWT
	AttrCurrentEbcEncoding   Attrib = C.SCARD_ATTR_CURRENT_EBC_ENCODING
	AttrExtendedBwt          Attrib = C.SCARD_ATTR_EXTENDED_BWT
	AttrIccPresence          Attrib = C.SCARD_ATTR_ICC_PRESENCE
	AttrIccInterfaceStatus   Attrib = C.SCARD_ATTR_ICC_INTERFACE_STATUS
	AttrCurrentIoState       Attrib = C.SCARD_ATTR_CURRENT_IO_STATE
	AttrAtrString            Attrib = C.SCARD_ATTR_ATR_STRING
	AttrIccTypePerAtr        Attrib = C.SCARD_ATTR_ICC_TYPE_PER_ATR
	AttrEscReset             Attrib = C.SCARD_ATTR_ESC_RESET
	AttrEscCancel            Attrib = C.SCARD_ATTR_ESC_CANCEL
	AttrEscAuthrequest       Attrib = C.SCARD_ATTR_ESC_AUTHREQUEST
	AttrMaxinput             Attrib = C.SCARD_ATTR_MAXINPUT
	AttrDeviceUnit           Attrib = C.SCARD_ATTR_DEVICE_UNIT
	AttrDeviceInUse          Attrib = C.SCARD_ATTR_DEVICE_IN_USE
	AttrDeviceFriendlyName   Attrib = C.SCARD_ATTR_DEVICE_FRIENDLY_NAME
	AttrDeviceSystemName     Attrib = C.SCARD_ATTR_DEVICE_SYSTEM_NAME
	AttrSupressT1IfsRequest  Attrib = C.SCARD_ATTR_SUPRESS_T1_IFS_REQUEST
)

type Error uint32

const (
	ErrSuccess                Error = C.SCARD_S_SUCCESS
	ErrInternalError          Error = C.SCARD_F_INTERNAL_ERROR
	ErrCancelled              Error = C.SCARD_E_CANCELLED
	ErrInvalidHandle          Error = C.SCARD_E_INVALID_HANDLE
	ErrInvalidParameter       Error = C.SCARD_E_INVALID_PARAMETER
	ErrInvalidTarget          Error = C.SCARD_E_INVALID_TARGET
	ErrNoMemory               Error = C.SCARD_E_NO_MEMORY
	ErrWaitedTooLong          Error = C.SCARD_F_WAITED_TOO_LONG
	ErrInsufficientBuffer     Error = C.SCARD_E_INSUFFICIENT_BUFFER
	ErrUnknownReader          Error = C.SCARD_E_UNKNOWN_READER
	ErrTimeout                Error = C.SCARD_E_TIMEOUT
	ErrSharingViolation       Error = C.SCARD_E_SHARING_VIOLATION
	ErrNoSmartcard            Error = C.SCARD_E_NO_SMARTCARD
	ErrUnknownCard            Error = C.SCARD_E_UNKNOWN_CARD
	ErrCantDispose            Error = C.SCARD_E_CANT_DISPOSE
	ErrProtoMismatch          Error = C.SCARD_E_PROTO_MISMATCH
	ErrNotReady               Error = C.SCARD_E_NOT_READY
	ErrInvalidValue           Error = C.SCARD_E_INVALID_VALUE
	ErrSystemCancelled        Error = C.SCARD_E_SYSTEM_CANCELLED
	ErrCommError              Error = C.SCARD_F_COMM_ERROR
	ErrUnknownError           Error = C.SCARD_F_UNKNOWN_ERROR
	ErrInvalidAtr             Error = C.SCARD_E_INVALID_ATR
	ErrNotTransacted          Error = C.SCARD_E_NOT_TRANSACTED
	ErrReaderUnavailable      Error = C.SCARD_E_READER_UNAVAILABLE
	ErrShutdown               Error = C.SCARD_P_SHUTDOWN
	ErrPciTooSmall            Error = C.SCARD_E_PCI_TOO_SMALL
	ErrReaderUnsupported      Error = C.SCARD_E_READER_UNSUPPORTED
	ErrDuplicateReader        Error = C.SCARD_E_DUPLICATE_READER
	ErrCardUnsupported        Error = C.SCARD_E_CARD_UNSUPPORTED
	ErrNoService              Error = C.SCARD_E_NO_SERVICE
	ErrServiceStopped         Error = C.SCARD_E_SERVICE_STOPPED
	ErrUnexpected             Error = C.SCARD_E_UNEXPECTED
	ErrUnsupportedFeature     Error = C.SCARD_E_UNSUPPORTED_FEATURE
	ErrIccInstallation        Error = C.SCARD_E_ICC_INSTALLATION
	ErrIccCreateorder         Error = C.SCARD_E_ICC_CREATEORDER
	ErrFileNotFound           Error = C.SCARD_E_FILE_NOT_FOUND
	ErrNoDir                  Error = C.SCARD_E_NO_DIR
	ErrNoFile                 Error = C.SCARD_E_NO_FILE
	ErrNoAccess               Error = C.SCARD_E_NO_ACCESS
	ErrWriteTooMany           Error = C.SCARD_E_WRITE_TOO_MANY
	ErrBadSeek                Error = C.SCARD_E_BAD_SEEK
	ErrInvalidChv             Error = C.SCARD_E_INVALID_CHV
	ErrUnknownResMng          Error = C.SCARD_E_UNKNOWN_RES_MNG
	ErrNoSuchCertificate      Error = C.SCARD_E_NO_SUCH_CERTIFICATE
	ErrCertificateUnavailable Error = C.SCARD_E_CERTIFICATE_UNAVAILABLE
	ErrNoReadersAvailable     Error = C.SCARD_E_NO_READERS_AVAILABLE
	ErrCommDataLost           Error = C.SCARD_E_COMM_DATA_LOST
	ErrNoKeyContainer         Error = C.SCARD_E_NO_KEY_CONTAINER
	ErrServerTooBusy          Error = C.SCARD_E_SERVER_TOO_BUSY
	ErrUnsupportedCard        Error = C.SCARD_W_UNSUPPORTED_CARD
	ErrUnresponsiveCard       Error = C.SCARD_W_UNRESPONSIVE_CARD
	ErrUnpoweredCard          Error = C.SCARD_W_UNPOWERED_CARD
	ErrResetCard              Error = C.SCARD_W_RESET_CARD
	ErrRemovedCard            Error = C.SCARD_W_REMOVED_CARD
	ErrSecurityViolation      Error = C.SCARD_W_SECURITY_VIOLATION
	ErrWrongChv               Error = C.SCARD_W_WRONG_CHV
	ErrChvBlocked             Error = C.SCARD_W_CHV_BLOCKED
	ErrEof                    Error = C.SCARD_W_EOF
	ErrCancelledByUser        Error = C.SCARD_W_CANCELLED_BY_USER
	ErrCardNotAuthenticated   Error = C.SCARD_W_CARD_NOT_AUTHENTICATED
)

type Protocol uint32

const (
	ProtocolUndefined Protocol = C.SCARD_PROTOCOL_UNDEFINED
	ProtocolT0        Protocol = C.SCARD_PROTOCOL_T0
	ProtocolT1        Protocol = C.SCARD_PROTOCOL_T1
	ProtocolAny       Protocol = ProtocolT0 | ProtocolT1
)

type ShareMode uint32

const (
	ShareExclusive ShareMode = C.SCARD_SHARE_EXCLUSIVE
	ShareShared    ShareMode = C.SCARD_SHARE_SHARED
	ShareDirect    ShareMode = C.SCARD_SHARE_DIRECT
)

type Disposition uint32

const (
	LeaveCard   Disposition = C.SCARD_LEAVE_CARD
	ResetCard   Disposition = C.SCARD_RESET_CARD
	UnpowerCard Disposition = C.SCARD_UNPOWER_CARD
	EjectCard   Disposition = C.SCARD_EJECT_CARD
)

type Scope uint32

const (
	ScopeUser     Scope = C.SCARD_SCOPE_USER
	ScopeTerminal Scope = C.SCARD_SCOPE_TERMINAL
	ScopeSystem   Scope = C.SCARD_SCOPE_SYSTEM
)

type State uint32

const (
	Unknown    State = C.SCARD_UNKNOWN
	Absent     State = C.SCARD_ABSENT
	Present    State = C.SCARD_PRESENT
	Swallowed  State = C.SCARD_SWALLOWED
	Powered    State = C.SCARD_POWERED
	Negotiable State = C.SCARD_NEGOTIABLE
	Specific   State = C.SCARD_SPECIFIC
)

type StateFlag uint32

const (
	StateUnaware     StateFlag = C.SCARD_STATE_UNAWARE
	StateIgnore      StateFlag = C.SCARD_STATE_IGNORE
	StateChanged     StateFlag = C.SCARD_STATE_CHANGED
	StateUnknown     StateFlag = C.SCARD_STATE_UNKNOWN
	StateUnavailable StateFlag = C.SCARD_STATE_UNAVAILABLE
	StateEmpty       StateFlag = C.SCARD_STATE_EMPTY
	StatePresent     StateFlag = C.SCARD_STATE_PRESENT
	StateAtrmatch    StateFlag = C.SCARD_STATE_ATRMATCH
	StateExclusive   StateFlag = C.SCARD_STATE_EXCLUSIVE
	StateInuse       StateFlag = C.SCARD_STATE_INUSE
	StateMute        StateFlag = C.SCARD_STATE_MUTE
	StateUnpowered   StateFlag = C.SCARD_STATE_UNPOWERED
)

const (
	maxBufferSize         = C.MAX_BUFFER_SIZE
	maxBufferSizeExtended = C.MAX_BUFFER_SIZE_EXTENDED
	maxReadername         = C.MAX_READERNAME
	maxAtrSize            = C.MAX_ATR_SIZE
)

const (
	infiniteTimeout = C.INFINITE
)
