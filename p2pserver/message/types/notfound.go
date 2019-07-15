/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package types

import (
	"io"

	"github.com/ontio/multi-chain/common"
	comm "github.com/ontio/multi-chain/p2pserver/common"
)

type NotFound struct {
	Hash common.Uint256
}

//Serialize message payload
func (this NotFound) Serialization(sink *common.ZeroCopySink) error {
	sink.WriteHash(this.Hash)
	return nil
}

func (this NotFound) CmdType() string {
	return comm.NOT_FOUND_TYPE
}

//Deserialize message payload
func (this *NotFound) Deserialization(source *common.ZeroCopySource) error {
	var eof bool
	this.Hash, eof = source.NextHash()
	if eof {
		return io.ErrUnexpectedEOF
	}

	return nil
}
