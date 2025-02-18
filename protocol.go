package main

type ProtocolHandler interface {
	Handle(labels map[string]string) bool
	GetObjectId() string
	ValidateDimensions(dimensions map[string]interface{}) bool
	ProcessDimensions(dimensions map[string]interface{}) bool
}

var protocolHandlers = map[string]ProtocolHandler{
	K8sClusterObjectId: &K8sClusterHandler{},
	K8sPodObjectId:     &K8sPodHandler{},
	K8sNodeObjectId:    &K8sNodeHandler{},
	IPMI:               &IpmiHandler{},
	SNMP:               &SNMPHandler{},
}
