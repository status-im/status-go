
# This module (see also `./status_go/impl.nim`) wraps the API supplied by
# status-go when it is compiled into `libstatus.a|dll|dylib|so`.

# It is expected that this module will be consumed as an import in another Nim
# module; when doing so, it is not necessary to separately compile
# `./status_go/impl.nim` but it is the responsibility of the user to link
# status-go compiled to `libstatus` into the final executable.

import ./status_go/impl as go_shim

export SignalCallback

proc hashMessage*(message: string): string =
  var funcOut = go_shim.hashMessage(message.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc initializeApplication*(paramsJSON: string): string =
  var funcOut = go_shim.initializeApplication(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc toggleCentralizedMetrics*(paramsJSON: string): string =
  var funcOut = go_shim.toggleCentralizedMetrics(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc addCentralizedMetric*(paramsJSON: string): string =
  var funcOut = go_shim.addCentralizedMetric(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc centralizedMetricsInfo*(): string =
  var funcOut = go_shim.centralizedMetricsInfo()
  defer: go_shim.free(funcOut)
  return $funcOut

proc createAccountFromMnemonicAndDeriveAccountsForPaths*(paramsJSON: string): string =
  var funcOut = go_shim.createAccountFromMnemonicAndDeriveAccountsForPaths(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc deriveAccountsPublicInfoFromExtendedPublicKeyForPaths*(paramsJSON: string): string =
  var funcOut = go_shim.deriveAccountsPublicInfoFromExtendedPublicKeyForPaths(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc deriveExtendedPublicKeyAtPath*(paramsJSON: string): string =
  var funcOut = go_shim.deriveExtendedPublicKeyAtPath(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc convertURCryptoHDKeyToXPub*(ur: string): string =
  var funcOut = go_shim.convertURCryptoHDKeyToXPub(ur.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc createAccountFromPrivateKey*(paramsJSON: string): string =
  var funcOut = go_shim.createAccountFromPrivateKey(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountImportPrivateKey*(paramsJSON: string): string =
  var funcOut = go_shim.multiAccountImportPrivateKey(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountDeriveAddresses*(paramsJSON: string): string =
  var funcOut = go_shim.multiAccountDeriveAddresses(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc deleteMultiAccount*(keyUID: string, keyStoreDir: string): string =
  var funcOut = go_shim.deleteMultiAccount(keyUID.cstring, keyStoreDir.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc callRPC*(inputJSON: string): string =
  var funcOut = go_shim.callRPC(inputJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc callPrivateRPC*(inputJSON: string): string =
  var funcOut = go_shim.callPrivateRPC(inputJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc addPeer*(peer: string): string =
  var funcOut = go_shim.addPeer(peer.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc setSignalEventCallback*(callback: SignalCallback) =
  go_shim.setSignalEventCallback(callback)

proc sendTransaction*(jsonArgs: string, password: string): string =
  var funcOut = go_shim.sendTransaction(jsonArgs.cstring, password.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc sendTransactionWithChainId*(chainId: int, jsonArgs: string, password: string): string =
  var funcOut = go_shim.sendTransactionWithChainId(chainId.cint, jsonArgs.cstring, password.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc generateAlias*(pk: string): string =
  var funcOut = go_shim.generateAlias(pk)
  defer: go_shim.free(funcOut)
  return $funcOut

proc isAlias*(value: string): string =
  var funcOut = go_shim.isAlias(value)
  defer: go_shim.free(funcOut)
  return $funcOut

proc identicon*(pk: string): string =
  var funcOut = go_shim.identicon(pk)
  defer: go_shim.free(funcOut)
  return $funcOut

proc emojiHash*(pk: string): string =
  var funcOut = go_shim.emojiHash(pk)
  defer: go_shim.free(funcOut)
  return $funcOut

proc colorHash*(pk: string): string =
  var funcOut = go_shim.colorHash(pk)
  defer: go_shim.free(funcOut)
  return $funcOut

proc colorID*(pk: string): string =
  var funcOut = go_shim.colorID(pk)
  defer: go_shim.free(funcOut)
  return $funcOut

proc login*(accountData: string, password: string): string =
  var funcOut = go_shim.login(accountData.cstring, password.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc logout*(): string =
  var funcOut = go_shim.logout()
  defer: go_shim.free(funcOut)
  return $funcOut

proc changeDatabasePasswordV2*(paramsJSON: string): string =
  var funcOut = go_shim.changeDatabasePasswordV2(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc validateMnemonic*(mnemonic: string): string =
  var funcOut = go_shim.validateMnemonic(mnemonic.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getRandomMnemonic*(): string =
  var funcOut = go_shim.getRandomMnemonic()
  defer: go_shim.free(funcOut)
  return $funcOut

proc hashTransaction*(txArgsJSON: string): string =
  var funcOut = go_shim.hashTransaction(txArgsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc extractGroupMembershipSignatures*(signaturePairsStr: string): string =
  var funcOut = go_shim.extractGroupMembershipSignatures(signaturePairsStr.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc connectionChange*(typ: string, expensive: string) =
  go_shim.connectionChange(typ.cstring, expensive.cstring)

proc multiformatSerializePublicKey*(key: string, outBase: string): string =
  var funcOut = go_shim.multiformatSerializePublicKey(key.cstring, outBase.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiformatDeserializePublicKey*(key: string, outBase: string): string =
  var funcOut = go_shim.multiformatDeserializePublicKey(key.cstring, outBase.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc decompressPublicKey*(key: string): string =
  var funcOut = go_shim.decompressPublicKey(key.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc compressPublicKey*(key: string): string =
  var funcOut = go_shim.compressPublicKey(key.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc validateNodeConfig*(configJSON: string): string =
  var funcOut = go_shim.validateNodeConfig(configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc convertToKeycardAccount*(accountData: string, settingsJSON: string, keycardUid: string, password: string, newPassword: string): string =
  var funcOut = go_shim.convertToKeycardAccount(accountData.cstring, settingsJSON.cstring, keycardUid.cstring, password.cstring, newPassword.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc convertToRegularAccount*(mnemonic: string, currPassword: string, newPassword: string): string =
  var funcOut = go_shim.convertToRegularAccount(mnemonic.cstring, currPassword.cstring, newPassword.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc recover*(rpcParams: string): string =
  var funcOut = go_shim.recover(rpcParams.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc writeHeapProfile*(dataDir: string): string =
  var funcOut = go_shim.writeHeapProfile(dataDir.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc hashTypedData*(data: string): string =
  var funcOut = go_shim.hashTypedData(data.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc hashTypedDataV4*(data: string): string =
  var funcOut = go_shim.hashTypedDataV4(data.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc resetChainData*(): string =
  var funcOut = go_shim.resetChainData()
  defer: go_shim.free(funcOut)
  return $funcOut

proc signMessage*(rpcParams: string): string =
  var funcOut = go_shim.signMessage(rpcParams.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc signTypedData*(data: string, address: string, password: string): string =
  var funcOut = go_shim.signTypedData(data.cstring, address.cstring, password.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getNodesFromContract*(rpcEndpoint: string, contractAddress: string): string =
  var funcOut = go_shim.getNodesFromContract(rpcEndpoint.cstring, contractAddress.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc exportNodeLogs*(): string =
  var funcOut = go_shim.exportNodeLogs()
  defer: go_shim.free(funcOut)
  return $funcOut

proc chaosModeUpdate*(on: int): string =
  var funcOut = go_shim.chaosModeUpdate(on.cint)
  defer: go_shim.free(funcOut)
  return $funcOut

proc signHash*(hexEncodedHash: string): string =
  var funcOut = go_shim.signHash(hexEncodedHash.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc sendTransactionWithSignature*(txtArgsJSON: string, sigString: string): string =
  var funcOut = go_shim.sendTransactionWithSignature(txtArgsJSON.cstring, sigString.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc appStateChange*(state: string) =
  go_shim.appStateChange(state.cstring)

proc signGroupMembership*(content: string): string =
  var funcOut = go_shim.signGroupMembership(content.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountStoreAccount*(paramsJSON: string): string =
  var funcOut = go_shim.multiAccountStoreAccount(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountLoadAccount*(paramsJSON: string): string =
  var funcOut = go_shim.multiAccountLoadAccount(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountGenerate*(paramsJSON: string): string =
  var funcOut = go_shim.multiAccountGenerate(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc multiAccountReset*(): string =
  var funcOut = go_shim.multiAccountReset()
  defer: go_shim.free(funcOut)
  return $funcOut

proc migrateKeyStoreDir*(accountData: string, password: string, oldKeystoreDir: string, multiaccountKeystoreDir: string): string =
  var funcOut = go_shim.migrateKeyStoreDir(accountData.cstring, password.cstring, oldKeystoreDir.cstring, multiaccountKeystoreDir.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc startWallet*(watchNewBlocks: bool): string =
  var funcOut = go_shim.startWallet(watchNewBlocks)
  defer: go_shim.free(funcOut)
  return $funcOut

proc stopWallet*(): string =
  var funcOut = go_shim.stopWallet()
  defer: go_shim.free(funcOut)
  return $funcOut

proc startLocalNotifications*(): string =
  var funcOut = go_shim.startLocalNotifications()
  defer: go_shim.free(funcOut)
  return $funcOut

proc stopLocalNotifications*(): string =
  var funcOut = go_shim.stopLocalNotifications()
  defer: go_shim.free(funcOut)
  return $funcOut

proc getNodeConfig*(): string =
  var funcOut = go_shim.getNodeConfig()
  defer: go_shim.free(funcOut)
  return $funcOut

proc imageServerTLSCert*(): string =
  var funcOut = go_shim.imageServerTLSCert()
  defer: go_shim.free(funcOut)
  return $funcOut

proc getPasswordStrength*(paramsJSON: string): string =
  var funcOut = go_shim.getPasswordStrength(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getPasswordStrengthScore*(paramsJSON: string): string =
  var funcOut = go_shim.getPasswordStrengthScore(paramsJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc switchFleet*(newFleet: string, configJSON: string): string =
  var funcOut = go_shim.switchFleet(newFleet.cstring, configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc generateImages*(imagePath: string, aX: int, aY: int, bX: int, bY: int): string =
  var funcOut = go_shim.generateImages(imagePath.cstring, aX.cint, aY.cint, bX.cint, bY.cint)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getConnectionStringForBeingBootstrapped*(configJSON: string): string =
  var funcOut = go_shim.getConnectionStringForBeingBootstrapped(configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getConnectionStringForBootstrappingAnotherDevice*(configJSON: string): string =
  var funcOut = go_shim.getConnectionStringForBootstrappingAnotherDevice(configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc inputConnectionStringForBootstrapping*(connectionString: string, configJSON: string): string =
  var funcOut = go_shim.inputConnectionStringForBootstrapping(connectionString.cstring, configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc validateConnectionString*(connectionString: string): string =
  var funcOut = go_shim.validateConnectionString(connectionString.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getConnectionStringForExportingKeypairsKeystores*(configJSON: string): string =
  var funcOut = go_shim.getConnectionStringForExportingKeypairsKeystores(configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc inputConnectionStringForImportingKeypairsKeystores*(connectionString: string, configJSON: string): string =
  var funcOut = go_shim.inputConnectionStringForImportingKeypairsKeystores(connectionString.cstring, configJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc loginAccount*(requestJson: string): string =
  var funcOut = go_shim.loginAccount(requestJson.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc createAccountAndLogin*(requestJSON: string): string =
  var funcOut = go_shim.createAccountAndLogin(requestJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc restoreAccountAndLogin*(requestJSON: string): string =
  var funcOut = go_shim.restoreAccountAndLogin(requestJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc deleteMultiaccount*(requestJSON: string): string =
  var funcOut = go_shim.deleteMultiaccount(requestJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc performLocalBackup*(): string =
  var funcOut = go_shim.performLocalBackup()
  defer: go_shim.free(funcOut)
  return $funcOut

proc loadLocalBackup*(filePath: string): string =
  var funcOut = go_shim.loadLocalBackup(filePath)
  defer: go_shim.free(funcOut)
  return $funcOut

proc connectionChange*(requestJSON: string): string =
  var funcOut = go_shim.connectionChange(requestJSON.cstring)
  defer: go_shim.free(funcOut)
  return $funcOut

proc getActiveAccount*(): string =
  var funcOut = go_shim.getActiveAccount()
  defer: go_shim.free(funcOut)
  return $funcOut

proc keyUID*(): string =
  var funcOut = go_shim.keyUID()
  defer: go_shim.free(funcOut)
  return $funcOut
proc statusBackendRunServer*(address: string): string =
  ## Runs the status-backend HTTP server on `address` (host:port), blocking
  ## until it stops. Returns "" on clean shutdown, an error message otherwise.
  var funcOut = go_shim.statusBackendRunServer(address.cstring)
  if funcOut == nil:
    return ""
  defer: go_shim.free(funcOut)
  return $funcOut
