package common

const WalletAccountDefaultName = "Account 1"

// AddressBytesLength is the expected length of the address in bytes
const AddressBytesLength = 20

// AddressHexLength is the expected length of the address in hex (with 0x prefix)
const AddressHexLength = 2*AddressBytesLength + 2

const PathMaster = "m"

const PathEIP1581Root = "m/43'/60'/1581'"
const PathEIP1581Chat = PathEIP1581Root + "/0'/0"
const PathEIP1581Encryption = PathEIP1581Root + "/1'/0"

const WalletPath = "m/44'"
const PathWalletRoot = "m/44'/60'/0'/0"
const PathDefaultWalletAccount = PathWalletRoot + "/0"
const CustomWalletPath1 = PathWalletRoot + "/1"
const CustomWalletPath2 = PathWalletRoot + "/2"
