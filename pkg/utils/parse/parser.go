// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package parse

type CGIType int

const (
	CGITypeNrCGI CGIType = iota
	CGITypeECGI
	CGITypeUnknown
)

func (c CGIType) String() string {
	return [...]string{"CGITypeNRCGI", "CGITypeECGI", "CGITypeUnknown"}[c]
}

