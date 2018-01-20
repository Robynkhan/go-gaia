// Copyright 2015 The go-fairblock Authors
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/fairblock/go-fairblock/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("fbc/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("fbc/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("fbc/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("fbc/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("fbc/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("fbc/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("fbc/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("fbc/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("fbc/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("fbc/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("fbc/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("fbc/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("fbc/downloader/states/in")
	stateDropMeter = metrics.NewMeter("fbc/downloader/states/drop")
)
