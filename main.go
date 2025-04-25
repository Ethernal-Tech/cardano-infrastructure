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
	"github.com/Ethernal-Tech/cardano-infrastructure/indexer/gouroboros"
	"github.com/Ethernal-Tech/cardano-infrastructure/logger"
	"github.com/hashicorp/go-hclog"
)

func startSyncer(ctx context.Context, chainType int, id int, baseDirectory string) error {
	var (
		address             string
		networkMagic        uint32
		addressesOfInterest []string
	)

	switch chainType {
	case 0:
		address = "localhost:5200"
		networkMagic = uint32(1127)
		addressesOfInterest = []string{}
	case 1:
		address = "localhost:5100"
		networkMagic = uint32(3311)
		addressesOfInterest = []string{}
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

	indexerConfig := &indexer.BlockIndexerConfig{
		StartingBlockPoint: &indexer.BlockPoint{
			BlockSlot: 0,
			BlockHash: [32]byte{},
		},
		AddressCheck:            indexer.AddressCheckAll,
		ConfirmationBlockCount:  10,
		AddressesOfInterest:     addressesOfInterest,
		SoftDeleteUtxo:          false,
		KeepAllTxOutputsInDB:    false,
		KeepAllTxsHashesInBlock: true,
	}
	syncerConfig := &gouroboros.BlockSyncerConfig{
		NetworkMagic:   networkMagic,
		NodeAddress:    address,
		RestartOnError: true,
		RestartDelay:   time.Second * 2,
		KeepAlive:      true,
	}

	indexerObj := indexer.NewBlockIndexer(indexerConfig, confirmedBlockHandler, dbs, logger.Named("block_indexer"))
	syncer := gouroboros.NewBlockSyncer(syncerConfig, indexerObj, logger.Named("block_syncer"))

	go func() {
		select {
		case <-ctx.Done():
			syncer.Close()
			dbs.Close()
		case err := <-syncer.ErrorCh():
			logger.Error("syncer fatal err", "err", err)

			dbs.Close()
		}
	}()

	err = syncer.Sync()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	syncerTimeout := time.Second * 50
	sequenceCount := 10

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

		if err := startSyncer(timeOutContext, 3, i, baseDirectory); err != nil {
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
