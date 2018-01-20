// Copyright 2017 The go-fairblock Authors
// This file is part of the go-fairblock library.
//
// The go-fairblock library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-fairblock library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-fairblock library. If not, see <http://www.gnu.org/licenses/>.

package fbc

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/fairblock/go-fairblock/common"
	"github.com/fairblock/go-fairblock/common/hexutil"
	"github.com/fairblock/go-fairblock/core"
	"github.com/fairblock/go-fairblock/fbc/downloader"
	"github.com/fairblock/go-fairblock/fbc/gasprice"
	"github.com/fairblock/go-fairblock/params"
)

// DefaultConfig contains default settings for use on the Fairblock main net.
var DefaultConfig = Config{
	SyncMode:             downloader.FastSync,
	FbcashCacheDir:       "fbcash",
	FbcashCachesInMem:    2,
	FbcashCachesOnDisk:   3,
	FbcashDatasetsInMem:  1,
	FbcashDatasetsOnDisk: 2,
	NetworkId:            1,
	LightPeers:           20,
	DatabaseCache:        128,
	GasPrice:             big.NewInt(18 * params.Shannon),

	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     10,
		Percentile: 50,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.FbcashDatasetDir = filepath.Join(home, "AppData", "Fbcash")
	} else {
		DefaultConfig.FbcashDatasetDir = filepath.Join(home, ".fbcash")
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Fairblock main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

	// Mining-related options
	Fairblockbase    common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

	// Fbcash options
	FbcashCacheDir       string
	FbcashCachesInMem    int
	FbcashCachesOnDisk   int
	FbcashDatasetDir     string
	FbcashDatasetsInMem  int
	FbcashDatasetsOnDisk int

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot   string `toml:"-"`
	PowFake   bool   `toml:"-"`
	PowTest   bool   `toml:"-"`
	PowShared bool   `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
