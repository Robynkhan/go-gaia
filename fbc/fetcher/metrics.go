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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/fairblock/go-fairblock/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("fbc/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("fbc/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("fbc/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("fbc/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("fbc/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("fbc/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("fbc/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("fbc/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("fbc/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("fbc/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("fbc/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("fbc/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("fbc/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("fbc/fetcher/filter/bodies/out")
)
