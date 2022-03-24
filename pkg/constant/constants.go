package constant

const (
	LoggieOperatorLabel = "loggie-operator"
	LoggieOperatorAgentName = "loggie-operator-inject-agent"
	LoggieAgentImageName = "hub.c.163.com/loggie/loggie:v1.0.0"

	AnnotationAutoCreateKey               = "loggie.io/create"
	AnnotationCreateSidecarConfigMapValue = "configmap"

	AutoCreateConfigMapData = "pipelines.yml"
	AutoCreateSystemConfigMapData = "loggie.yml"
)