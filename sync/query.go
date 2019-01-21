package sync

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/crypto/hash"
	syncpb "github.com/medibloc/go-medibloc/sync/pb"
)

// FindBaseQuery is a query to find base block
type FindBaseQuery struct {
	pb *syncpb.FindBaseRequest
}

// Hash returns hash
func (q *FindBaseQuery) Hash() []byte {
	return hash.Sha3256Pb(q.pb)
}

// MessageType returns message type
func (*FindBaseQuery) MessageType() string {
	return BaseSearch
}

// ProtoBuf returns protobuf
func (q *FindBaseQuery) ProtoBuf() proto.Message {
	return q.pb
}

// SetID sets query id
func (q *FindBaseQuery) SetID(id string) {
	q.pb.Id = id
}

func newFindBaseRequest(d *download) *FindBaseQuery {
	return &FindBaseQuery{
		pb: &syncpb.FindBaseRequest{
			TryHeight:    d.try,
			TargetHeight: d.targetHeight,
			Timestamp:    time.Now().UnixNano(),
		}}
}

// DownloadByHeightQuery is a query to download block
type DownloadByHeightQuery struct {
	pb *syncpb.BlockByHeightRequest
}

// Hash returns hash
func (q *DownloadByHeightQuery) Hash() []byte {
	return hash.Sha3256Pb(q.pb)
}

// MessageType returns message type
func (*DownloadByHeightQuery) MessageType() string {
	return BlockRequest
}

// ProtoBuf returns proto buffer
func (q *DownloadByHeightQuery) ProtoBuf() proto.Message {
	return q.pb
}

// SetID sets query id
func (q *DownloadByHeightQuery) SetID(id string) {
	q.pb.Id = id
}

func newBlockByHeightRequest(d *download, height uint64) *DownloadByHeightQuery {
	return &DownloadByHeightQuery{
		pb: &syncpb.BlockByHeightRequest{
			TargetHeight: d.targetHeight,
			BlockHeight:  height,
			Timestamp:    time.Now().Unix(),
		}}
}
