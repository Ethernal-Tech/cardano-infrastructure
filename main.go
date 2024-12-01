package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/Ethernal-Tech/cardano-infrastructure/indexer/db"
	"github.com/Ethernal-Tech/cardano-infrastructure/logger"
	"github.com/hashicorp/go-hclog"
)

func startSyncer(
	ctx context.Context, chainType int, id int, baseDirectory string, addressesOfInterest []string,
	blockHashStr string, blockSlot uint64,
) error {
	var (
		address      string
		networkMagic uint32
	)

	switch chainType {
	case 0:
		address = "localhost:5200"
		networkMagic = uint32(1127)
	case 1:
		address = "localhost:5100"
		networkMagic = uint32(3311)
	case 2:
		address = "backbone.cardano-mainnet.iohk.io:3001"
		networkMagic = uint32(764824073)
	case 3:
		address = "preprod-node.play.dev.cardano.org:3001"
		networkMagic = 1
	}

	logger, err := logger.NewLogger(logger.LoggerConfig{
		LogLevel:      hclog.Debug,
		JSONLogFormat: false,
		AppendFile:    true,
		LogFilePath:   filepath.Join(baseDirectory, fmt.Sprintf("logs-%d.log", id)),
	})
	if err != nil {
		return err
	}

	dbs, err := db.NewDatabaseInit("", filepath.Join(baseDirectory, fmt.Sprintf("burek-%d.db", id)))
	if err != nil {
		return err
	}

	confirmedBlockHandler := func(confirmedBlock *indexer.CardanoBlock, txs []*indexer.Tx) error {
		logger.Info("Confirmed block",
			"hash", hex.EncodeToString(confirmedBlock.Hash[:]), "slot", confirmedBlock.Slot,
			"allTxs", len(confirmedBlock.Txs), "ourTxs", len(txs))

		unprocessedTxs, err := dbs.GetUnprocessedConfirmedTxs(0)
		if err != nil {
			return err
		}

		for _, tx := range unprocessedTxs {
			logger.Info("Tx has been processed", "tx", tx.String())
		}

		return dbs.MarkConfirmedTxsProcessed(unprocessedTxs)
	}

	blockHash, _ := hex.DecodeString(blockHashStr)
	if blockHashStr == "" {
		blockHash = make([]byte, 32)
	}

	indexerConfig := &indexer.BlockIndexerConfig{
		StartingBlockPoint: &indexer.BlockPoint{
			BlockSlot:   blockSlot,
			BlockHash:   [32]byte(blockHash),
			BlockNumber: 0,
		},
		AddressCheck:            indexer.AddressCheckAll,
		ConfirmationBlockCount:  10,
		AddressesOfInterest:     addressesOfInterest,
		SoftDeleteUtxo:          false,
		KeepAllTxOutputsInDB:    false,
		KeepAllTxsHashesInBlock: true,
	}
	syncerConfig := &indexer.BlockSyncerConfig{
		NetworkMagic:   networkMagic,
		NodeAddress:    address,
		RestartOnError: true,
		RestartDelay:   time.Second * 2,
		KeepAlive:      true,
	}

	indexerObj := indexer.NewBlockIndexer(indexerConfig, confirmedBlockHandler, dbs, logger.Named("block_indexer"))
	syncer := indexer.NewBlockSyncer(syncerConfig, indexerObj, logger.Named("block_syncer"))

	go func() {
		select {
		case <-ctx.Done():
			syncer.Close()
			dbs.Close()
		case err := <-syncer.ErrorCh():
			logger.Error("syncer fatal err", "err", err)

			indexerObj.Close()
			dbs.Close()
		}
	}()

	go indexerObj.Start(ctx)

	err = syncer.Sync()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	syncerTimeout := time.Second * 250
	sequenceCount := 10
	addressesOfInterest := []string{"addr_test1wr64gtafm8rpkndue4ck2nx95u4flhwf643l2qmg9emjajg2ww0nj"}
	blockHash := "28de818c3aa1103ab12964307441d2d12790e04d5869789be9d4de1a01014a07"
	blockSlot := uint64(75130796)

	baseDirectory, err := os.MkdirTemp("", "syncer-test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer os.RemoveAll(baseDirectory)

	signalChannel := make(chan os.Signal, 1)
	// Notify the signalChannel when the interrupt signal is received (Ctrl+C)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for i := 1; i <= sequenceCount; i++ {
		fmt.Println("starting syncer ", i, baseDirectory)

		timeOutContext, cancel := context.WithTimeout(context.Background(), syncerTimeout)

		err := startSyncer(timeOutContext, 3, i, baseDirectory, addressesOfInterest, blockHash, blockSlot)
		if err != nil {
			fmt.Println("syncer error", err)
		}

		select {
		case <-signalChannel:
			cancel()

			return
		case <-timeOutContext.Done():
		}
	}
}
