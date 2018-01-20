// Copyright 2016 The go-fairblock Authors
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

package fbcclient

import "github.com/fairblock/go-fairblock"

// Verify that Client implements the fairblock interfaces.
var (
	_ = fairblock.ChainReader(&Client{})
	_ = fairblock.TransactionReader(&Client{})
	_ = fairblock.ChainStateReader(&Client{})
	_ = fairblock.ChainSyncReader(&Client{})
	_ = fairblock.ContractCaller(&Client{})
	_ = fairblock.GasEstimator(&Client{})
	_ = fairblock.GasPricer(&Client{})
	_ = fairblock.LogFilterer(&Client{})
	_ = fairblock.PendingStateReader(&Client{})
	// _ = fairblock.PendingStateEventer(&Client{})
	_ = fairblock.PendingContractCaller(&Client{})
)
