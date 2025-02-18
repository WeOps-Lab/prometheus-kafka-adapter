package main

type SNMPHandler struct{}

func (h *SNMPHandler) Handle(labels map[string]string) bool {
	// SNMP指标直接通过,具体处理在ValidateDimensions和ProcessDimensions中
	return true
}

func (h *SNMPHandler) GetObjectId() string {
	return SNMP
}

func (h *SNMPHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	return dimensions["bk_inst_name"] != nil
}

func (h *SNMPHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	dimensions["instance_name"] = dimensions["bk_inst_name"]
	return true
}
