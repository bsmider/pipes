package factory

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewHop(binaryId string, timestamp time.Time) *Hop {
	return &Hop{
		BinaryId:  binaryId,
		Timestamp: timestamppb.New(timestamp),
	}
}
