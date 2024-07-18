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

type ChainType byte

const (
	ChainVector ChainType = iota
	ChainPrime
	ChainMainNet
	ChainTestNet
)

func startSyncer(
	ctx context.Context, chainType ChainType, id int, baseDirectory string, addressesOfInterest []string,
) error {
	var (
		address      string
		networkMagic uint32
	)

	switch chainType {
	case ChainVector:
		address = "localhost:5200"
		networkMagic = uint32(1127)
	case ChainPrime:
		address = "localhost:5100"
		networkMagic = uint32(3311)
	case ChainMainNet:
		address = "backbone.cardano-mainnet.iohk.io:3001"
		networkMagic = uint32(764824073)
	case ChainTestNet:
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
			logger.Info("Tx has been processed", "tx", tx)
			for _, ot := range tx.Outputs {
				fmt.Println(tx.Hash.String(), "-", ot.Address)
			}
		}

		return dbs.MarkConfirmedTxsProcessed(unprocessedTxs)
	}

	indexerConfig := &indexer.BlockIndexerConfig{
		StartingBlockPoint: &indexer.BlockPoint{
			BlockSlot:   0,
			BlockHash:   [32]byte{},
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
			fmt.Println("syncer fatal err: ", err)

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
	syncerTimeout := time.Second * 10
	sequenceCount := 10
	chainType := ChainVector
	addressesOfInterest := []string{"vector_test1w2f6af09h85uxe63qn4vv6ywlnu9ttdlfpm9tz8snewthrgklp6sz",
		"vector_test1wfpm9w2zzzh5g2ayt66rm7hl0dgsvxuaxkudj00y3hqwn8sgfzpgj"}

	baseDirectory, err := os.MkdirTemp("", "syncer-test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer func() {
		os.RemoveAll(baseDirectory)
		os.Remove(baseDirectory)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChannel := make(chan os.Signal, 1)
	// Notify the signalChannel when the interrupt signal is received (Ctrl+C)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for i := 1; i <= sequenceCount; i++ {
		fmt.Println("starting syncer ", i, baseDirectory)

		timeOutContext, cancel := context.WithCancel(ctx)

		if err := startSyncer(timeOutContext, chainType, i, baseDirectory, addressesOfInterest); err != nil {
			fmt.Println("error", err)
		}

		select {
		case <-signalChannel:
			cancel()

			return
		case <-ctx.Done():
			cancel()

			return
		case <-time.After(syncerTimeout):
			fmt.Println("stopping syncer")

			cancel()
		}
	}
}
