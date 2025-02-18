package main

type IpmiHandler struct{}

func (h *IpmiHandler) Handle(labels map[string]string) bool {
	metricName := labels["__name__"]
	if MetricName, exists := TelegrafIpmiMetrics[metricName]; exists {
		labels["__name__"] = MetricName
		return true
	}
	return false
}

func (h *IpmiHandler) GetObjectId() string {
	return IPMI
}

func (h *IpmiHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	return dimensions["bk_inst_name"] != nil
}

func (h *IpmiHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	dimensions["instance_name"] = dimensions["bk_inst_name"]
	return true
}
