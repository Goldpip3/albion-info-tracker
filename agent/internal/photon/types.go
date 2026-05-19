package photon

// commandType identifies the kind of Photon command in a UDP packet.
type commandType byte

const (
	cmdDisconnect     commandType = 4
	cmdSendReliable   commandType = 6
	cmdSendUnreliable commandType = 7
	cmdSendFragment   commandType = 8
)

// messageType identifies whether a SendReliable payload is an event,
// operation request, or operation response.
type messageType byte

const (
	msgOperationRequest  messageType = 2
	msgOperationResponse messageType = 3
	msgEvent             messageType = 4
)

// p18Type is the Protocol18 type-code prefix preceding every value.
type p18Type byte

const (
	p18Unknown             p18Type = 0
	p18Boolean             p18Type = 2
	p18Byte                p18Type = 3
	p18Short               p18Type = 4
	p18Float               p18Type = 5
	p18Double              p18Type = 6
	p18String              p18Type = 7
	p18Null                p18Type = 8
	p18CompressedInt       p18Type = 9
	p18CompressedLong      p18Type = 10
	p18Int1                p18Type = 11
	p18Int1Negative        p18Type = 12
	p18Int2                p18Type = 13
	p18Int2Negative        p18Type = 14
	p18Long1               p18Type = 15
	p18Long1Negative       p18Type = 16
	p18Long2               p18Type = 17
	p18Long2Negative       p18Type = 18
	p18Custom              p18Type = 19
	p18Dictionary          p18Type = 20
	p18Hashtable           p18Type = 21
	p18ObjectArray         p18Type = 23
	p18OperationRequest    p18Type = 24
	p18OperationResponse   p18Type = 25
	p18EventData           p18Type = 26
	p18BooleanFalse        p18Type = 27
	p18BooleanTrue         p18Type = 28
	p18ShortZero           p18Type = 29
	p18IntZero             p18Type = 30
	p18LongZero            p18Type = 31
	p18FloatZero           p18Type = 32
	p18DoubleZero          p18Type = 33
	p18ByteZero            p18Type = 34
	p18Array               p18Type = 64
	p18BooleanArray        p18Type = 66
	p18ByteArray           p18Type = 67
	p18ShortArray          p18Type = 68
	p18FloatArray          p18Type = 69
	p18DoubleArray         p18Type = 70
	p18StringArray         p18Type = 71
	p18CompressedIntArray  p18Type = 73
	p18CompressedLongArray p18Type = 74
	p18CustomTypeArray     p18Type = 83
	p18DictionaryArray     p18Type = 84
	p18HashtableArray      p18Type = 85
	p18CustomTypeSlim      p18Type = 128
	p18MaxSlimCustom       p18Type = 228
)

// CustomType is an opaque blob with a Photon-assigned type code. The most
// common custom type is GUID (typeCode 0x07, 16 bytes). Handlers can inspect
// the TypeCode to decide how to interpret Data.
type CustomType struct {
	TypeCode byte
	Data     []byte
}

// EventData is one game event delivered by the server.
type EventData struct {
	Code       byte
	Parameters map[byte]any
}

// OperationRequest is a request sent by the client.
type OperationRequest struct {
	OperationCode byte
	Parameters    map[byte]any
}

// OperationResponse is the server's reply to an OperationRequest.
type OperationResponse struct {
	OperationCode byte
	ReturnCode    int16
	DebugMessage  string
	Parameters    map[byte]any
}
