package photonports

var udp = map[uint16]struct{}{
	5055: {},
	5056: {},
	5058: {},
}

func IsPhoton(port uint16) bool {
	_, ok := udp[port]
	return ok
}

func LooksLikePhotonPayload(payload []byte) bool {
	if len(payload) < 3 {
		return false
	}
	b := payload[0]
	return b == 0xF1 || b == 0xF2 || b == 0xFE
}
