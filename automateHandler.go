package main

type AutomateHandler struct{}

func (h *AutomateHandler) Handle(labels map[string]string) bool {
	// SNMP指标直接通过,具体处理在ValidateDimensions和ProcessDimensions中
	return true
}

func (h *AutomateHandler) GetObjectId() string { return "" }

func (h *AutomateHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	return true
}

func (h *AutomateHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	return true
}
