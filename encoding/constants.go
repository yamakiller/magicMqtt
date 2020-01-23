package encoding

//PType 数据包类型
type PType int

const (
	//PTypeReserved1 Reserved1 packet
	PTypeReserved1 PType = 0
	//PTypeConnect Connect packet
	PTypeConnect PType = 1
	//PTypeConnack Connack packet
	PTypeConnack PType = 2
	//PTypePublish Publish packet
	PTypePublish PType = 3
	//PTypePuback Puback packet
	PTypePuback PType = 4
	//PTypePubrec Pubrec packet
	PTypePubrec PType = 5
	//PTypePubrel Pubrel packet
	PTypePubrel PType = 6
	//PTypePubcomp Pubcomp packet
	PTypePubcomp PType = 7
	//PTypeSubscribe Subscribe packet
	PTypeSubscribe PType = 8
	//PTypeSuback Suback  packet
	PTypeSuback PType = 9
	//PTypeUnsubscribe Unsubscribe packet
	PTypeUnsubscribe PType = 10
	//PTypeUnsuback Unsuback packet
	PTypeUnsuback PType = 11
	//PTypePingreq  Pingreq pakcet
	PTypePingreq PType = 12
	//PTypePingresp PingResp packet
	PTypePingresp PType = 13
	//PTypeDisconnect Disconnect packet
	PTypeDisconnect PType = 14
	//PTypeReserved2 Reserved2 packet
	PTypeReserved2 PType = 15
)

//RCode 返回码
type RCode int

const (
	//ConnectionAccepted Return Code
	ConnectionAccepted RCode = 1
	//ConnectionRefusedUnacceptableProtocolVersion Return Code
	ConnectionRefusedUnacceptableProtocolVersion RCode = 2
	//ConnectionRefusedIdentifierRejected Return Code
	ConnectionRefusedIdentifierRejected RCode = 3
	//ConnectionRefusedServerUnavailable Return Code
	ConnectionRefusedServerUnavailable RCode = 4
	//ConnectionRefusedBadUserNameOrPassword Return Code
	ConnectionRefusedBadUserNameOrPassword RCode = 5
	//ConnectionRefusedNotAuthorized Return Code
	ConnectionRefusedNotAuthorized RCode = 6
)
