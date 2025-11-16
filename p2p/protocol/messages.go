package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"time"

	"github.com/cosmos/gogoproto/proto"
)

// Protocol version constants
const (
	ProtocolVersion1       uint8 = 1
	CurrentProtocolVersion       = ProtocolVersion1

	// Maximum message sizes
	MaxMessageSize     = 10 * 1024 * 1024 // 10 MB
	MaxBlockSize       = 5 * 1024 * 1024  // 5 MB
	MaxTxSize          = 1 * 1024 * 1024  // 1 MB
	MaxPeerAddressList = 1000
)

// MessageType defines the type of P2P message
type MessageType uint8

const (
	// Handshake messages
	MsgTypeHandshake    MessageType = 0x01
	MsgTypeHandshakeAck MessageType = 0x02

	// Block messages
	MsgTypeNewBlock      MessageType = 0x10
	MsgTypeBlockRequest  MessageType = 0x11
	MsgTypeBlockResponse MessageType = 0x12
	MsgTypeBlockAnnounce MessageType = 0x13

	// Transaction messages
	MsgTypeNewTx       MessageType = 0x20
	MsgTypeTxRequest   MessageType = 0x21
	MsgTypeTxResponse  MessageType = 0x22
	MsgTypeMempoolSync MessageType = 0x23

	// Sync messages
	MsgTypeSyncRequest     MessageType = 0x30
	MsgTypeSyncResponse    MessageType = 0x31
	MsgTypeStateRequest    MessageType = 0x32
	MsgTypeStateResponse   MessageType = 0x33
	MsgTypeCatchupRequest  MessageType = 0x34
	MsgTypeCatchupResponse MessageType = 0x35

	// Peer discovery messages
	MsgTypePeerRequest  MessageType = 0x40
	MsgTypePeerResponse MessageType = 0x41
	MsgTypePeerAnnounce MessageType = 0x42

	// Consensus messages
	MsgTypeVote     MessageType = 0x50
	MsgTypeProposal MessageType = 0x51
	MsgTypeCommit   MessageType = 0x52

	// Status messages
	MsgTypePing   MessageType = 0x60
	MsgTypePong   MessageType = 0x61
	MsgTypeStatus MessageType = 0x62

	// Error messages
	MsgTypeError MessageType = 0xFF
)

// String returns the string representation of MessageType
func (mt MessageType) String() string {
	switch mt {
	case MsgTypeHandshake:
		return "Handshake"
	case MsgTypeHandshakeAck:
		return "HandshakeAck"
	case MsgTypeNewBlock:
		return "NewBlock"
	case MsgTypeBlockRequest:
		return "BlockRequest"
	case MsgTypeBlockResponse:
		return "BlockResponse"
	case MsgTypeBlockAnnounce:
		return "BlockAnnounce"
	case MsgTypeNewTx:
		return "NewTx"
	case MsgTypeTxRequest:
		return "TxRequest"
	case MsgTypeTxResponse:
		return "TxResponse"
	case MsgTypeMempoolSync:
		return "MempoolSync"
	case MsgTypeSyncRequest:
		return "SyncRequest"
	case MsgTypeSyncResponse:
		return "SyncResponse"
	case MsgTypeStateRequest:
		return "StateRequest"
	case MsgTypeStateResponse:
		return "StateResponse"
	case MsgTypeCatchupRequest:
		return "CatchupRequest"
	case MsgTypeCatchupResponse:
		return "CatchupResponse"
	case MsgTypePeerRequest:
		return "PeerRequest"
	case MsgTypePeerResponse:
		return "PeerResponse"
	case MsgTypePeerAnnounce:
		return "PeerAnnounce"
	case MsgTypeVote:
		return "Vote"
	case MsgTypeProposal:
		return "Proposal"
	case MsgTypeCommit:
		return "Commit"
	case MsgTypePing:
		return "Ping"
	case MsgTypePong:
		return "Pong"
	case MsgTypeStatus:
		return "Status"
	case MsgTypeError:
		return "Error"
	default:
		return fmt.Sprintf("Unknown(%d)", mt)
	}
}

// Message is the base interface for all P2P messages
type Message interface {
	Type() MessageType
	Validate() error
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// MessageHeader contains common header fields
type MessageHeader struct {
	Version    uint8
	Type       MessageType
	Flags      uint8
	Timestamp  int64
	PayloadLen uint32
	Checksum   uint32
}

// MessageEnvelope wraps a message with routing and metadata
type MessageEnvelope struct {
	Header  MessageHeader
	Payload []byte
}

// Handshake message for initial peer connection
type HandshakeMessage struct {
	ProtocolVersion uint8
	ChainID         string
	NodeID          string
	ListenAddr      string
	Capabilities    []string
	GenesisHash     []byte
	BestHeight      int64
	BestHash        []byte
	Timestamp       int64
}

func (m *HandshakeMessage) Type() MessageType { return MsgTypeHandshake }

func (m *HandshakeMessage) Validate() error {
	if m.ProtocolVersion != CurrentProtocolVersion {
		return fmt.Errorf("unsupported protocol version: %d", m.ProtocolVersion)
	}
	if m.ChainID == "" {
		return errors.New("chain_id is required")
	}
	if m.NodeID == "" {
		return errors.New("node_id is required")
	}
	if len(m.GenesisHash) == 0 {
		return errors.New("genesis_hash is required")
	}
	return nil
}

func (m *HandshakeMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write protocol version
	if err := binary.Write(buf, binary.BigEndian, m.ProtocolVersion); err != nil {
		return nil, err
	}

	// Write strings with length prefix
	writeString := func(s string) error {
		if err := binary.Write(buf, binary.BigEndian, uint32(len(s))); err != nil {
			return err
		}
		_, err := buf.WriteString(s)
		return err
	}

	if err := writeString(m.ChainID); err != nil {
		return nil, err
	}
	if err := writeString(m.NodeID); err != nil {
		return nil, err
	}
	if err := writeString(m.ListenAddr); err != nil {
		return nil, err
	}

	// Write capabilities
	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Capabilities))); err != nil {
		return nil, err
	}
	for _, cap := range m.Capabilities {
		if err := writeString(cap); err != nil {
			return nil, err
		}
	}

	// Write genesis hash
	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.GenesisHash))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.GenesisHash); err != nil {
		return nil, err
	}

	// Write best height and hash
	if err := binary.Write(buf, binary.BigEndian, m.BestHeight); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.BestHash))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.BestHash); err != nil {
		return nil, err
	}

	// Write timestamp
	if err := binary.Write(buf, binary.BigEndian, m.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *HandshakeMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	// Read protocol version
	if err := binary.Read(buf, binary.BigEndian, &m.ProtocolVersion); err != nil {
		return err
	}

	// Read strings with length prefix
	readString := func() (string, error) {
		var length uint32
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return "", err
		}
		if length > MaxMessageSize {
			return "", errors.New("string too long")
		}
		strBytes := make([]byte, length)
		if _, err := buf.Read(strBytes); err != nil {
			return "", err
		}
		return string(strBytes), nil
	}

	var err error
	if m.ChainID, err = readString(); err != nil {
		return err
	}
	if m.NodeID, err = readString(); err != nil {
		return err
	}
	if m.ListenAddr, err = readString(); err != nil {
		return err
	}

	// Read capabilities
	var capCount uint32
	if err := binary.Read(buf, binary.BigEndian, &capCount); err != nil {
		return err
	}
	if capCount > 100 {
		return errors.New("too many capabilities")
	}
	m.Capabilities = make([]string, capCount)
	for i := uint32(0); i < capCount; i++ {
		if m.Capabilities[i], err = readString(); err != nil {
			return err
		}
	}

	// Read genesis hash
	var hashLen uint32
	if err := binary.Read(buf, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	if hashLen > 128 {
		return errors.New("genesis hash too long")
	}
	m.GenesisHash = make([]byte, hashLen)
	if _, err := buf.Read(m.GenesisHash); err != nil {
		return err
	}

	// Read best height and hash
	if err := binary.Read(buf, binary.BigEndian, &m.BestHeight); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	if hashLen > 128 {
		return errors.New("best hash too long")
	}
	m.BestHash = make([]byte, hashLen)
	if _, err := buf.Read(m.BestHash); err != nil {
		return err
	}

	// Read timestamp
	if err := binary.Read(buf, binary.BigEndian, &m.Timestamp); err != nil {
		return err
	}

	return nil
}

// HandshakeAckMessage acknowledges a handshake
type HandshakeAckMessage struct {
	Accepted bool
	Reason   string
	NodeID   string
}

func (m *HandshakeAckMessage) Type() MessageType { return MsgTypeHandshakeAck }

func (m *HandshakeAckMessage) Validate() error {
	if !m.Accepted && m.Reason == "" {
		return errors.New("reason required for rejected handshake")
	}
	return nil
}

func (m *HandshakeAckMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	accepted := byte(0)
	if m.Accepted {
		accepted = 1
	}
	if err := buf.WriteByte(accepted); err != nil {
		return nil, err
	}

	writeString := func(s string) error {
		if err := binary.Write(buf, binary.BigEndian, uint32(len(s))); err != nil {
			return err
		}
		_, err := buf.WriteString(s)
		return err
	}

	if err := writeString(m.Reason); err != nil {
		return nil, err
	}
	if err := writeString(m.NodeID); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *HandshakeAckMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	accepted, err := buf.ReadByte()
	if err != nil {
		return err
	}
	m.Accepted = accepted == 1

	readString := func() (string, error) {
		var length uint32
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return "", err
		}
		if length > MaxMessageSize {
			return "", errors.New("string too long")
		}
		strBytes := make([]byte, length)
		if _, err := buf.Read(strBytes); err != nil {
			return "", err
		}
		return string(strBytes), nil
	}

	if m.Reason, err = readString(); err != nil {
		return err
	}
	if m.NodeID, err = readString(); err != nil {
		return err
	}

	return nil
}

// BlockMessage carries block data
type BlockMessage struct {
	Height    int64
	Hash      []byte
	BlockData []byte
	Source    string
}

func (m *BlockMessage) Type() MessageType { return MsgTypeNewBlock }

func (m *BlockMessage) Validate() error {
	if m.Height < 0 {
		return errors.New("invalid block height")
	}
	if len(m.Hash) == 0 {
		return errors.New("block hash is required")
	}
	if len(m.BlockData) == 0 {
		return errors.New("block data is required")
	}
	if len(m.BlockData) > MaxBlockSize {
		return fmt.Errorf("block data too large: %d > %d", len(m.BlockData), MaxBlockSize)
	}
	return nil
}

func (m *BlockMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, m.Height); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Hash))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.Hash); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.BlockData))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.BlockData); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Source))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(m.Source); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *BlockMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.BigEndian, &m.Height); err != nil {
		return err
	}

	var hashLen uint32
	if err := binary.Read(buf, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	if hashLen > 128 {
		return errors.New("hash too long")
	}
	m.Hash = make([]byte, hashLen)
	if _, err := buf.Read(m.Hash); err != nil {
		return err
	}

	var dataLen uint32
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return err
	}
	if dataLen > MaxBlockSize {
		return errors.New("block data too large")
	}
	m.BlockData = make([]byte, dataLen)
	if _, err := buf.Read(m.BlockData); err != nil {
		return err
	}

	var sourceLen uint32
	if err := binary.Read(buf, binary.BigEndian, &sourceLen); err != nil {
		return err
	}
	if sourceLen > 1024 {
		return errors.New("source too long")
	}
	sourceBytes := make([]byte, sourceLen)
	if _, err := buf.Read(sourceBytes); err != nil {
		return err
	}
	m.Source = string(sourceBytes)

	return nil
}

// TxMessage carries transaction data
type TxMessage struct {
	TxHash []byte
	TxData []byte
	Source string
}

func (m *TxMessage) Type() MessageType { return MsgTypeNewTx }

func (m *TxMessage) Validate() error {
	if len(m.TxHash) == 0 {
		return errors.New("transaction hash is required")
	}
	if len(m.TxData) == 0 {
		return errors.New("transaction data is required")
	}
	if len(m.TxData) > MaxTxSize {
		return fmt.Errorf("transaction too large: %d > %d", len(m.TxData), MaxTxSize)
	}
	return nil
}

func (m *TxMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.TxHash))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.TxHash); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.TxData))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.TxData); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Source))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(m.Source); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *TxMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	var hashLen uint32
	if err := binary.Read(buf, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	if hashLen > 128 {
		return errors.New("hash too long")
	}
	m.TxHash = make([]byte, hashLen)
	if _, err := buf.Read(m.TxHash); err != nil {
		return err
	}

	var dataLen uint32
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return err
	}
	if dataLen > MaxTxSize {
		return errors.New("transaction data too large")
	}
	m.TxData = make([]byte, dataLen)
	if _, err := buf.Read(m.TxData); err != nil {
		return err
	}

	var sourceLen uint32
	if err := binary.Read(buf, binary.BigEndian, &sourceLen); err != nil {
		return err
	}
	if sourceLen > 1024 {
		return errors.New("source too long")
	}
	sourceBytes := make([]byte, sourceLen)
	if _, err := buf.Read(sourceBytes); err != nil {
		return err
	}
	m.Source = string(sourceBytes)

	return nil
}

// PeerAddress represents a peer's network address
type PeerAddress struct {
	ID        string
	IP        string
	Port      uint16
	Timestamp int64
}

// PeerListMessage carries a list of known peers
type PeerListMessage struct {
	Peers []PeerAddress
}

func (m *PeerListMessage) Type() MessageType { return MsgTypePeerResponse }

func (m *PeerListMessage) Validate() error {
	if len(m.Peers) > MaxPeerAddressList {
		return fmt.Errorf("too many peers: %d > %d", len(m.Peers), MaxPeerAddressList)
	}
	return nil
}

func (m *PeerListMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Peers))); err != nil {
		return nil, err
	}

	for _, peer := range m.Peers {
		if err := binary.Write(buf, binary.BigEndian, uint32(len(peer.ID))); err != nil {
			return nil, err
		}
		if _, err := buf.WriteString(peer.ID); err != nil {
			return nil, err
		}

		if err := binary.Write(buf, binary.BigEndian, uint32(len(peer.IP))); err != nil {
			return nil, err
		}
		if _, err := buf.WriteString(peer.IP); err != nil {
			return nil, err
		}

		if err := binary.Write(buf, binary.BigEndian, peer.Port); err != nil {
			return nil, err
		}

		if err := binary.Write(buf, binary.BigEndian, peer.Timestamp); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *PeerListMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	var peerCount uint32
	if err := binary.Read(buf, binary.BigEndian, &peerCount); err != nil {
		return err
	}
	if peerCount > MaxPeerAddressList {
		return errors.New("too many peers")
	}

	m.Peers = make([]PeerAddress, peerCount)

	for i := uint32(0); i < peerCount; i++ {
		var length uint32

		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return err
		}
		if length > 256 {
			return errors.New("peer ID too long")
		}
		idBytes := make([]byte, length)
		if _, err := buf.Read(idBytes); err != nil {
			return err
		}
		m.Peers[i].ID = string(idBytes)

		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return err
		}
		if length > 64 {
			return errors.New("IP too long")
		}
		ipBytes := make([]byte, length)
		if _, err := buf.Read(ipBytes); err != nil {
			return err
		}
		m.Peers[i].IP = string(ipBytes)

		if err := binary.Read(buf, binary.BigEndian, &m.Peers[i].Port); err != nil {
			return err
		}

		if err := binary.Read(buf, binary.BigEndian, &m.Peers[i].Timestamp); err != nil {
			return err
		}
	}

	return nil
}

// StatusMessage carries node status
type StatusMessage struct {
	Height      int64
	BestHash    []byte
	Timestamp   int64
	TxPoolSize  int
	PeerCount   int
	Syncing     bool
	NetworkLoad float64
}

func (m *StatusMessage) Type() MessageType { return MsgTypeStatus }

func (m *StatusMessage) Validate() error {
	if m.Height < 0 {
		return errors.New("invalid height")
	}
	return nil
}

func (m *StatusMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, m.Height); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.BestHash))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.BestHash); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, m.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, int32(m.TxPoolSize)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, int32(m.PeerCount)); err != nil {
		return nil, err
	}

	syncing := byte(0)
	if m.Syncing {
		syncing = 1
	}
	if err := buf.WriteByte(syncing); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, m.NetworkLoad); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *StatusMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.BigEndian, &m.Height); err != nil {
		return err
	}

	var hashLen uint32
	if err := binary.Read(buf, binary.BigEndian, &hashLen); err != nil {
		return err
	}
	if hashLen > 128 {
		return errors.New("hash too long")
	}
	m.BestHash = make([]byte, hashLen)
	if _, err := buf.Read(m.BestHash); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &m.Timestamp); err != nil {
		return err
	}

	var txPoolSize, peerCount int32
	if err := binary.Read(buf, binary.BigEndian, &txPoolSize); err != nil {
		return err
	}
	m.TxPoolSize = int(txPoolSize)

	if err := binary.Read(buf, binary.BigEndian, &peerCount); err != nil {
		return err
	}
	m.PeerCount = int(peerCount)

	syncing, err := buf.ReadByte()
	if err != nil {
		return err
	}
	m.Syncing = syncing == 1

	if err := binary.Read(buf, binary.BigEndian, &m.NetworkLoad); err != nil {
		return err
	}

	return nil
}

// ErrorMessage carries error information
type ErrorMessage struct {
	Code    uint32
	Message string
}

func (m *ErrorMessage) Type() MessageType { return MsgTypeError }

func (m *ErrorMessage) Validate() error {
	return nil
}

func (m *ErrorMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, m.Code); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(m.Message))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(m.Message); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *ErrorMessage) Unmarshal(data []byte) error {
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.BigEndian, &m.Code); err != nil {
		return err
	}

	var length uint32
	if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
		return err
	}
	if length > MaxMessageSize {
		return errors.New("message too long")
	}

	msgBytes := make([]byte, length)
	if _, err := buf.Read(msgBytes); err != nil {
		return err
	}
	m.Message = string(msgBytes)

	return nil
}

// Envelope methods

// MarshalEnvelope serializes a message into an envelope
func MarshalEnvelope(msg Message) (*MessageEnvelope, error) {
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	payload, err := msg.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	if len(payload) > MaxMessageSize {
		return nil, fmt.Errorf("payload too large: %d > %d", len(payload), MaxMessageSize)
	}

	checksum := crc32.ChecksumIEEE(payload)

	envelope := &MessageEnvelope{
		Header: MessageHeader{
			Version:    CurrentProtocolVersion,
			Type:       msg.Type(),
			Flags:      0,
			Timestamp:  time.Now().Unix(),
			PayloadLen: uint32(len(payload)),
			Checksum:   checksum,
		},
		Payload: payload,
	}

	return envelope, nil
}

// WriteEnvelope writes an envelope to a writer
func WriteEnvelope(w io.Writer, envelope *MessageEnvelope) error {
	// Write header
	if err := binary.Write(w, binary.BigEndian, envelope.Header.Version); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, envelope.Header.Type); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, envelope.Header.Flags); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, envelope.Header.Timestamp); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, envelope.Header.PayloadLen); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, envelope.Header.Checksum); err != nil {
		return err
	}

	// Write payload
	if _, err := w.Write(envelope.Payload); err != nil {
		return err
	}

	return nil
}

// ReadEnvelope reads an envelope from a reader
func ReadEnvelope(r io.Reader) (*MessageEnvelope, error) {
	envelope := &MessageEnvelope{}

	// Read header
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.Type); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.Flags); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.PayloadLen); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &envelope.Header.Checksum); err != nil {
		return nil, err
	}

	// Validate header
	if envelope.Header.Version != CurrentProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", envelope.Header.Version)
	}
	if envelope.Header.PayloadLen > MaxMessageSize {
		return nil, fmt.Errorf("payload too large: %d > %d", envelope.Header.PayloadLen, MaxMessageSize)
	}

	// Read payload
	envelope.Payload = make([]byte, envelope.Header.PayloadLen)
	if _, err := io.ReadFull(r, envelope.Payload); err != nil {
		return nil, err
	}

	// Verify checksum
	checksum := crc32.ChecksumIEEE(envelope.Payload)
	if checksum != envelope.Header.Checksum {
		return nil, errors.New("checksum mismatch")
	}

	return envelope, nil
}

// UnmarshalMessage creates a message from an envelope
func UnmarshalMessage(envelope *MessageEnvelope) (Message, error) {
	var msg Message

	switch envelope.Header.Type {
	case MsgTypeHandshake:
		msg = &HandshakeMessage{}
	case MsgTypeHandshakeAck:
		msg = &HandshakeAckMessage{}
	case MsgTypeNewBlock:
		msg = &BlockMessage{}
	case MsgTypeNewTx:
		msg = &TxMessage{}
	case MsgTypePeerResponse:
		msg = &PeerListMessage{}
	case MsgTypeStatus:
		msg = &StatusMessage{}
	case MsgTypeError:
		msg = &ErrorMessage{}
	default:
		return nil, fmt.Errorf("unknown message type: %d", envelope.Header.Type)
	}

	if err := msg.Unmarshal(envelope.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return msg, nil
}

// Helper to avoid unused import error
var _ = proto.Marshal
