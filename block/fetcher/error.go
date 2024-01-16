package fetcher

import (
	"encoding/binary"
	"fmt"

	bin "github.com/streamingfast/binary"
)

type errDetail interface {
	Encode(encoder *bin.Encoder) error
}

type TrxErrCode int32

const (
	TrxErr_AccountInUse TrxErrCode = iota
	TrxErr_AccountLoadedTwice
	TrxErr_AccountNotFound
	TrxErr_ProgramAccountNotFound
	TrxErr_InsufficientFundsForFee
	TrxErr_InvalidAccountForFee
	TrxErr_AlreadyProcessed
	TrxErr_BlockhashNotFound
	TrxErr_InstructionError
	TrxErr_CallChainTooDeep
	TrxErr_MissingSignatureForFee
	TrxErr_InvalidAccountIndex
	TrxErr_SignatureFailure
	TrxErr_InvalidProgramForExecution
	TrxErr_SanitizeFailure
	TrxErr_ClusterMaintenance
	TrxErr_AccountBorrowOutstanding
	TrxErr_WouldExceedMaxBlockCostLimit
	TrxErr_UnsupportedVersion
	TrxErr_InvalidWritableAccount
	TrxErr_WouldExceedMaxAccountCostLimit
	TrxErr_WouldExceedAccountDataBlockLimit
	TrxErr_TooManyAccountLocks
	TrxErr_AddressLookupTableNotFound
	TrxErr_InvalidAddressLookupTableOwner
	TrxErr_InvalidAddressLookupTableData
	TrxErr_InvalidAddressLookupTableIndex
	TrxErr_InvalidRentPayingAccount
	TrxErr_WouldExceedMaxVoteCostLimit
	TrxErr_WouldExceedAccountDataTotalLimit
	TrxErr_DuplicateInstruction
	TrxErr_InsufficientFundsForRent
	TrxErr_MaxLoadedAccountsDataSizeExceeded
	TrxErr_InvalidLoadedAccountsDataSizeLimit
	TrxErr_ResanitizationNeeded
	TrxErr_ProgramExecutionTemporarilyRestricted
	TrxErr_UnbalancedTransaction
)

var trxErrorMap = map[string]TrxErrCode{
	"AccountInUse":                          TrxErr_AccountInUse,
	"AccountLoadedTwice":                    TrxErr_AccountLoadedTwice,
	"AccountNotFound":                       TrxErr_AccountNotFound,
	"ProgramAccountNotFound":                TrxErr_ProgramAccountNotFound,
	"InsufficientFundsForFee":               TrxErr_InsufficientFundsForFee,
	"InvalidAccountForFee":                  TrxErr_InvalidAccountForFee,
	"AlreadyProcessed":                      TrxErr_AlreadyProcessed,
	"BlockhashNotFound":                     TrxErr_BlockhashNotFound,
	"InstructionError":                      TrxErr_InstructionError,
	"CallChainTooDeep":                      TrxErr_CallChainTooDeep,
	"MissingSignatureForFee":                TrxErr_MissingSignatureForFee,
	"InvalidAccountIndex":                   TrxErr_InvalidAccountIndex,
	"SignatureFailure":                      TrxErr_SignatureFailure,
	"InvalidProgramForExecution":            TrxErr_InvalidProgramForExecution,
	"SanitizeFailure":                       TrxErr_SanitizeFailure,
	"ClusterMaintenance":                    TrxErr_ClusterMaintenance,
	"AccountBorrowOutstanding":              TrxErr_AccountBorrowOutstanding,
	"WouldExceedMaxBlockCostLimit":          TrxErr_WouldExceedMaxBlockCostLimit,
	"UnsupportedVersion":                    TrxErr_UnsupportedVersion,
	"InvalidWritableAccount":                TrxErr_InvalidWritableAccount,
	"WouldExceedMaxAccountCostLimit":        TrxErr_WouldExceedMaxAccountCostLimit,
	"WouldExceedAccountDataBlockLimit":      TrxErr_WouldExceedAccountDataBlockLimit,
	"TooManyAccountLocks":                   TrxErr_TooManyAccountLocks,
	"AddressLookupTableNotFound":            TrxErr_AddressLookupTableNotFound,
	"InvalidAddressLookupTableOwner":        TrxErr_InvalidAddressLookupTableOwner,
	"InvalidAddressLookupTableData":         TrxErr_InvalidAddressLookupTableData,
	"InvalidAddressLookupTableIndex":        TrxErr_InvalidAddressLookupTableIndex,
	"InvalidRentPayingAccount":              TrxErr_InvalidRentPayingAccount,
	"WouldExceedMaxVoteCostLimit":           TrxErr_WouldExceedMaxVoteCostLimit,
	"WouldExceedAccountDataTotalLimit":      TrxErr_WouldExceedAccountDataTotalLimit,
	"DuplicateInstruction":                  TrxErr_DuplicateInstruction,
	"InsufficientFundsForRent":              TrxErr_InsufficientFundsForRent,
	"MaxLoadedAccountsDataSizeExceeded":     TrxErr_MaxLoadedAccountsDataSizeExceeded,
	"InvalidLoadedAccountsDataSizeLimit":    TrxErr_InvalidLoadedAccountsDataSizeLimit,
	"ResanitizationNeeded":                  TrxErr_ResanitizationNeeded,
	"ProgramExecutionTemporarilyRestricted": TrxErr_ProgramExecutionTemporarilyRestricted,
	"UnbalancedTransaction":                 TrxErr_UnbalancedTransaction,
}

type TransactionError struct {
	TrxErrCode
	detail errDetail
}

func MustNewTransactionError(e any) *TransactionError {
	if e == nil {
		return nil
	}

	if errorName, ok := e.(string); ok {
		if errorCode, ok := trxErrorMap[errorName]; ok {
			return &TransactionError{errorCode, nil}
		}
		panic(fmt.Errorf("unknown error name: %s", errorName))
	}

	if mapErr, ok := e.(map[string]interface{}); ok {
		if len(mapErr) != 1 {
			panic(fmt.Errorf("unknown error map: %v", mapErr))
		}
		for errorName, detailMap := range mapErr {
			if errorCode, ok := trxErrorMap[errorName]; ok {
				var errorDetail errDetail

				//	//8 0 0 0 3 25 0 0 0 113 23 0 0
				//	//"err":{"InstructionError":[3,{"Custom":22}]}
				//
				//	//[1 0 0 0]
				//	//"err":"AccountInUse"
				//
				//	//8 0 0 0 -> TransactionError.InstructionError
				//	//3 -> instruction index
				//	//25 0 0 0 -> InstructionError.Custom
				//	//113 23 0 0 -> u32 error code

				switch errorCode {
				case TrxErr_InstructionError:
					errorDetail = MustNewInstructionError(detailMap)
				case TrxErr_DuplicateInstruction:
					errorDetail = MustNewDuplicateInstructionError(detailMap)
				case TrxErr_InsufficientFundsForRent:
					errorDetail = MustNewInsufficientFundsForRentError(detailMap)
				case TrxErr_ProgramExecutionTemporarilyRestricted:
					errorDetail = MustNewProgramExecutionTemporarilyRestrictedError(detailMap)
				default:
					panic(fmt.Errorf("unknown error code: %d", errorCode))
				}

				return &TransactionError{errorCode, errorDetail}
			}
			panic(fmt.Errorf("unknown error name: %s", errorName))
		}
		//we should never get here since we checked the length of the map and we are exiting only one element
	}

	panic(fmt.Errorf("unknown error type: %T", e))
}

type DuplicateInstructionError struct {
	duplicateInstructionIndex byte
}

type InsufficientFundsForRentError struct {
	AccountIndex byte
}

type ProgramExecutionTemporarilyRestrictedError struct {
	AccountIndex byte
}

func (e *ProgramExecutionTemporarilyRestrictedError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteByte(e.AccountIndex)
	if err != nil {
		return fmt.Errorf("unable to encode byte: %w", err)
	}
	return nil
}

func (e *InsufficientFundsForRentError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteByte(e.AccountIndex)
	if err != nil {
		return fmt.Errorf("unable to encode byte: %w", err)
	}
	return nil
}

func MustNewProgramExecutionTemporarilyRestrictedError(e any) *ProgramExecutionTemporarilyRestrictedError {
	accountIndex, ok := e.(uint8)
	if !ok {
		panic(fmt.Errorf("expected byte, got: %T", e))
	}
	return &ProgramExecutionTemporarilyRestrictedError{
		AccountIndex: accountIndex,
	}
}

func MustNewInsufficientFundsForRentError(e any) *InsufficientFundsForRentError {
	accountIndex, ok := e.(uint8)
	if !ok {
		panic(fmt.Errorf("expected byte, got: %T", e))
	}
	return &InsufficientFundsForRentError{
		AccountIndex: accountIndex,
	}
}
func MustNewDuplicateInstructionError(e any) *DuplicateInstructionError {
	duplicateInstructionIndex, ok := e.(uint8)
	if !ok {
		panic(fmt.Errorf("expected byte, got: %T", e))
	}
	return &DuplicateInstructionError{
		duplicateInstructionIndex: duplicateInstructionIndex,
	}
}

func (e *DuplicateInstructionError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteByte(e.duplicateInstructionIndex)
	if err != nil {
		return fmt.Errorf("unable to encode byte: %w", err)
	}
	return nil
}

func (e *TransactionError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteUint32(uint32(e.TrxErrCode), binary.LittleEndian)
	if err != nil {
		return err
	}
	if e.detail == nil {
		return nil
	}
	err = e.detail.Encode(encoder)
	if err != nil {
		return fmt.Errorf("unable to encode error detail: %w", err)
	}
	return nil
}

type InstructionErrorCode uint32

const (
	InstructionError_GenericError InstructionErrorCode = iota
	InstructionError_InvalidArgument
	InstructionError_InvalidInstructionData
	InstructionError_InvalidAccountData
	InstructionError_AccountDataTooSmall
	InstructionError_InsufficientFunds
	InstructionError_IncorrectProgramId
	InstructionError_MissingRequiredSignature
	InstructionError_AccountAlreadyInitialized
	InstructionError_UninitializedAccount
	InstructionError_UnbalancedInstruction
	InstructionError_ModifiedProgramId
	InstructionError_ExternalAccountLamportSpend
	InstructionError_ExternalAccountDataModified
	InstructionError_ReadonlyLamportChange
	InstructionError_ReadonlyDataModified
	InstructionError_DuplicateAccountIndex
	InstructionError_ExecutableModified
	InstructionError_RentEpochModified
	InstructionError_NotEnoughAccountKeys
	InstructionError_AccountDataSizeChanged
	InstructionError_AccountNotExecutable
	InstructionError_AccountBorrowFailed
	InstructionError_AccountBorrowOutstanding
	InstructionError_DuplicateAccountOutOfSync
	InstructionError_Custom
	InstructionError_InvalidError
	InstructionError_ExecutableDataModified
	InstructionError_ExecutableLamportChange
	InstructionError_ExecutableAccountNotRentExempt
	InstructionError_UnsupportedProgramId
	InstructionError_CallDepth
	InstructionError_MissingAccount
	InstructionError_ReentrancyNotAllowed
	InstructionError_MaxSeedLengthExceeded
	InstructionError_InvalidSeeds
	InstructionError_InvalidRealloc
	InstructionError_ComputationalBudgetExceeded
	InstructionError_PrivilegeEscalation
	InstructionError_ProgramEnvironmentSetupFailure
	InstructionError_ProgramFailedToComplete
	InstructionError_ProgramFailedToCompile
	InstructionError_Immutable
	InstructionError_IncorrectAuthority
	InstructionError_BorshIoError
	InstructionError_AccountNotRentExempt
	InstructionError_InvalidAccountOwner
	InstructionError_ArithmeticOverflow
	InstructionError_UnsupportedSysvar
	InstructionError_IllegalOwner
	InstructionError_MaxAccountsDataAllocationsExceeded
	InstructionError_MaxAccountsExceeded
	InstructionError_MaxInstructionTraceLengthExceeded
	InstructionError_BuiltinProgramsMustConsumeComputeUnits
)

var instructionErrorMap = map[string]InstructionErrorCode{
	"GenericError":                           InstructionError_GenericError,
	"InvalidArgument":                        InstructionError_InvalidArgument,
	"InvalidInstructionData":                 InstructionError_InvalidInstructionData,
	"InvalidAccountData":                     InstructionError_InvalidAccountData,
	"AccountDataTooSmall":                    InstructionError_AccountDataTooSmall,
	"InsufficientFunds":                      InstructionError_InsufficientFunds,
	"IncorrectProgramId":                     InstructionError_IncorrectProgramId,
	"MissingRequiredSignature":               InstructionError_MissingRequiredSignature,
	"AccountAlreadyInitialized":              InstructionError_AccountAlreadyInitialized,
	"UninitializedAccount":                   InstructionError_UninitializedAccount,
	"UnbalancedInstruction":                  InstructionError_UnbalancedInstruction,
	"ModifiedProgramId":                      InstructionError_ModifiedProgramId,
	"ExternalAccountLamportSpend":            InstructionError_ExternalAccountLamportSpend,
	"ExternalAccountDataModified":            InstructionError_ExternalAccountDataModified,
	"ReadonlyLamportChange":                  InstructionError_ReadonlyLamportChange,
	"ReadonlyDataModified":                   InstructionError_ReadonlyDataModified,
	"DuplicateAccountIndex":                  InstructionError_DuplicateAccountIndex,
	"ExecutableModified":                     InstructionError_ExecutableModified,
	"RentEpochModified":                      InstructionError_RentEpochModified,
	"NotEnoughAccountKeys":                   InstructionError_NotEnoughAccountKeys,
	"AccountDataSizeChanged":                 InstructionError_AccountDataSizeChanged,
	"AccountNotExecutable":                   InstructionError_AccountNotExecutable,
	"AccountBorrowFailed":                    InstructionError_AccountBorrowFailed,
	"AccountBorrowOutstanding":               InstructionError_AccountBorrowOutstanding,
	"DuplicateAccountOutOfSync":              InstructionError_DuplicateAccountOutOfSync,
	"Custom":                                 InstructionError_Custom,
	"InvalidError":                           InstructionError_InvalidError,
	"ExecutableDataModified":                 InstructionError_ExecutableDataModified,
	"ExecutableLamportChange":                InstructionError_ExecutableLamportChange,
	"ExecutableAccountNotRentExempt":         InstructionError_ExecutableAccountNotRentExempt,
	"UnsupportedProgramId":                   InstructionError_UnsupportedProgramId,
	"CallDepth":                              InstructionError_CallDepth,
	"MissingAccount":                         InstructionError_MissingAccount,
	"ReentrancyNotAllowed":                   InstructionError_ReentrancyNotAllowed,
	"MaxSeedLengthExceeded":                  InstructionError_MaxSeedLengthExceeded,
	"InvalidSeeds":                           InstructionError_InvalidSeeds,
	"InvalidRealloc":                         InstructionError_InvalidRealloc,
	"ComputationalBudgetExceeded":            InstructionError_ComputationalBudgetExceeded,
	"PrivilegeEscalation":                    InstructionError_PrivilegeEscalation,
	"ProgramEnvironmentSetupFailure":         InstructionError_ProgramEnvironmentSetupFailure,
	"ProgramFailedToComplete":                InstructionError_ProgramFailedToComplete,
	"ProgramFailedToCompile":                 InstructionError_ProgramFailedToCompile,
	"Immutable":                              InstructionError_Immutable,
	"IncorrectAuthority":                     InstructionError_IncorrectAuthority,
	"BorshIoError":                           InstructionError_BorshIoError,
	"AccountNotRentExempt":                   InstructionError_AccountNotRentExempt,
	"InvalidAccountOwner":                    InstructionError_InvalidAccountOwner,
	"ArithmeticOverflow":                     InstructionError_ArithmeticOverflow,
	"UnsupportedSysvar":                      InstructionError_UnsupportedSysvar,
	"IllegalOwner":                           InstructionError_IllegalOwner,
	"MaxAccountsDataAllocationsExceeded":     InstructionError_MaxAccountsDataAllocationsExceeded,
	"MaxAccountsExceeded":                    InstructionError_MaxAccountsExceeded,
	"MaxInstructionTraceLengthExceeded":      InstructionError_MaxInstructionTraceLengthExceeded,
	"BuiltinProgramsMustConsumeComputeUnits": InstructionError_BuiltinProgramsMustConsumeComputeUnits,
}

type InstructionError struct {
	InstructionErrorCode
	InstructionIndex byte
	detail           errDetail
}

func MustNewInstructionError(e any) *InstructionError {
	if e == nil {
		return nil
	}

	parts := e.([]any)
	if len(parts) != 2 {
		panic(fmt.Errorf("invalid number of parts for InstructionError: %d", len(parts)))
	}

	instructionIndex := byte(parts[0].(float64))

	if errorName, isString := parts[1].(string); isString {
		if errorCode, ok := instructionErrorMap[errorName]; ok {
			return &InstructionError{InstructionErrorCode: errorCode, InstructionIndex: instructionIndex}
		}
		panic(fmt.Errorf("unknown error name: %s", errorName))
	}

	if mapErr, ok := parts[1].(map[string]any); ok {
		if len(mapErr) != 1 {
			panic(fmt.Errorf("unknown error map: %v", mapErr))
		}
		for errorName, details := range mapErr {
			if errorCode, ok := instructionErrorMap[errorName]; ok {
				var errorDetail errDetail

				switch errorCode {
				case InstructionError_Custom:
					errorDetail = MustNewInstructionCustomError(details)
				case InstructionError_BorshIoError:
					errorDetail = MustNewBorshIoError(details)
				default:
					panic(fmt.Errorf("unknown error code: %d", errorCode))
				}

				return &InstructionError{InstructionErrorCode: errorCode, InstructionIndex: instructionIndex, detail: errorDetail}
			}
			panic(fmt.Errorf("unknown error name: %s", errorName))
		}
		//we should never get here since we checked the length of the map and we are exiting only one element
	}

	panic(fmt.Errorf("unknown error type: %T", e))

}

func (i *InstructionError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteByte(i.InstructionIndex)
	if err != nil {
		return fmt.Errorf("unable to encode instruction index: %w", err)
	}
	err = encoder.WriteUint32(uint32(i.InstructionErrorCode), binary.LittleEndian)
	if err != nil {
		return fmt.Errorf("unable to encode error code: %w", err)
	}
	if i.detail == nil {
		return nil
	}
	err = i.detail.Encode(encoder)
	if err != nil {
		return fmt.Errorf("unable to encode error detail: %w", err)
	}
	return nil
}

type InstructionCustomError struct {
	CustomErrorCode uint32
}

func MustNewInstructionCustomError(e any) InstructionCustomError {
	customErrorCode, ok := e.(float64)
	if !ok {
		panic(fmt.Errorf("expected float64, got: %T", e))
	}

	return InstructionCustomError{
		CustomErrorCode: uint32(customErrorCode),
	}
}

func (i InstructionCustomError) Encode(encoder *bin.Encoder) error {
	err := encoder.WriteUint32(i.CustomErrorCode, binary.LittleEndian)
	if err != nil {
		return fmt.Errorf("unable to encode custom error code: %w", err)
	}
	return nil
}

type BorshIoError struct {
	Msg string
}

func MustNewBorshIoError(a any) BorshIoError {
	msg, ok := a.(string)
	if !ok {
		panic(fmt.Errorf("expected string, got: %T", a))
	}
	return BorshIoError{Msg: msg}
}

func (b BorshIoError) Encode(encoder *bin.Encoder) error {
	err := WriteString(b.Msg, encoder)
	if err != nil {
		return fmt.Errorf("unable to encode borsh io error: %w", err)
	}
	return nil
}

func WriteString(b string, e *bin.Encoder) error {
	length := len(b)
	err := e.WriteInt64(int64(length), binary.LittleEndian)
	if err != nil {
		return fmt.Errorf("unable to encode string length: %w", err)
	}
	err = e.WriteRaw([]byte(b))
	if err != nil {
		return fmt.Errorf("unable to encode string: %w", err)
	}
	return nil
}
