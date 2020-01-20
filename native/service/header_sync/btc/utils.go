package btc

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/ontio/multi-chain/common"
	"github.com/ontio/multi-chain/common/config"
	"github.com/ontio/multi-chain/common/log"
	cstates "github.com/ontio/multi-chain/core/states"
	"github.com/ontio/multi-chain/native"
	"github.com/ontio/multi-chain/native/event"
	scom "github.com/ontio/multi-chain/native/service/header_sync/common"
	"github.com/ontio/multi-chain/native/service/utils"
	"math/big"
	"time"
)

const (
	targetTimespan      = time.Hour * 24 * 14
	targetSpacing       = time.Minute * 10
	epochLength         = int32(targetTimespan / targetSpacing) // 2016
	maxDiffAdjust       = 4
	minRetargetTimespan = int64(targetTimespan / maxDiffAdjust)
	maxRetargetTimespan = int64(targetTimespan * maxDiffAdjust)
)

var netParam = &chaincfg.RegressionNetParams

func putGenesisBlockHeader(native *native.NativeService, chainID uint64, blockHeader StoredHeader) {
	contract := utils.HeaderSyncContractAddress
	blockHash := blockHeader.Header.BlockHash()
	blockHeight := blockHeader.Height

	sink := new(common.ZeroCopySink)
	blockHeader.Serialization(sink)
	native.GetCacheDB().Put(utils.ConcatKey(contract, []byte(scom.GENESIS_HEADER), utils.GetUint64Bytes(chainID)),
		cstates.GenRawStorageItem(sink.Bytes()))

	putBlockHash(native, chainID, blockHeight, blockHash)

	putBlockHeader(native, chainID, blockHeader)

	putBestBlockHeader(native, chainID, blockHeader)

	notifyPutHeader(native, chainID, blockHeight, hex.EncodeToString(blockHash.CloneBytes()))
}

func notifyPutHeader(native *native.NativeService, chainID uint64, height uint32, blockHash string) {
	if !config.DefConfig.Common.EnableEventLog {
		return
	}
	native.AddNotify(
		&event.NotifyEventInfo{
			ContractAddress: utils.HeaderSyncContractAddress,
			States:          []interface{}{chainID, height, blockHash, native.GetHeight()},
		})
}

func putBlockHash(native *native.NativeService, chainID uint64, height uint32, hash chainhash.Hash) {
	native.GetCacheDB().Put(utils.ConcatKey(utils.HeaderSyncContractAddress, []byte(scom.HEADER_INDEX), utils.GetUint64Bytes(chainID), utils.GetUint32Bytes(height)),
		cstates.GenRawStorageItem(hash.CloneBytes()))
}

func GetBlockHashByHeight(native *native.NativeService, chainID uint64, height uint32) (*chainhash.Hash, error) {
	contract := utils.HeaderSyncContractAddress

	hashStore, err := native.GetCacheDB().Get(utils.ConcatKey(contract, []byte(scom.HEADER_INDEX), utils.GetUint64Bytes(chainID), utils.GetUint32Bytes(height)))
	if err != nil {
		return nil, fmt.Errorf("GetBlockHashByHeight, get heightBlockHashStore error: %v", err)
	}
	if hashStore == nil {
		return nil, fmt.Errorf("GetBlockHashByHeight, can not find any index records")
	}
	hashBs, err := cstates.GetValueFromRawStorageItem(hashStore)
	if err != nil {
		return nil, fmt.Errorf("GetBlockHashByHeight, deserialize blockHashBytes from raw storage item err:%v", err)
	}

	hash := new(chainhash.Hash)
	err = hash.SetBytes(hashBs)
	if err != nil {
		return nil, fmt.Errorf("GetBlockHashByHeight at height = %d, error:%v", height, err)
	}
	return hash, nil
}

func putBlockHeader(native *native.NativeService, chainID uint64, sh StoredHeader) {
	contract := utils.HeaderSyncContractAddress

	blockHash := sh.Header.BlockHash()
	sink := new(common.ZeroCopySink)
	sh.Serialization(sink)
	native.GetCacheDB().Put(utils.ConcatKey(contract, []byte(scom.BLOCK_HEADER), utils.GetUint64Bytes(chainID), blockHash.CloneBytes()),
		cstates.GenRawStorageItem(sink.Bytes()))
}

func GetHeaderByHash(native *native.NativeService, chainID uint64, hash chainhash.Hash) (*StoredHeader, error) {
	contract := utils.HeaderSyncContractAddress

	headerStore, err := native.GetCacheDB().Get(utils.ConcatKey(contract, []byte(scom.BLOCK_HEADER), utils.GetUint64Bytes(chainID), hash.CloneBytes()))
	if err != nil {
		return nil, fmt.Errorf("GetHeaderByHash, get hashBlockHeaderStore error: %v", err)
	}
	if headerStore == nil {
		return nil, fmt.Errorf("GetHeaderByHash, can not find any index records")
	}
	shBs, err := cstates.GetValueFromRawStorageItem(headerStore)
	if err != nil {
		return nil, fmt.Errorf("GetHeaderByHash, deserialize blockHashBytes from raw storage item err: %v", err)
	}

	sh := new(StoredHeader)
	if err := sh.Deserialization(common.NewZeroCopySource(shBs)); err != nil {
		return nil, fmt.Errorf("GetStoredHeader, deserializeHeader error: %v", err)
	}

	return sh, nil
}
func GetHeaderByHeight(native *native.NativeService, chainID uint64, height uint32) (*StoredHeader, error) {
	blockHash, err := GetBlockHashByHeight(native, chainID, height)
	if err != nil {
		return nil, fmt.Errorf("GetHeaderByHeight, error: %v", err)
	}
	storedHeader, err := GetHeaderByHash(native, chainID, *blockHash)
	if err != nil {
		return nil, fmt.Errorf("GetHeaderByHeight, error: %v", err)
	}
	return storedHeader, nil
}

func putBestBlockHeader(native *native.NativeService, chainID uint64, bestHeader StoredHeader) {
	contract := utils.HeaderSyncContractAddress

	sink := new(common.ZeroCopySink)
	bestHeader.Serialization(sink)
	native.GetCacheDB().Put(utils.ConcatKey(contract, []byte(scom.CURRENT_HEADER_HEIGHT), utils.GetUint64Bytes(chainID)),
		cstates.GenRawStorageItem(sink.Bytes()))
}

func GetBestBlockHeader(native *native.NativeService, chainID uint64) (*StoredHeader, error) {
	contract := utils.HeaderSyncContractAddress

	bestBlockHeaderStore, err := native.GetCacheDB().Get(utils.ConcatKey(contract, []byte(scom.CURRENT_HEADER_HEIGHT), utils.GetUint64Bytes(chainID)))
	if err != nil {
		return nil, fmt.Errorf("GetBestBlockHeader, get BestBlockHeader error: %v", err)
	}
	if bestBlockHeaderStore == nil {
		return nil, fmt.Errorf("GetBestBlockHeader, can not find any index records")
	}
	bestBlockHeaderBs, err := cstates.GetValueFromRawStorageItem(bestBlockHeaderStore)
	if err != nil {
		return nil, fmt.Errorf("GetBestBlockHeader, deserialize bestBlockHeaderBytes from raw storage item err: %v", err)
	}
	bestBlockHeader := new(StoredHeader)
	err = bestBlockHeader.Deserialization(common.NewZeroCopySource(bestBlockHeaderBs))
	if err != nil {
		return nil, fmt.Errorf("GetBestBlockHeader, deserialize storedHeader error: %v", err)
	}
	return bestBlockHeader, nil
}

func GetPreviousHeader(native *native.NativeService, chainID uint64, header wire.BlockHeader) (*StoredHeader, error) {
	return GetHeaderByHash(native, chainID, header.PrevBlock)
}

func CheckHeader(native *native.NativeService, chainID uint64, header wire.BlockHeader, prevHeader *StoredHeader) (bool, error) {
	// Get hash of n-1 header
	prevHash := prevHeader.Header.BlockHash()
	height := prevHeader.Height

	// Check if headers link together.  That whole 'blockchain' thing.
	if !prevHash.IsEqual(&header.PrevBlock) {
		return false, fmt.Errorf("CheckHeader error: Headers %d and %d don't link.", height, height+1)
	}

	// Check the header meets the difficulty requirement
	diffTarget, err := calcRequiredWork(native, chainID, header, int32(height+1), prevHeader)
	if err != nil {
		return false, fmt.Errorf("CheckHeader, calclating difficulty error: %v", err)
	}
	if header.Bits != diffTarget {
		return false, fmt.Errorf("CheckHeader, Block %d %s incorrect difficulty.  Read %d, expect %d\n",
			height+1, header.BlockHash().String(), header.Bits, diffTarget)
	}

	// Check if there's a valid proof of work.  That whole "Bitcoin" thing.
	if !checkProofOfWork(header, netParam) {
		log.Debugf("CheckHeader, Block %d bad proof of work.", height+1)
		return false, nil
	}

	return true, nil // it must have worked if there's no errors and got to the end.
}

// Get the PoW target this block should meet. We may need to handle a difficulty adjustment
// or testnet difficulty rules.
func calcRequiredWork(native *native.NativeService, chainID uint64, header wire.BlockHeader, height int32, prevHeader *StoredHeader) (uint32, error) {
	// If this is not a difficulty adjustment period
	if height%epochLength != 0 {
		// If we are on testnet
		if netParam.ReduceMinDifficulty {
			// If it's been more than 20 minutes since the last header return the minimum difficulty
			if header.Timestamp.After(prevHeader.Header.Timestamp.Add(targetSpacing * 2)) {
				return netParam.PowLimitBits, nil
			} else {
				// Otherwise return the difficulty of the last block not using special difficulty rules
				for {
					var err error = nil
					for err == nil && int32(prevHeader.Height)%epochLength != 0 && prevHeader.Header.Bits == netParam.PowLimitBits {
						var sh *StoredHeader
						sh, err = GetPreviousHeader(native, chainID, prevHeader.Header)
						// Error should only be non-nil if prevHeader is the checkpoint.
						// In that case we should just return checkpoint bits
						if err == nil {
							prevHeader = sh
						}

					}
					return prevHeader.Header.Bits, nil
				}
			}
		}
		// Just return the bits from the last header
		return prevHeader.Header.Bits, nil
	}
	// We are on a difficulty adjustment period so we need to correctly calculate the new difficulty.
	epoch, err := GetEpoch(native, chainID)
	if err != nil {
		return 0, err
	}
	return calcDiffAdjust(*epoch, prevHeader.Header, netParam), nil
}

func GetEpoch(native *native.NativeService, chainID uint64) (*wire.BlockHeader, error) {
	sh, err := GetBestBlockHeader(native, chainID)
	if err != nil {
		return &sh.Header, err
	}
	for i := 0; i < 2015; i++ {
		sh, err = GetPreviousHeader(native, chainID, sh.Header)
		if err != nil {
			return &sh.Header, err
		}
	}
	log.Debug("Epoch", sh.Header.BlockHash().String())
	return &sh.Header, nil
}

func GetCommonAncestor(native *native.NativeService, chainID uint64, bestHeader, prevBestHeader *StoredHeader) (*StoredHeader, []chainhash.Hash, error) {
	var err error
	bestHash := bestHeader.Header.BlockHash()
	hdrs := []chainhash.Hash{bestHash}

	majority := bestHeader
	minority := prevBestHeader
	if bestHeader.Height > prevBestHeader.Height {
		for i := 0; i < int(bestHeader.Height-prevBestHeader.Height); i++ {
			majority, err = GetPreviousHeader(native, chainID, majority.Header)
			if err != nil {
				return nil, nil, fmt.Errorf("VerifyFromBtcProof, failed to get previous header for %s: %v",
					majority.Header.BlockHash().String(), err)
			}
			majorityHash := majority.Header.BlockHash()
			hdrs = append(hdrs, majorityHash)
		}
	} else if prevBestHeader.Height > bestHeader.Height {
		minority, err = GetHeaderByHeight(native, chainID, bestHeader.Height)
		if err != nil {
			return nil, nil, fmt.Errorf("VerifyFromBtcProof, get header at height %d to verify btc merkle proof error:%s", bestHeader.Height, err)
		}
	}

	majorityHash, minorityHash := majority.Header.BlockHash(), minority.Header.BlockHash()
	for !majorityHash.IsEqual(&minorityHash) {
		majority, err = GetPreviousHeader(native, chainID, majority.Header)
		if err != nil {
			return nil, nil, err
		}
		minority, err = GetPreviousHeader(native, chainID, minority.Header)
		if err != nil {
			return nil, nil, err
		}
		majorityHash, minorityHash = majority.Header.BlockHash(), minority.Header.BlockHash()
		hdrs = append(hdrs, majorityHash)
	}

	return majority, hdrs[:len(hdrs)-1], nil
}

func ReIndexHeaderHeight(native *native.NativeService, chainID uint64, bestHeaderHeight uint32, hdrs []chainhash.Hash,
	newBlock *StoredHeader) error {
	contract := utils.HeaderSyncContractAddress
	for i := bestHeaderHeight; i > newBlock.Height; i-- {
		native.GetCacheDB().Delete(utils.ConcatKey(contract, []byte("HeightToBlockHash"), utils.GetUint64Bytes(chainID), utils.GetUint32Bytes(i)))
	}

	for i, v := range hdrs {
		putBlockHash(native, chainID, newBlock.Height-uint32(i), v)
	}

	return nil
}

// Verifies the header hashes into something lower than specified by the 4-byte bits field.
func checkProofOfWork(header wire.BlockHeader, p *chaincfg.Params) bool {
	target := blockchain.CompactToBig(header.Bits)

	// The target must more than 0.  Why can you even encode negative...
	if target.Sign() <= 0 {
		log.Debugf("Block target %064x is neagtive(??)\n", target.Bytes())
		return false
	}
	// The target must be less than the maximum allowed (difficulty 1)
	if target.Cmp(p.PowLimit) > 0 {
		log.Debugf("Block target %064x is "+
			"higher than max of %064x", target, p.PowLimit.Bytes())
		return false
	}
	// The header hash must be less than the claimed target in the header.
	blockHash := header.BlockHash()
	hashNum := blockchain.HashToBig(&blockHash)
	if hashNum.Cmp(target) > 0 {
		log.Debugf("Block hash %064x is higher than "+
			"required target of %064x", hashNum, target)
		return false
	}
	return true
}

// This function takes in a start and end block header and uses the timestamps in each
// to calculate how much of a difficulty adjustment is needed. It returns a new compact
// difficulty target.
func calcDiffAdjust(start, end wire.BlockHeader, p *chaincfg.Params) uint32 {
	duration := end.Timestamp.UnixNano() - start.Timestamp.UnixNano()
	if duration < minRetargetTimespan {
		log.Debugf("Whoa there, block %s off-scale high 4X diff adjustment!",
			end.BlockHash().String())
		duration = minRetargetTimespan
	} else if duration > maxRetargetTimespan {
		log.Debugf("Uh-oh! block %s off-scale low 0.25X diff adjustment!\n",
			end.BlockHash().String())
		duration = maxRetargetTimespan
	}

	// calculation of new 32-byte difficulty target
	// first turn the previous target into a big int
	prevTarget := blockchain.CompactToBig(end.Bits)
	// new target is old * duration...
	newTarget := new(big.Int).Mul(prevTarget, big.NewInt(duration))
	// divided by 2 weeks
	newTarget.Div(newTarget, big.NewInt(int64(targetTimespan)))
	// clip again if above minimum target (too easy)
	if newTarget.Cmp(p.PowLimit) > 0 {
		fmt.Println(p.PowLimit.String(), newTarget.String())
		newTarget.Set(p.PowLimit)
	}

	return blockchain.BigToCompact(newTarget)
}
